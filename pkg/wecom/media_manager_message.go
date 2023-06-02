package wecom

// 临时素材上传的回包
type UploadTemporaryMediaMessageRsp struct {
	ErrCode   int         `json:"errcode"`
	ErrMsg    string      `json:"errmsg"`
	MsgType   MessageType `json:"type,omitempty"`
	MedisId   string      `json:"media_id,omitempty"`
	CreatedAt string      `json:"created_at,omitempty"`
}
