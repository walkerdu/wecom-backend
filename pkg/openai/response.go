// pkg/openai/response.go

package openai

// CompletionResponse 是OpenAI API的/completions响应结构体
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string    `json:"text"`
		Index        int       `json:"index"`
		Logprobs     *Logprobs `json:"logprobs,omitempty"`
		FinishReason string    `json:"finish_reason"`
	} `json:"choices"`
}

// Logprobs 是OpenAI API的/completions响应中的logprobs结构体
type Logprobs struct {
	Tokens []string  `json:"tokens"`
	Probs  []float64 `json:"probs"`
}
