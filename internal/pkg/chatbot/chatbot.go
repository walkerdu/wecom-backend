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

	"github.com/walkerdu/weixin-backend/configs"
	openai "github.com/walkerdu/weixin-backend/pkg/openai-v1"
)

const (
	maxChatSessionCtxLength     = 6   // 聊天的最大会话长度
	maxChatResponseCahceTimeout = 600 // 聊天回包保存的最大时效10min
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
	openaiClient         *openai.Client
	chatResponseCacheMap map[string]*chatResponseCache // 用户消息处理结果的cache，超过5s，就cache住, 等待用户指令进行推送
	chatSessionCtxMap    map[string]*chatSessionCtx    // 保存聊天的上下文
	rspCacheMu           sync.Mutex
	sessionCtxMu         sync.Mutex
}

var chatbot *Chatbot

// NewChatbot 返回一个新的Chatbot实例
func NewChatbot(config *configs.Config) *Chatbot {
	chatbot = &Chatbot{
		// 用户消息处理结果的cache，超过5s，就cache住, 等待用户指令进行推送
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

	if cache.content != "" {
		return cache.content, nil
	}

	select {
	case content := <-cache.asyncMsgChan:
		cache.content = content
		c.AddChatSessionCtx(userID, content, false)

		log.Printf("[INFO][PreProcess] cache hit async message userID=%s", userID)
		delete(c.chatResponseCacheMap, userID)
	default:
		log.Printf("[DEBUG][PreProcess] cache not hit userID=%s", userID)
	}

	return cache.content, nil
}

func (c *Chatbot) buildChatCache(userID string) *chatResponseCache {
	c.rspCacheMu.Lock()
	defer c.rspCacheMu.Unlock()

	if cache, exist := c.chatResponseCacheMap[userID]; exist {
		return cache
	}

	cache := &chatResponseCache{
		asyncMsgChan: make(chan string, 1),
		begin:        time.Now().Unix(),
	}

	c.chatResponseCacheMap[userID] = cache

	return cache
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
			return cacheContent, nil
		} else {
			return "后台数据生成中，请稍后输入: \"继续\", 获取结果", nil
		}
	} else if c.isProcessing(userID) {
		return "有提问在后台数据生成中，请稍后输入: \"继续\", 获取结果", nil
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
		return "", err
	}

	chatRsp, ok := rsp.(*openai.ChatCompletionRsp)
	if !ok {
		log.Printf("[ERROR]GetResponse] rsp invalid rsp:%v", rsp)
		return "", err
	}

	return chatRsp.GetContent(), nil
}
