// pkg/wechat/wechat.go

package wechat

import (
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Wechat 是微信公众号API结构体
type Wechat struct {
	appID              string
	appSecret          string
	token              string
	encodingKey        string
	msgHandlerMap      map[MessageType]MessageHandler      // 注册各个消息类型对应的逻辑处理Handler
	logicMsgHandlerMap map[MessageType]LogicMessageHandler // 注册各个消息类型对应的业务逻辑处理Handler
	concurrencyMsgMap  map[int64]struct{}                  // 按照MsgId防并发
	mu                 sync.Mutex
}

type MessageHandler func(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF)

type LogicMessageHandler func(MessageIF) (MessageIF, error)

// NewWechat 返回一个新的Wechat实例
func NewWechat(appID, appSecret, token, encodingKey string) *Wechat {
	w := &Wechat{
		appID:              appID,
		appSecret:          appSecret,
		token:              token,
		encodingKey:        encodingKey,
		msgHandlerMap:      make(map[MessageType]MessageHandler),
		logicMsgHandlerMap: make(map[MessageType]LogicMessageHandler),
		concurrencyMsgMap:  make(map[int64]struct{}),
	}

	w.registerMsgHandler()

	return w
}

func (w *Wechat) RegisterLogicMsgHandler(msgType MessageType, handler LogicMessageHandler) {
	w.logicMsgHandlerMap[msgType] = handler
}

// registerMsgHandler 注册消息的处理器
func (w *Wechat) registerMsgHandler() {
	w.msgHandlerMap[MessageTypeText] = w.handleTextMessage
	w.msgHandlerMap[MessageTypeImage] = w.handleImageMessage
	w.msgHandlerMap[MessageTypeVoice] = w.handleVoiceMessage
	w.msgHandlerMap[MessageTypeVideo] = w.handleVideoMessage
	w.msgHandlerMap[MessageTypeShortVideo] = w.handleShortVideoMessage
	w.msgHandlerMap[MessageTypeLocation] = w.handleLocationMessage
	w.msgHandlerMap[MessageTypeLink] = w.handleLinkMessage
	w.msgHandlerMap[MessageTypeEvent] = w.handleEventMessage
}

// ServeHTTP 实现http.Handler接口
func (w *Wechat) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Printf("[DEBUG]ServeHttp|recv request URL:%s, Method:%s", req.URL, req.Method)

	// 鉴权
	ret := w.handleValidationRequest(wr, req)
	if !ret {
		// 返回错误
		http.Error(wr, "Invalid signature", http.StatusBadRequest)
		return
	}

	if req.Method == http.MethodGet {
		// 处理微信公众号的验证请求, 返回echostr
		echostr := req.URL.Query().Get("echostr")
		fmt.Fprintf(wr, echostr)
	} else if req.Method == http.MethodPost {
		// 处理微信公众号的消息请求
		w.handleMessageRequest(wr, req)
	}
}

// handleValidationRequest 处理微信公众号的验证请求
func (w *Wechat) handleValidationRequest(wr http.ResponseWriter, req *http.Request) bool {
	query := req.URL.Query()
	signature := query.Get("signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")

	// 验证请求签名
	if w.validateSignature(signature, timestamp, nonce) {
		return true
	} else {
		log.Printf("[ERROR]handleValidationRequest: Invalid signature")
		return false
	}
}

// validateSignature 验证请求签名
func (w *Wechat) validateSignature(signature, timestamp, nonce string) bool {
	// 将token、timestamp、nonce三个参数进行字典序排序
	strs := []string{w.token, timestamp, nonce}
	sort.Strings(strs)

	// 将三个参数字符串拼接成一个字符串进行sha1加密
	str := strings.Join(strs, "")
	h := sha1.New()
	h.Write([]byte(str))
	sha1Sum := fmt.Sprintf("%x", h.Sum(nil))

	// 将加密后的字符串与signature进行比较
	return sha1Sum == signature
}

// handleMessageRequest 处理微信公众号的消息请求
func (w *Wechat) handleMessageRequest(wr http.ResponseWriter, req *http.Request) {
	// 读取HTTP请求体
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		err = fmt.Errorf("Failed to read request Body:%s", err)
		log.Printf("[DEBUG]%s", err)

		http.Error(wr, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG]handleMessageRequest|recv request Body:%s", body)

	// 解析XML消息
	var msg MessageReq
	err = xml.Unmarshal(body, &msg)
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
		handler(wr, req, body, &msg)
		return
	}
}

// handleTextMessage 处理文本消息
func (w *Wechat) handleTextMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 解析文本消息
	var textMsg TextMessageReq
	err := xml.Unmarshal(body, &textMsg)
	if err != nil {
		http.Error(wr, "Failed to parse text message", http.StatusBadRequest)
		return
	}

	w.mu.Lock()
	if _, exist := w.concurrencyMsgMap[textMsg.MsgId]; exist {
		err := fmt.Sprintf("message is processing now, please wait a moment, MsgId=%d, FromUserName=%s", textMsg.MsgId, textMsg.FromUserName)
		log.Printf("[ERROR][handleTextMessage]%s", err)

		http.Error(wr, err, http.StatusInternalServerError)
		return
	}

	w.concurrencyMsgMap[textMsg.MsgId] = struct{}{}
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.concurrencyMsgMap, textMsg.MsgId)
		w.mu.Unlock()
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

	log.Printf("[DEBUG]handleTextMessage|reponse:%s", xmlResponse)
	fmt.Fprintf(wr, string(xmlResponse))
}

func (w *Wechat) handleImageMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理图片消息
}

func (w *Wechat) handleVoiceMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理语音消息
}

func (w *Wechat) handleVideoMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理视频消息
}

func (w *Wechat) handleShortVideoMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理短视频消息
}

func (w *Wechat) handleLocationMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理位置消息
}

func (w *Wechat) handleLinkMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理链接消息
}

func (w *Wechat) handleEventMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理事件消息
}
