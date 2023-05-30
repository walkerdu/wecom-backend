package configs

import (
	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

// openai配置
type OpenAIConfig struct {
	ApiKey string `json:"api_key"`
}

// 企业微信配置
type WeComConfig struct {
	AgentConfig wecom.AgentConfig `json:"agent_config"`
	Addr        string            `json:"addr"`
}

type Config struct {
	OpenAI OpenAIConfig `json:"open_ai"`
	WeCom  WeComConfig  `json:"we_com"`
}
