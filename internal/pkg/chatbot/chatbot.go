// internal/chatbot/chatbot.go

package chatbot

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/walkerdu/weixin-backend/configs"
	openai "github.com/walkerdu/weixin-backend/pkg/openai-v1"
)

type chatResponseCache struct {
	content      string
	msgId        int64
	asyncMsgChan chan string
}

// Chatbot 是聊天机器人结构体
type Chatbot struct {
	openaiClient         *openai.Client
	chatResponseCacheMap map[string]*chatResponseCache // 用户消息处理结果的cache，超过5s，就cache住, 等待用户指令进行推送
	mu                   sync.Mutex
}

var chatbot *Chatbot

// NewChatbot 返回一个新的Chatbot实例
func NewChatbot(config *configs.Config) *Chatbot {
	chatbot = &Chatbot{
		// 用户消息处理结果的cache，超过5s，就cache住, 等待用户指令进行推送
		chatResponseCacheMap: make(map[string]*chatResponseCache),
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

// 读取channel中异步推送的数据
func (c *Chatbot) preHitProcess(userID string, input string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
		log.Printf("[INFO][PreProcess] cache hit async message userID=%s", userID)
	default:
		log.Printf("[DEBUG][PreProcess] cache not hit userID=%s", userID)
	}

	defer delete(c.chatResponseCacheMap, userID)

	return cache.content, nil
}

func (c *Chatbot) buildChatCache(userID string) *chatResponseCache {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cache, exist := c.chatResponseCacheMap[userID]; exist {
		return cache
	}

	cache := &chatResponseCache{
		asyncMsgChan: make(chan string, 1),
	}

	c.chatResponseCacheMap[userID] = cache

	return cache
}

// GetResponse 调用聊天机器人API获取响应
func (c *Chatbot) GetResponse(userID string, input string) (string, error) {
	cacheContent, _ := c.preHitProcess(userID, input)

	// 用户指令，命中后，直接从cache中读取
	if strings.TrimSpace(input) == "继续" && cacheContent != "" {
		return cacheContent, nil
	}

	// 构造请求参数
	chatMsg := openai.ChatMessage{
		Role:    openai.User,
		Content: input,
	}

	req := &openai.ChatCompletionReq{
		Model:    openai.Gpt35Turbo,
		Messages: []openai.ChatMessage{chatMsg},
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
