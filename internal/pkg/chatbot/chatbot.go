// internal/chatbot/chatbot.go

package chatbot

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/walkerdu/weixin-backend/configs"
	openai "github.com/walkerdu/weixin-backend/pkg/openai-v1"
)

// Chatbot 是聊天机器人结构体
type Chatbot struct {
	openaiClient *openai.Client
}

var chatbot *Chatbot

// NewChatbot 返回一个新的Chatbot实例
func NewChatbot(config *configs.Config) *Chatbot {
	chatbot = &Chatbot{}

	if config.OpenAI.ApiKey != "" {
		log.Printf("[INFO][NewChatbot] create openai client")
		chatbot.openaiClient = openai.NewClient(config.OpenAI.ApiKey)
	}

	return chatbot
}

func MustChatbot() *Chatbot {
	return chatbot
}

// GetResponse 调用聊天机器人API获取响应
func (c *Chatbot) GetResponse(userID string, input string) (string, error) {
	// 构造请求参数
	chatMsg := openai.ChatMessage{
		Role:    openai.User,
		Content: input,
	}

	req := &openai.ChatCompletionReq{
		Model:    openai.Gpt35Turbo,
		Messages: []openai.ChatMessage{chatMsg},
		User:     userID,
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
	rsp, err := c.openaiClient.Post(client, string(openai.OpenAIPathChatCompletion), reqBytes)
	if err != nil {
		log.Printf("[ERROR]GetResponse] Post failed, err:%s", err)
		return "", err
	}

	chatRsp, ok := rsp.(*openai.ChatCompletionRsp)
	if !ok {
		log.Printf("[ERROR]GetResponse] rsp invalid rsp:%v", rsp)
		return "", err
	}

	var rspContent string
	for _, choice := range chatRsp.Choices {
		rspContent += choice.Message.Content
	}

	return rspContent, nil
}
