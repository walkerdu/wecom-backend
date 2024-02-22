package configs

import (
	"github.com/walkerdu/wecom-backend/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

// 企业微信配置
type WeComConfig struct {
	AgentConfig wecom.AgentConfig `json:"agent_config"`
	Addr        string            `json:"addr"`
}

type Config struct {
	OpenAI chatbot.OpenAIConfig `json:"open_ai"`
	WeCom  WeComConfig          `json:"we_com"`
}
