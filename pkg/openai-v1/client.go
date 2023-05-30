// pkg/openai/client.go

package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const MaxStreamWaitTimeSecs = 60

// OpenAI API客户端结构体
type Client struct {
	apiKey        string
	baseURL       string
	msgHandlerMap map[OpenAIPath]MessageHandler
}

type MessageHandler func(*http.Response, chan string) (MessageIF, error)

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
func (c *Client) Post(httpClient *http.Client, path string, requestBody []byte, asyncMsgChan chan string) (MessageIF, error) {
	log.Printf("[DEBUG][Post]requestBody %s", requestBody)

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

	// 异步等待OpenAI后端推流,  需要延后关闭
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		defer resp.Body.Close()
	}

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned %d status code", resp.StatusCode)
	}

	rspMsg, err := c.handleMessage(path, resp, asyncMsgChan)
	if err != nil {
		log.Printf("[ERROR][Post]handlerMessage err=%s", err)
		return nil, err
	}

	return rspMsg, nil
}

func (c *Client) handleMessage(path string, rsp *http.Response, asyncMsgChan chan string) (MessageIF, error) {
	if handler, ok := c.msgHandlerMap[OpenAIPath(path)]; !ok {
		err := fmt.Errorf("Unsupported message type, path=%s", path)
		return nil, err
	} else {
		return handler(rsp, asyncMsgChan)
	}
}

func (c *Client) handleChatMessage(rsp *http.Response, asyncMsgChan chan string) (MessageIF, error) {
	log.Printf("[DEBUG][handleChatMessage] rsp Header:%v", rsp.Header)

	var chatRsp ChatCompletionRsp

	//Content-Type:[text/event-stream]
	if rsp.Header.Get("Content-Type") == "text/event-stream" {
		choice := ChatCompletionChoice{
			Message: ChatMessage{
				Content: "OpenAI数据生成中，请稍后， 生成完成会进行推送~ \n也可输入:\"继续\"，获取结果~",
			},
		}

		chatRsp.Choices = []ChatCompletionChoice{}
		chatRsp.Choices = append(chatRsp.Choices, choice)

		streamReader := streamReader{
			reader:   bufio.NewReader(rsp.Body),
			response: &ChatCompletionRsp{},
		}

		// 异步等待OpenAI后端推流
		go func() {
			defer rsp.Body.Close()

			begin := time.Now().Unix()
			var asyncStream string

			log.Printf("[INFO][handleChatMessage] Ready to streamReader.Recv()")
			firstRecv := false

			for !streamReader.isFinished {
				if err := streamReader.Recv(); err != nil && err != io.EOF {
					log.Printf("[ERROR][handleChatMessage] streamReader.Recv() error:%s", err)
					break
				}

				// 收到一条推流
				for _, choice := range streamReader.response.Choices {
					asyncStream += choice.GetDeltaContent()
					if !firstRecv {
						firstRecv = true
						log.Printf("[INFO][handleChatMessage] streamReader.Recv() first response, choice:%v", choice)
					}
				}

				now := time.Now().Unix()
				if now-begin >= MaxStreamWaitTimeSecs {
					log.Printf("[ERROR][handleChatMessage] streamReader.Recv() timeout")
					break
				}
			}

			if streamReader.isFinished {
				log.Printf("[INFO][handleChatMessage] streamReader.Recv() finish, full message:%v", asyncStream)
			}

			select {
			case asyncMsgChan <- asyncStream:
				log.Printf("[INFO][handleChatMessage] push stream into recv chan")
			default:
				log.Printf("[ERROR][handleChatMessage] push stream into recv chan failed")
			}
		}()

		return &chatRsp, nil
	}

	rspBytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Printf("[ERROR][handleChatMessage]ReadAll err=%s", err)
		return nil, err
	}

	log.Printf("[DEBUG][handleChatMessage] rsp Body:%s", rspBytes)

	if err := json.Unmarshal(rspBytes, &chatRsp); nil != err {
		log.Printf("[ERROR][handleChatMessage]Unmarshal failed err=%s", err)
		return nil, err
	}

	return &chatRsp, nil
}
