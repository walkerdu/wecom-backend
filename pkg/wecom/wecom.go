package wecom

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Wecom struct {
	corpID              string
	agentID             string
	agentSecret         string
	agentToken          string
	agentEncodingAESKey string

	msgHandlerMap      map[MessageType]MessageHandler      // 注册各个消息类型对应的逻辑处理Handler
	logicMsgHandlerMap map[MessageType]LogicMessageHandler // 注册各个消息类型对应的业务逻辑处理Handler
	concurrencyMsgMap  map[int64]struct{}                  // 按照MsgId防并发
	mu                 sync.Mutex

	cryptoHelper *WXBizMsgCrypt // 消息加解密工具类
}

type MessageHandler func(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF)

type LogicMessageHandler func(MessageIF) (MessageIF, error)

// NewWecom 返回一个新的Wecom实例
func NewWecom(corpID, agentID, agentSecret, agentToken, agentEncodingAESKey string) *Wecom {
	w := &Wecom{
		corpID:              corpID,
		agentID:             agentID,
		agentSecret:         agentSecret,
		agentToken:          agentToken,
		agentEncodingAESKey: agentEncodingAESKey,
		msgHandlerMap:       make(map[MessageType]MessageHandler),
		logicMsgHandlerMap:  make(map[MessageType]LogicMessageHandler),
		concurrencyMsgMap:   make(map[int64]struct{}),
	}

	w.cryptoHelper = NewWXBizMsgCrypt(agentToken, agentEncodingAESKey, corpID, XmlType)

	w.registerMsgHandler()

	return w
}

func (w *Wecom) RegisterLogicMsgHandler(msgType MessageType, handler LogicMessageHandler) {
	w.logicMsgHandlerMap[msgType] = handler
}

// registerMsgHandler 注册消息的处理器
func (w *Wecom) registerMsgHandler() {
	w.msgHandlerMap[MessageTypeText] = w.handleTextMessage
	w.msgHandlerMap[MessageTypeImage] = w.handleImageMessage
	w.msgHandlerMap[MessageTypeVoice] = w.handleVoiceMessage
	w.msgHandlerMap[MessageTypeVideo] = w.handleVideoMessage
	w.msgHandlerMap[MessageTypeLocation] = w.handleLocationMessage
	w.msgHandlerMap[MessageTypeLink] = w.handleLinkMessage
	w.msgHandlerMap[MessageTypeEvent] = w.handleEventMessage
}

// ServeHTTP 实现http.Handler接口
func (w *Wecom) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Printf("[DEBUG]ServeHttp|recv request URL:%s, Method:%s", req.URL, req.Method)

	query := req.URL.Query()
	signature := query.Get("msg_signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")

	if req.Method == http.MethodGet {
		// 处理企业微信的验证请求, 返回echostr
		echostr := req.URL.Query().Get("echostr")
		msg, cryptoErr := w.cryptoHelper.VerifyURL(signature, timestamp, nonce, echostr)
		if cryptoErr != nil {
			fmt.Fprintf(wr, cryptoErr.ErrMsg)
		} else {
			fmt.Fprintf(wr, string(msg))
		}

	} else if req.Method == http.MethodPost {
		// 读取HTTP请求体
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			err = fmt.Errorf("Failed to read request Body:%s", err)
			log.Printf("[DEBUG]%s", err)

			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[DEBUG]ServeHTTP|recv request Body:%s", body)

		msg, cryptoErr := w.cryptoHelper.DecryptMsg(signature, timestamp, nonce, body)
		if cryptoErr != nil {
			fmt.Fprintf(wr, cryptoErr.ErrMsg)
			return
		}

		// 处理微信公众号的消息请求
		w.handleMessageRequest(wr, req, msg)
	}
}

// handleMessageRequest 处理微信公众号的消息请求
func (w *Wecom) handleMessageRequest(wr http.ResponseWriter, req *http.Request, msgBody []byte) {
	// 解析XML消息
	var msg MessageReq
	err := xml.Unmarshal(msgBody, &msg)
	if err != nil {
		err = fmt.Errorf("Failed to parse XML message:%s", err)
		log.Printf("[DEBUG]%s", err)

		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG]handleMessageRequest|Unmarshal message:%v", msg)

	// 处理不同类型的消息

	if handler, ok := w.msgHandlerMap[MessageType(msg.MsgType)]; !ok {
		http.Error(wr, "Unsupported message type", http.StatusBadRequest)
		return
	} else {
		handler(wr, req, msgBody, &msg)
		return
	}
}

// handleTextMessage 处理文本消息
func (w *Wecom) handleTextMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 解析文本消息
	var textMsg TextMessageReq
	err := xml.Unmarshal(body, &textMsg)
	if err != nil {
		http.Error(wr, "Failed to parse text message", http.StatusBadRequest)
		return
	}

	// 并发检测，封装在一个闭包中，保证异常锁可以正常释放
	concurrency_check_lmd := func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()

		if _, exist := w.concurrencyMsgMap[textMsg.MsgId]; exist {
			err := fmt.Sprintf("message is processing now, please wait a moment, MsgId=%d, FromUserName=%s", textMsg.MsgId, textMsg.FromUserName)
			log.Printf("[ERROR][handleTextMessage]%s", err)

			http.Error(wr, err, http.StatusInternalServerError)
			return false
		}

		w.concurrencyMsgMap[textMsg.MsgId] = struct{}{}
		return true
	}

	// 并发则返回
	if !concurrency_check_lmd() {
		return
	}

	// 保证处理完释放并发控制
	defer func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		delete(w.concurrencyMsgMap, textMsg.MsgId)
	}()

	// 调用处理器处理消息
	handler, ok := w.logicMsgHandlerMap[MessageTypeText]
	if !ok {
		http.Error(wr, "No text message handler registered", http.StatusInternalServerError)
		return
	}

	//response, err := handler((*Message)(unsafe.Pointer(&textMsg)))
	responseIF, err := handler(&textMsg)
	if err != nil {
		http.Error(wr, "Failed to handle text message", http.StatusInternalServerError)
		return
	}
	response, _ := responseIF.(*TextMessageRsp)

	// 返回响应消息
	response.ToUserName = textMsg.FromUserName
	response.FromUserName = textMsg.ToUserName
	response.CreateTime = time.Now().Unix()
	response.MsgType = MessageTypeText
	xmlResponse, err := xml.Marshal(response)
	if err != nil {
		err = fmt.Errorf("Failed to marshal XML response:%s", err)
		log.Printf("[ERROR]%s", err)

		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}

	// 构建加密消息体
	encryptMsg, cryptErr := w.cryptoHelper.EncryptMsg(string(xmlResponse), strconv.Itoa(int(response.CreateTime)), w.cryptoHelper.randString(16))
	if cryptErr != nil {
		log.Printf("[ERROR]handleTextMessage|EncryptMsg failed%s", cryptErr.ErrMsg)
		http.Error(wr, cryptErr.ErrMsg, http.StatusInternalServerError)
	}

	log.Printf("[DEBUG]handleTextMessage|reponse:%s", encryptMsg)
	fmt.Fprintf(wr, string(encryptMsg))
}

func (w *Wecom) handleImageMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理图片消息
}

func (w *Wecom) handleVoiceMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理语音消息
}

func (w *Wecom) handleVideoMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理视频消息
}

func (w *Wecom) handleShortVideoMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理短视频消息
}

func (w *Wecom) handleLocationMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理位置消息
}

func (w *Wecom) handleLinkMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理链接消息
}

func (w *Wecom) handleEventMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理事件消息
}
