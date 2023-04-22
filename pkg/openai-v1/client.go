// pkg/openai/client.go

package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// OpenAI API客户端结构体
type Client struct {
	apiKey        string
	baseURL       string
	msgHandlerMap map[OpenAIPath]MessageHandler
}

type MessageHandler func(*http.Response) (MessageIF, error)

// 创建一个新的OpenAI实例
func NewClient(apiKey string) *Client {
	client := &Client{
		apiKey:        apiKey,
		baseURL:       "https://api.openai.com",
		msgHandlerMap: make(map[OpenAIPath]MessageHandler),
	}

	client.RegisterMessageHandler()

	return client
}

func (c *Client) RegisterMessageHandler() {
	c.msgHandlerMap[OpenAIPathChatCompletion] = c.handleChatMessage
}

// Post 发送HTTP POST请求到OpenAI API
func (c *Client) Post(httpClient *http.Client, path string, requestBody []byte) (MessageIF, error) {
	// 构造HTTP请求
	url := c.baseURL + "/" + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送HTTP请求
	if httpClient != nil {
		httpClient = &http.Client{}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR][Post]httpClient Do() err:%s", err)
		return nil, err
	}

	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("OpenAI API returned non-200 status code")
	}

	rspMsg, err := c.handleMessage(path, resp)
	if err != nil {
		log.Printf("[ERROR][Post]handlerMessage err=%s", err)
		return nil, err
	}

	return rspMsg, nil
}

func (c *Client) handleMessage(path string, rsp *http.Response) (MessageIF, error) {
	if handler, ok := c.msgHandlerMap[OpenAIPath(path)]; !ok {
		err := fmt.Errorf("Unsupported message type, path=%s", path)
		return nil, err
	} else {
		return handler(rsp)
	}
}

func (c *Client) handleChatMessage(rsp *http.Response) (MessageIF, error) {
	rspBytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Printf("[ERROR][handleChatMessage]ReadAll err=%s", err)
		return nil, err
	}

	var chatRsp ChatCompletionRsp
	if err := json.Unmarshal(rspBytes, &chatRsp); nil != err {
		log.Printf("[ERROR][handleChatMessage]Unmarshal failed err=%s", err)
		return nil, err
	}

	return &chatRsp, nil
}
