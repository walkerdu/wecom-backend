package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

type Client struct {
	apiKey  string
	baseURL string
}

func NewClient(apiKey string) *Client {
	client := &Client{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1/messages",
	}

	return client
}

func (c *Client) Post(httpClient *http.Client, requestBody []byte, asyncMsgChan chan string) error {
	log.Printf("[DEBUG][Post]requestBody %s", requestBody)

	// 构造HTTP请求
	url := c.baseURL
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", string(V20230601))

	// 发送HTTP请求
	if httpClient != nil {
		httpClient = &http.Client{}
	}

	httpRsp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR][Post]httpClient Do() err:%s", err)
		return err
	}

	defer httpRsp.Body.Close()

	body, err := ioutil.ReadAll(httpRsp.Body)
	if err != nil {
		log.Printf("Error reading response body:%s", err)
		return err
	}

	var resp Response
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Printf("Error reading response body:%s", err)
		return err
	}

	log.Printf("Claude response:%+v", resp)

	if resp.Error != nil {
		log.Printf("Claude response error:%s", resp.Error.Message)
		return errors.New(resp.Error.Message)
	}

	asyncMsgChan <- resp.Content[0].Text

	return nil
}
