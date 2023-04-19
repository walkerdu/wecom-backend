// internal/chatbot/chatbot.go

package chatbot

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/walkerdu/weixin-backend/pkg/openai"
)

// Chatbot 是聊天机器人结构体
type Chatbot struct {
	client *openai.Client
}

// NewChatbot 返回一个新的Chatbot实例
func NewChatbot(client *openai.Client) *Chatbot {
	return &Chatbot{
		client: client,
	}
}

// GetResponse 调用聊天机器人API获取响应
func (c *Chatbot) GetResponse(input string) (string, error) {
	// 构造请求参数
	requestBody := struct {
		Model     string `json:"model"`
		Prompt    string `json:"prompt"`
		MaxTokens int    `json:"max_tokens"`
	}{
		Model:     "davinci",
		Prompt:    input,
		MaxTokens: 50,
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// 发送HTTP请求
	resp, err := c.client.Post("/completions", requestBodyBytes)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 解析HTTP响应
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("chatbot API returned non-200 status code")
	}
	responseBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var response openai.CompletionResponse
	err = json.Unmarshal(responseBodyBytes, &response)
	if err != nil {
		return "", err
	}

	// 返回响应文本
	return strings.TrimSpace(response.Choices[0].Text), nil
}
