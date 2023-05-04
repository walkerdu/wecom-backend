package configs

// openai配置
type OpenAIConfig struct {
	ApiKey string
}

// 企业微信配置
type WeComConfig struct {
	CorpID              string
	AgentID             int
	AgentSecret         string
	AgentToken          string
	AgentEncodingAESKey string
	Addr                string
}

type Config struct {
	OpenAI OpenAIConfig
	WeCom  WeComConfig
}
