package openai

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

type Message struct {
	Role    RoleType `json:"role"`    // 消息的角色，可以是“system”，“user”或“assistant”
	Content string   `json:"content"` // 消息的内容
}

type Request struct {
	Model            ModelType `json:"model"`                       // 模型的名称或ID
	Messages         []Message `json:"messages"`                    // 包含多个消息的数组
	Temperature      float64   `json:"temperature,omitempty"`       // 控制生成文本的随机性。默认值为1，表示完全随机。较小的值会导致更确定的输出，而较大的值会导致更多的随机性。
	MaxTokens        int       `json:"max_tokens,omitempty"`        // 控制生成文本的长度。默认值为2048。
	TopP             float64   `json:"top_p,omitempty"`             // 控制生成文本中每个单词被选择的概率。默认值为1，表示完全随机。较小的值会导致更确定的输出，而较大的值会导致更多的随机性。如果您不想使用`Temperature`字段，则可以将其设置为0。
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty"` // 控制生成文本中重复单词出现的频率。默认值为0。
	PresencePenalty  float64   `json:"presence_penalty,omitempty"`  // 控制生成文本中缺少单词出现的频率。默认值为0。
	Stop             []string  `json:"stop,omitempty"`              // 控制生成文本的停止条件。例如，如果您希望生成一篇关于狗的文章，则可以将其设置为["\n\n"]。
}
