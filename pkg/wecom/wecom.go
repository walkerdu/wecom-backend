package wecom

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type WeCom struct {
	corpID              string
	agentID             int
	agentSecret         string
	agentToken          string
	agentEncodingAESKey string

	msgHandlerMap      map[MessageType]MessageHandler      // 注册各个消息类型对应的逻辑处理Handler
	logicMsgHandlerMap map[MessageType]LogicMessageHandler // 注册各个消息类型对应的业务逻辑处理Handler

	concurrencyMsgMap map[int64]struct{} // 按照MsgId防并发
	mu                sync.Mutex

	accessToken      string
	tokenExpiredTime int64

	cryptoHelper *WXBizMsgCrypt // 消息加解密工具类
}

type MessageHandler func(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF)

type LogicMessageHandler func(MessageIF) (MessageIF, error)

// NewWeCom 返回一个新的WeCom实例
func NewWeCom(config *AgentConfig) *WeCom {
	w := &WeCom{
		corpID:              config.CorpID,
		agentID:             config.AgentID,
		agentSecret:         config.AgentSecret,
		agentToken:          config.AgentToken,
		agentEncodingAESKey: config.AgentEncodingAESKey,

		msgHandlerMap:      make(map[MessageType]MessageHandler),
		logicMsgHandlerMap: make(map[MessageType]LogicMessageHandler),
		concurrencyMsgMap:  make(map[int64]struct{}),
	}

	w.cryptoHelper = NewWXBizMsgCrypt(config.AgentToken, config.AgentEncodingAESKey, config.CorpID, XmlType)

	w.registerMsgHandler()

	return w
}

func (w *WeCom) RegisterLogicMsgHandler(msgType MessageType, handler LogicMessageHandler) {
	w.logicMsgHandlerMap[msgType] = handler
}

// registerMsgHandler 注册消息的处理器
func (w *WeCom) registerMsgHandler() {
	w.msgHandlerMap[MessageTypeText] = w.handleTextMessage
	w.msgHandlerMap[MessageTypeImage] = w.handleImageMessage
	w.msgHandlerMap[MessageTypeVoice] = w.handleVoiceMessage
	w.msgHandlerMap[MessageTypeVideo] = w.handleVideoMessage
	w.msgHandlerMap[MessageTypeLocation] = w.handleLocationMessage
	w.msgHandlerMap[MessageTypeLink] = w.handleLinkMessage
	w.msgHandlerMap[MessageTypeEvent] = w.handleEventMessage
}

// ServeHTTP 实现http.Handler接口
func (w *WeCom) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
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
func (w *WeCom) handleMessageRequest(wr http.ResponseWriter, req *http.Request, msgBody []byte) {
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
func (w *WeCom) handleTextMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 解析文本消息
	var textMsg TextMessageReq
	err := xml.Unmarshal(body, &textMsg)
	if err != nil {
		http.Error(wr, "Failed to parse text message", http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG]handleTextMessage|Unmarshal message:%v", textMsg)

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

func (w *WeCom) handleImageMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理图片消息
}

func (w *WeCom) handleVoiceMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理语音消息
}

func (w *WeCom) handleVideoMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理视频消息
}

func (w *WeCom) handleShortVideoMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理短视频消息
}

func (w *WeCom) handleLocationMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理位置消息
}

func (w *WeCom) handleLinkMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理链接消息
}

func (w *WeCom) handleEventMessage(wr http.ResponseWriter, req *http.Request, body []byte, msg MessageIF) {
	// 处理事件消息
}

// 获取Access Token信息
func (w *WeCom) getAccessToken() string {
	type AccessToken struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int64  `json:"errcode,omitempty"`
		ErrMsg      string `json:"errmsg,omitempty"`
	}

	if w.tokenExpiredTime > time.Now().Unix() {
		return w.accessToken
	}

	// 请求获取 access token 的 API 地址及参数
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", w.corpID, w.agentSecret)

	// 发送 GET 请求获取 access token
	res, err := http.Get(url)
	if err != nil {
		log.Printf("[ERROR]getAccessToken|http Get failed, err:%s", err)
		return ""
	}
	defer res.Body.Close()

	// 读取返回结果中的信息
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("[ERROR]getAccessToken|ReadAll failed, err:%s", err)
		return ""
	}

	log.Printf("[INFO]getAccessToken|res Body:%s", body)

	// 将返回结果中的 JSON 数据解析到 AccessToken 结构体中
	var accessToken AccessToken
	if err := json.Unmarshal(body, &accessToken); err != nil {
		log.Printf("[ERROR]getAccessToken|json Unmarshal failed, err:%s", err)
		return ""
	}

	// 判断是否获取 access token 成功
	if accessToken.ErrCode != 0 {
		log.Printf("[ERROR]getAccessToken|Failed to get access token, errcode: %d, errmsg: %s", accessToken.ErrCode, accessToken.ErrMsg)
		return ""
	}

	w.accessToken = accessToken.AccessToken
	w.tokenExpiredTime = time.Now().Unix() + accessToken.ExpiresIn

	return w.accessToken
}

// pushMessage 推送应用消息
func (w *WeCom) pushMessage(msgBytes []byte) error {
	accessToken := w.getAccessToken()
	if accessToken == "" {
		err := errors.New("access token is invalid")
		log.Printf("[ERROR]pushMessage|getAccessToken failed, err:%s", err)
		return err
	}

	// 消息发送接口的 API 地址
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", accessToken)

	// 发送 POST 请求推送消息
	res, err := http.Post(url, "application/json", bytes.NewReader(msgBytes))
	if err != nil {
		log.Printf("[ERROR]pushMessage|http Post failed, err:%s", err)
		return err
	}
	defer res.Body.Close()

	// 读取返回结果中的信息
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("[ERROR]pushMessage|ReadAll failed, err:%s", err)
		return err
	}

	// 解析返回结果中的 JSON 数据
	var msgRsp PushMessageRsp
	if err := json.Unmarshal(body, &msgRsp); err != nil {
		log.Printf("[ERROR]pushMessage|json Unmarshal failed, err:%s", err)
		return err
	}

	// 判断是否推送消息成功
	if msgRsp.ErrCode != 0 {
		err := fmt.Errorf("pushMessage|return error, errcode: %d, errmsg: %s", msgRsp.ErrCode, msgRsp.ErrMsg)
		log.Printf("[ERROR]|:%s", err)
		return err
	}

	return nil
}

// 推送文本消息的pusher，外部可以以方法表达式的方式进行注册和调用
func (w *WeCom) PushTextMessage(userID, content string) error {
	pushMsg := &TextPushMessage{
		PushMessage: PushMessage{
			ToUser:  userID,
			MsgType: MessageTypeText,
			AgentID: w.agentID,
		},
		Text: struct {
			Content string `json:"content"` // 文本消息内容
		}{
			Content: content,
		},
	}

	// 将消息转为 JSON 格式
	msgBytes, err := json.Marshal(pushMsg)
	if err != nil {
		log.Printf("[ERROR]PushTextMessage|json Marshal failed, err:%s", err)
		return err
	}

	log.Printf("[DEBUG]|PushTextMessage|ready to push text message :%s", string(msgBytes))

	return w.pushMessage(msgBytes)
}

// 推送文本消息的pusher，外部可以以方法表达式的方式进行注册和调用
func (w *WeCom) PushMarkdowntMessage(userID, content string) error {
	pushMsg := &MarkdownPushMessage{
		PushMessage: PushMessage{
			ToUser:  userID,
			MsgType: MessageTypeMarkdown,
			AgentID: w.agentID,
		},
		Markdown: struct {
			Content string `json:"content"` // 文本消息内容
		}{
			Content: content,
		},
	}

	// 将消息转为 JSON 格式
	msgBytes, err := json.Marshal(pushMsg)
	if err != nil {
		log.Printf("[ERROR]PushMarkdowntMessage|json Marshal failed, err:%s", err)
		return err
	}

	log.Printf("[DEBUG]|PushMarkdowntMessage|ready to push markdown message :%s", string(msgBytes))

	return w.pushMessage(msgBytes)
}
