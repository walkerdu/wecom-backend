package chatbot

// openai配置
type OpenAIConfig struct {
	ApiKey string `json:"api_key"`
	Enable bool   `json:"enable"`
}

// gemini配置
type GeminiConfig struct {
	ApiKey string `json:"api_key"`
	Enable bool   `json:"enable"`
}

type Config struct {
	OpenAI OpenAIConfig `json:"open_ai"`
	Gemini GeminiConfig `json:"gemini"`
}
