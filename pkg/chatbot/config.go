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

// claude配置
type ClaudeConfig struct {
	ApiKey string `json:"api_key"`
	Enable bool   `json:"enable"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Enable   bool   `json:"enable"`
}

type Config struct {
	OpenAI OpenAIConfig `json:"open_ai"`
	Gemini GeminiConfig `json:"gemini"`
	Claude ClaudeConfig `json:"claude"`
	Redis  RedisConfig  `json:"redis"`
}
