package claude

type ModelType string

// https://docs.anthropic.com/claude/docs/models-overview
const (
	Claude3Opus   ModelType = "claude-3-opus-20240229"
	Claude3Sonnet ModelType = "claude-3-sonnet-20240229"
	Claude3Haiku  ModelType = "claude-3-haiku-20240307"
)

type AnthropicVersion string

// https://docs.anthropic.com/claude/reference/versions
const (
	V20230601 AnthropicVersion = "2023-06-01"
	V20230101 AnthropicVersion = "2023-01-01"
)

type Message struct {
	Role    string `json:"role"`    // 消息的角色,可能是 "user" 或 "assistant"
	Content string `json:"content"` // 消息的实际内容
}

type Metadata struct {
	UserID string `json:"user_id"` // 用户的唯一标识符
}

type Request struct {
	Model         ModelType `json:"model"`                    // 要使用的 Claude 模型名称,例如 "claude-v1.3"
	MaxTokens     int       `json:"max_tokens,omitempty"`     // 响应的最大令牌数量
	Messages      []Message `json:"messages"`                 // 对话历史记录,包含用户和助手之前的消息
	Metadata      Metadata  `json:"metadata,omitempty"`       // 与用户相关的元数据
	StopSequences []string  `json:"stop_sequences,omitempty"` // 指定在遇到哪些序列时应该终止响应
	Stream        bool      `json:"stream,omitempty"`         // 是否启用流式响应模式
	Temperature   float32   `json:"temperature,omitempty"`    // 控制输出随机性的温度参数,范围 0-1
	TopP          float32   `json:"top_p,omitempty"`          // 另一个控制输出随机性的参数,范围 0-1
	TopK          int       `json:"top_k,omitempty"`          // 控制输出随机性的另一个参数
}

type Error struct {
	Message string `json:"message,omitempty"` // 错误消息
}

type Content struct {
	Text string `json:"text,omitempty"` // 响应内容文本
	Type string `json:"type,omitempty"` // 内容类型,通常为 "text"
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`  // 输入令牌数
	OutputTokens int `json:"output_tokens,omitempty"` // 输出令牌数
}

type Response struct {
	Error        *Error    `json:"error,omitempty"`         // 错误信息,如果没有错误则为 nil
	Content      []Content `json:"content,omitempty"`       // 响应内容,可能包含多个部分
	ID           string    `json:"id,omitempty"`            // 响应的唯一标识符
	Model        string    `json:"model,omitempty"`         // 使用的模型名称
	Role         string    `json:"role,omitempty"`          // 响应者的角色,通常为 "assistant"
	StopReason   string    `json:"stop_reason,omitempty"`   // 结束响应的原因,如 "end_turn"
	StopSequence *string   `json:"stop_sequence,omitempty"` // 触发结束响应的序列,如果没有则为 nil
	Type         string    `json:"type,omitempty"`          // 响应类型,通常为 "message"
	Usage        Usage     `json:"usage,omitempty"`         // 用量统计信息
}
