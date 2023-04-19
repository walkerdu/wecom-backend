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
}

// 企业微信配置
type WeComConfig struct {
}

type Config struct {
	wechatConfig *WeChatConfig
	openaiConfig *OpenAIConfig
	wecomConfig  *WeComConfig
}
