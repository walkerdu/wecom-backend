package wecom

// 推送应用消息基本结构
type PushMessage struct {
	Touser                 string      `json:"touser"`                             // 指定接收消息的成员，成员ID列表（消息接收者，多个接收者用‘|’分隔，最多支持1000个）。特殊情况：指定为@all，则向关注该企业应用的全部成员发送
	Toparty                string      `json:"toparty,omitempty"`                  // 指定接收消息的部门，部门ID列表，多个接收者用‘|’分隔，最多支持100个。当touser为@all时忽略本参数
	Totag                  string      `json:"totag,omitempty"`                    // 指定接收消息的标签，标签ID列表，多个接收者用‘|’分隔，最多支持100个。当touser为@all时忽略本参数
	Msgtype                MessageType `json:"msgtype"`                            // 消息类型，如text、image、voice、video、file、textcard、news、mpnews等
	Agentid                int         `json:"agentid"`                            // 企业应用的id，整型。可在应用的设置页面查看
	Safe                   int         `json:"safe,omitempty"`                     // 表示是否是保密消息，0表示否，1表示是，默认0
	EnableIdTrans          int         `json:"enable_id_trans,omitempty"`          // 表示是否开启id转译，0表示否，1表示是，默认0
	EnableDuplicateCheck   int         `json:"enable_duplicate_check,omitempty"`   // 表示是否开启重复消息检查，0表示否，1表示是，默认0
	DuplicateCheckInterval int         `json:"duplicate_check_interval,omitempty"` // 重复消息检查的时间间隔，默认1800s，最大不超过4小时
}

// 推送应用消息的通用回包结构
type PushMessageRsp struct {
	ErrCode        int    `json:"errcode"`
	ErrMsg         string `json:"errmsg"`
	InvalidUser    string `json:"invaliduser,omitempty"`    //如果部分接收人无权限或不存在，发送仍然执行，但会返回无效的部分, 格式"userid1|userid2"
	InvalidParty   string `json:"invalidparty,omitempty"`   //如果部分接收人无权限或不存在，发送仍然执行，但会返回无效的部分, 格式"partyid1|partyid2"
	InvalidTag     string `json:"invalidtag,omitempty"`     //如果部分接收人无权限或不存在，发送仍然执行，但会返回无效的部分, 格式"tagid1|tagid2"
	UnLicensedUser string `json:"unlicenseduser,omitempty"` //如果部分接收人无权限或不存在，发送仍然执行，但会返回无效的部分, 格式"userid1|userid2"
	MsgId          string `json:"msgid"`
	ResponseCode   string `json:"response_code"`
}

// 文本消息
type TextPushMessage struct {
	PushMessage
	Text struct {
		Content string `json:"content"` // 文本消息内容
	} `json:"text"`
}

// 图片消息
type ImagePushMessage struct {
	PushMessage
	Image struct {
		MediaId string `json:"media_id"` // 图片媒体文件id，可以调用上传临时素材接口获取
	} `json:"image"`
}

// 语音消息
type VoicePushMessage struct {
	PushMessage
	Voice struct {
		MediaId string `json:"media_id"` // 语音媒体文件id，可以调用上传临时素材接口获取
	} `json:"voice"`
}

// 视频消息
type VideoPushMessage struct {
	PushMessage
	Video struct {
		MediaId     string `json:"media_id"`    // 视频媒体文件id，可以调用上传临时素材接口获取
		Title       string `json:"title"`       // 视频消息的标题（可选）
		Description string `json:"description"` // 视频消息的描述（可选）
	} `json:"video"`
}

// 文件消息
type FilePushMessage struct {
	PushMessage
	File struct {
		MediaId string `json:"media_id"` // 文件媒体文件id，可以调用上传临时素材接口获取
	} `json:"file"`
}

// 文本卡片消息
type TextCardPushMessage struct {
	PushMessage
	TextCard struct {
		Title       string `json:"title"`       // 标题，不超过128个字节，超过会自动截断
		Description string `json:"description"` // 描述，不超过512个字节，超过会自动截断
		Url         string `json:"url"`         // 点击后跳转的链接
		Btntxt      string `json:"btntxt"`      // 按钮文字。 默认为“详情”， 不超过4个文字，超过自动截断
	} `json:"textcard"`
}

// 图文消息
type NewsPushMessage struct {
	PushMessage
	News struct {
		Articles []struct {
			Title       string `json:"title"`       // 标题，不超过128个字节，超过会自动截断
			Description string `json:"description"` // 描述，不超过512个字节，超过会自动截断
			Url         string `json:"url"`         // 点击后跳转的链接
			Picurl      string `json:"picurl"`      // 图文消息的图片链接，支持JPG、PNG格式，较好的效果为大图640*320，小图80*80
		} `json:"articles"`
	} `json:"news"`
}

// 图文消息（mpnews）
type MpNewsPushMessage struct {
	PushMessage
	MpNews struct {
		Articles []struct {
			Title            string `json:"title"`              // 标题，不超过128个字节，超过会自动截断
			ThumbMediaId     string `json:"thumb_media_id"`     // 缩略图的媒体ID，可以通过素材管理接口获得
			Author           string `json:"author"`             // 作者，不超过64个字节，超过会自动截断
			ContentSourceUrl string `json:"content_source_url"` // 图文消息点击“阅读原文”之后的页面链接
			Content          string `json:"content"`            // 图文消息的内容，支持html标签，不超过666 K个字节
			Digest           string `json:"digest"`             // 图文消息的描述，不超过512个字节，超过会自动截断
		} `json:"articles"`
	} `json:"mpnews"`
}

// Markdown消息
type MarkdownPushMessage struct {
	PushMessage
	Markdown struct {
		Content string `json:"content"` // Markdown内容，最长不超过4096个字节，必须是utf8编码
	} `json:"markdown"`
}

// 小程序通知消息
type MiniProgramNoticePushMessage struct {
	PushMessage
	MiniProgramNotice struct {
		AppId             string `json:"appid"`               // 小程序appid，必须是关联到企业的小程序应用
		Page              string `json:"page"`                // 点击消息卡片后进入的小程序页面路径
		Title             string `json:"title"`               // 消息标题，长度限制4-12个汉字
		Description       string `json:"description"`         // 消息描述，长度限制4-12个汉字
		EmphasisFirstItem bool   `json:"emphasis_first_item"` // 是否放大第一个content_item
		ContentItems      []struct {
			Key   string `json:"key"`   // 信息名称，长度限制4-20个汉字
			Value string `json:"value"` // 信息值，长度限制4-20个汉字
		} `json:"content_item"`
	} `json:"miniprogram_notice"`
}
