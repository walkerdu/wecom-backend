package wecom

// 企业微信一个Agent的配置
type AgentConfig struct {
	CorpID              string `json:"corp_id"`
	AgentID             int    `json:"agent_id"`
	AgentSecret         string `json:"agent_secret"`
	AgentToken          string `json:"agent_token"`
	AgentEncodingAESKey string `json:"agent_encoding_aes_key"`
}
