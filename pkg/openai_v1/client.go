// pkg/openai/client.go

package openai

import (
	"bytes"
	"errors"
	"net/http"
)

// Client 是OpenAI API客户端结构体
type Client struct {
	apiKey  string
	baseURL string
}

// NewClient 返回一个新的Client实例
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
	}
}

// Post 发送HTTP POST请求到OpenAI API
func (c *Client) Post(path string, requestBody []byte) (*http.Response, error) {
	// 构造HTTP请求
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送HTTP请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("OpenAI API returned non-200 status code")
	}

	return resp, nil
}
