package configs

// openai配置
type OpenAIConfig struct {
	ApiKey string
}

// 企业微信配置
type WeComConfig struct {
	CorpID              string
	AgentID             string
	AgentSecret         string
	AgentToken          string
	AgentEncodingAESKey string
	Addr                string
}

type Config struct {
	OpenAI OpenAIConfig
	Wecom  WeComConfig
}
