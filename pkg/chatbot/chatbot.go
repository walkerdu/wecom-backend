// internal/chatbot/chatbot.go

package chatbot

import (
	"container/list"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/walkerdu/wecom-backend/pkg/claude"
	openai "github.com/walkerdu/wecom-backend/pkg/openai-v1"

	"github.com/google/generative-ai-go/genai"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/option"
)

const (
	maxChatSessionCtxLength     = 6   // 聊天的最大会话长度
	maxChatResponseCahceTimeout = 120 // 聊天回包保存的最大时效2min

	ChatRoleUser = "user"
	ChatRoleAI   = "ai"

	AIName_OpenAI = "openai"
	AIName_Gemini = "gemini"
	AIName_Claude = "claude"
)

// 保存用户聊天请求的对应的回包，因为可能是异步触发返回
type chatResponseCache struct {
	content      string
	msgId        int64
	begin        int64
	asyncMsgChan chan string
	ai           string
}

// 每条消息，按userid持久化到DB
type chatMessage struct {
	Content string `json:"content"`
	Ts      int64  `json:"ts"`
	Role    string `json:"role"`
	Ai      string `json:"ai"`
}

type chatSessionCtx struct {
	//chatHistory []*openai.ChatMessage // 保证OpenAI聊天具有上下文感知能力
	chatHistory *list.List
}

// Chatbot 是聊天机器人结构体
type Chatbot struct {
	openaiClient *openai.Client
	geminiClient *genai.Client
	claudeClient *claude.Client

	redisClient *redis.Client

	publisher func(string, string) error

	chatResponseCacheMap map[string]*chatResponseCache // 用户消息处理结果的cache，用于并发限制和cache异步回包数据, 目前异步推送后会立刻清除
	rspCacheMu           sync.Mutex
	chatSessionCtxMap    map[string]*chatSessionCtx // 保存聊天的上下文
	sessionCtxMu         sync.Mutex
}

var chatbot *Chatbot

// NewChatbot 返回一个新的Chatbot实例
func NewChatbot(config *Config) *Chatbot {
	chatbot = &Chatbot{
		chatResponseCacheMap: make(map[string]*chatResponseCache),
		chatSessionCtxMap:    make(map[string]*chatSessionCtx),
	}

	if config.OpenAI.Enable {
		log.Printf("[INFO][NewChatbot] create openai client")
		chatbot.openaiClient = openai.NewClient(config.OpenAI.ApiKey)
	}

	if config.Gemini.Enable {
		log.Printf("[INFO][NewChatbot] create gemini client")
		ctx := context.Background()
		geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(config.Gemini.ApiKey))
		if err != nil {
			log.Fatal("NewChatbot| create gemini client failed")
		}

		chatbot.geminiClient = geminiClient
	}

	if config.Claude.Enable {
		log.Printf("[INFO][NewChatbot] create claude client")
		chatbot.claudeClient = claude.NewClient(config.Claude.ApiKey)
	}

	if config.Redis.Enable {
		rdb := redis.NewClient(&redis.Options{
			Addr:     config.Redis.Addr,
			Username: config.Redis.Username,
			Password: config.Redis.Password,
			DB:       config.Redis.DB,
		})

		if rdb == nil {
			log.Fatal("NewChatbot| create redis client failed, config:%+v", config.Redis)
		}

		chatbot.redisClient = rdb
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
			if content == "" {
				// 异常结束
				content = cache.content
			} else {
				// 保存聊天上下文
				// TODO: 存入DB
				c.AddChatSessionCtx(userID, content, ChatRoleAI, cache.ai)
				log.Printf("[INFO]WaitChatResponse|userID=%s wait sucess", userID)
				cache.content = content
			}

			// 消息推送
			if err := c.publisher(userID, content); err != nil {
				log.Printf("[ERROR]WaitChatResponse|publish message failed, userID=%s, err=%s", userID, err)
				return
			}

			log.Printf("[INFO]|PushTextMessage success, userID:%s", userID)
			c.clearChatCache(userID)

		case <-time.After(maxChatResponseCahceTimeout * time.Second):
			log.Printf("[WARN]WaitChatResponse|timeout, userID=%s", userID)
		}
	}()
}

func (c *Chatbot) AddChatSessionCtx(userID string, content string, role, aiName string) {
	c.sessionCtxMu.Lock()
	defer c.sessionCtxMu.Unlock()

	message := &chatMessage{
		Content: content,
		Ts:      time.Now().Unix(),
		Role:    role,
		Ai:      aiName,
	}

	if c.redisClient != nil {
		ctx := context.Background()
		key := "chatbot-" + aiName + "-" + userID

		data, err := json.Marshal(message)
		if err != nil {
			log.Printf("[ERROR][AddChatSessionCtx] json Marshal failed, err=%s", err)
			return
		}

		_, err = c.redisClient.RPush(ctx, key, data).Result()
		if err != nil {
			log.Printf("[ERROR][AddChatSessionCtx] redis RPush failed, err=%s", err)
		}

		log.Printf("[INFO][AddChatSessionCtx] redis RPush success, message=%v", string(data))
	} else {
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

		chatCtx.chatHistory.PushBack(message)
	}
}

func (c *Chatbot) GetChatMessageFromDB(userID, aiName string) []chatMessage {
	ctx := context.Background()
	key := "chatbot-" + aiName + "-" + userID

	result, err := c.redisClient.LRange(ctx, key, -maxChatSessionCtxLength, -1).Result()
	if err != nil {
		log.Printf("[ERROR][GetChatMessageFromDB] redis LRange failed, err=%s", err)
		return nil
	}

	messages := []chatMessage{}
	for _, data := range result {
		var message chatMessage
		err := json.Unmarshal([]byte(data), &message)
		if err != nil {
			log.Printf("[ERROR][GetChatMessageFromDB] json Unmarshal failed, err=%s", err)
			continue
		}

		messages = append(messages, message)
	}

	log.Printf("[INFO][GetChatMessageFromDB] redis LRange success, message=%+v", messages)

	return messages
}

func (c *Chatbot) GetChatSessionCtx(userID string, ctxs []openai.ChatMessage) []openai.ChatMessage {
	c.sessionCtxMu.Lock()
	defer c.sessionCtxMu.Unlock()

	if c.redisClient != nil {
		messages := c.GetChatMessageFromDB(userID, AIName_OpenAI)
		if messages == nil || len(messages) == 0 {
			return ctxs
		}

		for _, msg := range messages {
			role := msg.Role
			if role == ChatRoleAI {
				role = "assistant"
			}

			ctxs = append(ctxs, openai.ChatMessage{
				Content: msg.Content,
				Role:    openai.RoleType(role),
			})
		}
	} else {
		chatCtx, exist := c.chatSessionCtxMap[userID]
		if !exist {
			return ctxs
		}

		// 遍历队列中的元素
		for e := chatCtx.chatHistory.Front(); e != nil; e = e.Next() {
			msg, _ := e.Value.(*chatMessage)

			role := msg.Role
			if role == ChatRoleAI {
				role = "assistant"
			}

			ctxs = append(ctxs, openai.ChatMessage{
				Content: msg.Content,
				Role:    openai.RoleType(role),
			})
		}
	}

	return ctxs
}

func (c *Chatbot) GetClaudeChatSessionCtx(userID string, ctxs []claude.Message) []claude.Message {
	c.sessionCtxMu.Lock()
	defer c.sessionCtxMu.Unlock()

	if c.redisClient != nil {
		messages := c.GetChatMessageFromDB(userID, AIName_Claude)
		if messages == nil || len(messages) == 0 {
			return ctxs
		}

		for _, msg := range messages {
			role := msg.Role
			if role == ChatRoleAI {
				role = "assistant"
			}

			ctxs = append(ctxs, claude.Message{
				Content: msg.Content,
				Role:    role,
			})
		}
	} else {
		chatCtx, exist := c.chatSessionCtxMap[userID]
		if !exist {
			return ctxs
		}

		// 遍历队列中的元素
		for e := chatCtx.chatHistory.Front(); e != nil; e = e.Next() {
			msg, _ := e.Value.(*chatMessage)

			role := msg.Role
			if role == ChatRoleAI {
				role = "assistant"
			}
			ctxs = append(ctxs, claude.Message{
				Content: msg.Content,
				Role:    role,
			})
		}
	}

	var postCtx []claude.Message
	for _, msg := range ctxs {
		// 第一个一定要是user
		if len(postCtx) == 0 && msg.Role != "user" {
			continue
		}

		// 每个要不一样, 如果一样后面覆盖前面
		if len(postCtx) > 0 && postCtx[len(postCtx)-1].Role == msg.Role {
			postCtx[len(postCtx)-1] = msg
			continue
		}

		postCtx = append(postCtx, msg)
	}

	return postCtx
}

func (c *Chatbot) GetGeminiChatSessionCtx(userID string) []*genai.Content {
	c.sessionCtxMu.Lock()
	defer c.sessionCtxMu.Unlock()

	ctxs := []*genai.Content{}

	if c.redisClient != nil {
		messages := c.GetChatMessageFromDB(userID, AIName_Gemini)
		if messages == nil || len(messages) == 0 {
			return ctxs
		}

		for _, msg := range messages {
			role := msg.Role
			if role == ChatRoleAI {
				role = "model"
			}

			ctxs = append(ctxs, &genai.Content{
				Parts: []genai.Part{
					genai.Text(msg.Content),
				},
				Role: role,
			})
		}
	} else {
		chatCtx, exist := c.chatSessionCtxMap[userID]
		if !exist {
			return ctxs
		}

		// 遍历队列中的元素
		// gemini要求history必须是成对的，不能只有"user" 或者 "model"
		for e := chatCtx.chatHistory.Front(); e != nil; e = e.Next() {
			msg, _ := e.Value.(*chatMessage)

			role := msg.Role
			if role == ChatRoleAI {
				role = "model"
			}

			ctxs = append(ctxs, &genai.Content{
				Parts: []genai.Part{
					genai.Text(msg.Content),
				},
				Role: role,
			})
		}
	}

	roleUserCnt := 0
	for _, ctx := range ctxs {
		if ctx.Role != "model" {
			roleUserCnt++
		}

		log.Printf("[DEBUG][GetGeminiChatSessionCtx] %#v", *ctx)
	}

	// 历史错误，会导致gemini拒绝请求，400错误
	if roleUserCnt != len(ctxs)/2 {
		log.Printf("[ERROR][GetGeminiChatSessionCtx] history invalid")
		return []*genai.Content{}
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

	// 并发控制
	cache := c.buildChatCache(userID)

	if c.openaiClient != nil {
		cache.ai = AIName_OpenAI
		return c.OpenAIRequest(cache, userID, input)
	} else if c.geminiClient != nil {
		cache.ai = AIName_Gemini
		return c.GeminiRequest(cache, userID, input)
	} else if c.claudeClient != nil {
		cache.ai = AIName_Claude
		return c.ClaudeRequest(cache, userID, input)
	}

	return "no ai support", nil
}

func (c *Chatbot) OpenAIRequest(cache *chatResponseCache, userID string, input string) (string, error) {
	// OpenAI的多段对话需要先保存聊天上下文
	c.AddChatSessionCtx(userID, input, ChatRoleUser, AIName_OpenAI)

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

func (c *Chatbot) ClaudeRequest(cache *chatResponseCache, userID string, input string) (string, error) {
	// Claude的多段对话需要先保存聊天上下文
	c.AddChatSessionCtx(userID, input, ChatRoleUser, AIName_Claude)

	messages := []claude.Message{}
	messages = c.GetClaudeChatSessionCtx(userID, messages)

	req := &claude.Request{
		Model:     claude.Claude3Opus,
		Messages:  messages,
		MaxTokens: 2048,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("[ERROR][GetResponse] Marshal failed, err:%s", err)
		return "", err
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	go func() {
		// 发送HTTP请求
		err := c.claudeClient.Post(client, reqBytes, cache.asyncMsgChan)
		if err != nil {
			log.Printf("[ERROR]Claude Post failed, err:%s", err)
			cache.content = err.Error()
			close(cache.asyncMsgChan)
			return
		}
	}()

	c.WaitChatResponse(userID)

	return "Claude生成中...", nil

}

func (c *Chatbot) GeminiRequest(cache *chatResponseCache, userID string, input string) (string, error) {
	// For text-only input, use the gemini-pro model
	model := c.geminiClient.GenerativeModel("gemini-pro")
	// Initialize the chat
	cs := model.StartChat()
	cs.History = c.GetGeminiChatSessionCtx(userID)

	ctx := context.Background()

	go func() {
		resp, err := cs.SendMessage(ctx, genai.Text(input))
		if err != nil {
			log.Printf("[ERROR]|GeminiRequest:SendMessage failed, err:%v, resp:%v", err, resp)
			cache.content = err.Error()
			close(cache.asyncMsgChan)
			return
		}

		log.Printf("[INFO]|GeminiRequest: recv response::%v", resp)
		candidates := resp.Candidates
		if len(candidates) <= 0 {
			cache.content = "response candidates empty"
			close(cache.asyncMsgChan)
			return
		}

		content := candidates[0].Content
		if content == nil {
			cache.content = "response content invalid"
			close(cache.asyncMsgChan)
			return
		}

		parts := content.Parts
		if len(parts) == 0 {
			cache.content = "response parts empty"
			close(cache.asyncMsgChan)
			return
		}

		if text, ok := parts[0].(genai.Text); !ok {
			cache.content = "response parts not text"
			close(cache.asyncMsgChan)
			return
		} else {
			cache.asyncMsgChan <- string(text)
		}
	}()

	// Gemini的多段聊天需要事后保存聊天上下文, 其实它自己有History的能力，但是为了简单这里自己构造
	c.AddChatSessionCtx(userID, input, ChatRoleUser, AIName_Gemini)

	c.WaitChatResponse(userID)

	return "Gemini生成中...", nil
}
