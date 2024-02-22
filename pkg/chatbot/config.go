package chatbot

// openai配置
type OpenAIConfig struct {
	ApiKey string `json:"api_key"`
}

type Config struct {
	OpenAI OpenAIConfig `json:"open_ai"`
}
