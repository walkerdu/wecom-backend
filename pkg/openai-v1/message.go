package openai

// Usage Represents the total token usage per request to OpenAI.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type MessageType string

const (
	MessageTypeTextCompletion MessageType = "text_completion"
	MessageTypeChatCompletion MessageType = "chat.completion"
)

type MessageIF interface {
	GetMessageType() MessageType
}

type Message struct {
	msgType MessageType
}

func (m *Message) GetMessageType() MessageType {
	return m.msgType
}

type OpenAIPath string

const (
	OpenAIPathCompletion         OpenAIPath = "v1/completions"
	OpenAIPathChatCompletion     OpenAIPath = "v1/chat/completions"
	OpenAIPathEdits              OpenAIPath = "v1/edits"
	OpenAIPathImage              OpenAIPath = "v1/images/generations"
	OpenAIPathImageEdits         OpenAIPath = "v1/images/edits"
	OpenAIPathImageVariation     OpenAIPath = "v1/images/variations"
	OpenAIPathEmbedding          OpenAIPath = "v1/embeddings"
	OpenAIPathAudioTranscription OpenAIPath = "v1/audio/transcriptions"
	OpenAIPathAudioTranslation   OpenAIPath = "v1/audio/translations"
)

type ModelType string

const (
	Gpt4           ModelType = "gpt-4"              // GPT-4 model name
	Gpt40314       ModelType = "gpt-4-0314"         // GPT-4 model name with March 2021 parameters
	Gpt432k        ModelType = "gpt-4-32k"          // GPT-4 model name with 32k parameters
	Gpt432k0314    ModelType = "gpt-4-32k-0314"     // GPT-4 model name with 32k parameters and March 2021 parameters
	Gpt35Turbo     ModelType = "gpt-3.5-turbo"      // ChatGPT model name (gpt-3.5-turbo) is a language model designed for conversational interfaces
	Gpt35Turbo0301 ModelType = "gpt-3.5-turbo-0301" // ChatGPT model name (gpt-3.5-turbo) with March 2021 parameters
)

type RoleType string

const (
	System    RoleType = "system"    // 系统
	User      RoleType = "user"      // 用户
	Assistant RoleType = "assistant" // 机器人助手
)

type ChatMessage struct {
	Role    RoleType `json:"role"`    // 消息的角色，可以是“system”，“user”或“assistant”
	Content string   `json:"content"` // 消息的内容
}

type ChatCompletionReq struct {
	Model            ModelType      `json:"model"`                       // 模型的名称或ID
	Messages         []ChatMessage  `json:"messages"`                    // 包含多个消息的数组
	Temperature      float64        `json:"temperature,omitempty"`       // 控制生成文本的随机性。默认值为1，表示完全随机。较小的值会导致更确定的输出，而较大的值会导致更多的随机性。
	MaxTokens        int            `json:"max_tokens,omitempty"`        // 控制生成token的长度。默认值为2048。
	TopP             float64        `json:"top_p,omitempty"`             // 控制生成文本中每个单词被选择的概率。默认值为1，表示完全随机。较小的值会导致更确定的输出，而较大的值会导致更多的随机性。如果您不想使用`Temperature`字段，则可以将其设置为0。
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"` // 控制生成文本中重复单词出现的频率。默认值为0。
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`  // 控制生成文本中缺少单词出现的频率。默认值为0。
	Stop             []string       `json:"stop,omitempty"`              // 控制生成文本的停止条件。例如，Stop 字段的值为“我”，则模型会在生成回复时在“我”这个词处停止。这个字段可以用来控制模型生成回复的长度和内容。
	N                int            `json:"n,omitempty"`                 // 控制生成回复的选项数，默认值为1，假设你想让模型生成一个关于“狗”的回复，然后在n字段中指定要生成的回复数量。例如设置为 3，则模型将生成三个关于“狗”的回复供你选择。
	Stream           bool           `json:"stream,omitempty"`            // 开启流式传输，默认为false，开启后，生成的回复将会以text/event-stream的方式多次进行推送，直到全部回复完毕。
	LogitBias        map[string]int `json:"logit_bias,omitempty"`        // 控制生成文本中, 模型输出的概率分布，取值[-100, 100],例如可以更改模型生成某些单词或标记的倾向性。例如，如果您希望模型生成更积极的回复，可以为积极词汇设置较高的偏置值。相反，如果您希望减少某些单词或短语的出现频率，可以为它们设置较低的偏置值。
	User             string         `json:"user,omitempty"`              // 用来标识终端用户ID，作用是让模型能够根据不同的用户生成不同的文本，从而提高生成文本的个性化程度。
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	DeltaMessage ChatMessage `json:"delta"` // stream方式的回包结构
	FinishReason string      `json:"finish_reason"`
}

func (c *ChatCompletionChoice) GetDeltaContent() string {
	return c.DeltaMessage.Content
}

type ChatCompletionRsp struct {
	Message
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   Usage                  `json:"usage"`
}

func (c *ChatCompletionRsp) GetContent() string {
	var content string
	for _, choice := range c.Choices {
		content += choice.Message.Content
	}

	return content
}
