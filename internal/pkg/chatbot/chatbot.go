// internal/chatbot/chatbot.go

package chatbot

import (
	"container/list"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/walkerdu/wecom-backend/configs"
	openai "github.com/walkerdu/wecom-backend/pkg/openai-v1"
)

const (
	maxChatSessionCtxLength     = 6   // 聊天的最大会话长度
	maxChatResponseCahceTimeout = 120 // 聊天回包保存的最大时效2min
)

// 保存用户聊天请求的对应的回包，因为可能是异步触发返回
type chatResponseCache struct {
	content      string
	msgId        int64
	begin        int64
	asyncMsgChan chan string
}

type chatSessionCtx struct {
	//chatHistory []*openai.ChatMessage // 保证OpenAI聊天具有上下文感知能力
	chatHistory *list.List
}

// Chatbot 是聊天机器人结构体
type Chatbot struct {
	openaiClient *openai.Client

	publisher func(string, string) error

	chatResponseCacheMap map[string]*chatResponseCache // 用户消息处理结果的cache，用于并发限制和cache异步回包数据, 目前异步推送后会立刻清除
	rspCacheMu           sync.Mutex
	chatSessionCtxMap    map[string]*chatSessionCtx // 保存聊天的上下文
	sessionCtxMu         sync.Mutex
}

var chatbot *Chatbot

// NewChatbot 返回一个新的Chatbot实例
func NewChatbot(config *configs.Config) *Chatbot {
	chatbot = &Chatbot{
		chatResponseCacheMap: make(map[string]*chatResponseCache),
		chatSessionCtxMap:    make(map[string]*chatSessionCtx),
	}

	if config.OpenAI.ApiKey != "" {
		log.Printf("[INFO][NewChatbot] create openai client")
		chatbot.openaiClient = openai.NewClient(config.OpenAI.ApiKey)
	}

	return chatbot
}

func MustChatbot() *Chatbot {
	return chatbot
}

// 判断会话是否进行中，同时只能并发一个会话，超时时间2min
func (c *Chatbot) isProcessing(userID string) bool {
	c.rspCacheMu.Lock()
	defer c.rspCacheMu.Unlock()

	cache, exist := c.chatResponseCacheMap[userID]
	if exist && cache.begin+maxChatResponseCahceTimeout < time.Now().Unix() {
		delete(c.chatResponseCacheMap, userID)
		return false
	}

	return exist
}

// 读取channel中异步推送的数据
func (c *Chatbot) preHitProcess(userID string, input string) (string, error) {
	c.rspCacheMu.Lock()
	defer c.rspCacheMu.Unlock()

	cache, exist := c.chatResponseCacheMap[userID]
	if !exist {
		return "", nil
	}

	return cache.content, nil
}

// 创建chatResponseCacheMap
// 用户消息处理结果的cache，用于并发限制和cache异步回包数据, 目前异步推送后会立刻清除
func (c *Chatbot) buildChatCache(userID string) *chatResponseCache {
	c.rspCacheMu.Lock()
	defer c.rspCacheMu.Unlock()

	if cache, exist := c.chatResponseCacheMap[userID]; exist {
		cache.begin = time.Now().Unix()
		return cache
	}

	cache := &chatResponseCache{
		asyncMsgChan: make(chan string, 1),
		begin:        time.Now().Unix(),
	}

	c.chatResponseCacheMap[userID] = cache

	return cache
}

// 清理chatResponseCacheMap
func (c *Chatbot) clearChatCache(userID string) {
	c.rspCacheMu.Lock()
	defer c.rspCacheMu.Unlock()

	delete(c.chatResponseCacheMap, userID)
}

// 注册聊天消息的异步推送回调
// 其实这里比较好的设计应该是调用ChatBot的调用方，在发起聊天请求中注册一下异步推送的回调，这样就可以支持不同的pusher了
func (c *Chatbot) RegsiterMessagePublish(publisher func(string, string) error) {
	c.publisher = publisher
}

func (c *Chatbot) WaitChatResponse(userID string) {
	c.rspCacheMu.Lock()

	cache, exist := c.chatResponseCacheMap[userID]
	if !exist {
		log.Printf("[ERROR]WaitChatResponse|cache not exist userID=%s", userID)
		return
	}
	c.rspCacheMu.Unlock()

	go func() {
		select {
		case content := <-cache.asyncMsgChan:
			// 保存聊天上下文
			// TODO: 存入DB
			c.AddChatSessionCtx(userID, content, false)
			log.Printf("[INFO]WaitChatResponse|userID=%s wait sucess", userID)

			// 消息推送
			if err := c.publisher(userID, content); err != nil {
				log.Printf("[ERROR]WaitChatResponse|publish message failed, userID=%s, err=%s", userID, err)
				cache.content = content
				return
			}

			log.Printf("[INFO]|PushTextMessage success, userID:%s", userID)
			c.clearChatCache(userID)

		case <-time.After(maxChatResponseCahceTimeout * time.Second):
			log.Printf("[WARN]WaitChatResponse|timeout, userID=%s", userID)
		}
	}()
}

func (c *Chatbot) AddChatSessionCtx(userID string, content string, isUser bool) {
	c.sessionCtxMu.Lock()
	defer c.sessionCtxMu.Unlock()

	var chatCtx *chatSessionCtx

	chatCtx, exist := c.chatSessionCtxMap[userID]
	if !exist {
		chatCtx = &chatSessionCtx{
			chatHistory: list.New(),
		}

		c.chatSessionCtxMap[userID] = chatCtx
	}

	if chatCtx.chatHistory.Len() >= maxChatSessionCtxLength {
		chatCtx.chatHistory.Remove(chatCtx.chatHistory.Front())
	}

	message := &openai.ChatMessage{
		Content: content,
	}
	if isUser {
		message.Role = "user"
	} else {
		message.Role = "assistant"
	}

	chatCtx.chatHistory.PushBack(message)
}

func (c *Chatbot) GetChatSessionCtx(userID string, ctxs []openai.ChatMessage) []openai.ChatMessage {
	c.sessionCtxMu.Lock()
	defer c.sessionCtxMu.Unlock()

	chatCtx, exist := c.chatSessionCtxMap[userID]
	if !exist {
		return ctxs
	}

	// 遍历队列中的元素
	for e := chatCtx.chatHistory.Front(); e != nil; e = e.Next() {
		msg, _ := e.Value.(*openai.ChatMessage)
		ctxs = append(ctxs, *msg)
	}

	return ctxs
}

// GetResponse 调用聊天机器人API获取响应
func (c *Chatbot) GetResponse(userID string, input string) (string, error) {
	// 用户指令，命中后，直接从cache中读取
	if strings.TrimSpace(input) == "继续" {
		cacheContent, _ := c.preHitProcess(userID, input)
		if cacheContent != "" {
			c.clearChatCache(userID)
			return cacheContent, nil
		} else {
			return "后台数据生成中，请稍后，生成完成会进行推送~", nil
		}
	} else if c.isProcessing(userID) {
		return "有提问在后台数据生成中，请稍后，生成完成会进行推送~", nil
	}

	// 构造请求参数
	//chatMsg := openai.ChatMessage{
	//	Role:    openai.User,
	//	Content: input,
	//}

	c.AddChatSessionCtx(userID, input, true)
	messages := []openai.ChatMessage{}
	messages = c.GetChatSessionCtx(userID, messages)

	req := &openai.ChatCompletionReq{
		Model:    openai.Gpt35Turbo,
		Messages: messages,
		User:     userID,
		Stream:   true,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("[ERROR][GetResponse] Marshal failed, err:%s", err)
		return "", err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	cache := c.buildChatCache(userID)

	// 发送HTTP请求
	rsp, err := c.openaiClient.Post(client, string(openai.OpenAIPathChatCompletion), reqBytes, cache.asyncMsgChan)
	if err != nil {
		log.Printf("[ERROR]GetResponse] Post failed, err:%s", err)
		c.clearChatCache(userID)
		return "", err
	}

	chatRsp, ok := rsp.(*openai.ChatCompletionRsp)
	if !ok {
		log.Printf("[ERROR]GetResponse] rsp invalid rsp:%v", rsp)
		c.clearChatCache(userID)
		return "", err
	}

	c.WaitChatResponse(userID)

	return chatRsp.GetContent(), nil
}
