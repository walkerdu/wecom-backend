package configs

// 微信公众号配置
type WeChatConfig struct {
	AppID       string
	AppSecret   string
	Token       string
	EncodingKey string
	Addr        string
}

// openai配置
type OpenAIConfig struct {
	ApiKey string
}

// 企业微信配置
type WeComConfig struct {
}

type Config struct {
	Wechat WeChatConfig
	OpenAI OpenAIConfig
	Wecom  WeComConfig
}
