package wechat

// MessageType 是公众号消息类型
type MessageType string

const (
	MessageTypeText       MessageType = "text"       // 表示文本消息类型
	MessageTypeImage      MessageType = "image"      // 表示图片消息类型
	MessageTypeVoice      MessageType = "voice"      // 表示语音消息类型
	MessageTypeVideo      MessageType = "video"      // 表示视频消息类型
	MessageTypeShortVideo MessageType = "shortvideo" // 表示短视频消息类型
	MessageTypeLocation   MessageType = "location"   // 表示地理位置消息类型
	MessageTypeLink       MessageType = "link"       // 表示链接消息类型
	MessageTypeEvent      MessageType = "event"      // 表示事件消息类型
)

type MessageIF interface {
	GetMessageType() MessageType
}

// 公众号消息的通用结构
type Message struct {
	ToUserName   string      `xml:"ToUserName"`   // 开发者微信号
	FromUserName string      `xml:"FromUserName"` // 发送方帐号（一个OpenID）
	CreateTime   int64       `xml:"CreateTime"`   // 消息创建时间 （整型）
	MsgType      MessageType `xml:"MsgType"`      // 消息类型
}

func (m *Message) GetMessageType() MessageType {
	return m.MsgType
}

// 文本消息
type TextMessage struct {
	Message
	Content string `xml:"Content"` // 文本消息内容
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 图片消息
type ImageMessage struct {
	Message
	PicUrl  string `xml:"PicUrl"`  // 图片链接（由系统生成）
	MediaId string `xml:"MediaId"` // 图片消息媒体id，可以调用多媒体文件下载接口拉取数据。
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 语音消息
type VoiceMessage struct {
	Message
	MediaId     string `xml:"MediaId"`               // 语音消息媒体id，可以调用多媒体文件下载接口拉取数据。
	Format      string `xml:"Format"`                // 语音格式，如amr，speex等
	Recognition string `xml:"Recognition,omitempty"` // 开通语音识别后，会多出这个字段，表示语音识别结果
	MsgId       int64  `xml:"MsgId"`                 // 消息id，64位整型
}

// 视频消息
type VideoMessage struct {
	Message
	MediaId      string `xml:"MediaId"`      // 视频消息媒体id，可以调用多媒体文件下载接口拉取数据。
	ThumbMediaId string `xml:"ThumbMediaId"` // 视频消息缩略图的媒体id，可以调用多媒体文件下载接口拉取数据。
	MsgId        int64  `xml:"MsgId"`        // 消息id，64位整型
}

// 小视频消息
type ShortVideoMessage struct {
	Message
	MediaId      string `xml:"MediaId"`      // 视频消息媒体id，可以调用多媒体文件下载接口拉取数据。
	ThumbMediaId string `xml:"ThumbMediaId"` // 视频消息缩略图的媒体id，可以调用多媒体文件下载接口拉取数据。
	MsgId        int64  `xml:"MsgId"`        // 消息id，64位整型
}

// 地理位置消息
type LocationMessage struct {
	Message
	LocationX float64 `xml:"Location_X"` // 地理位置维度
	LocationY float64 `xml:"Location_Y"` // 地理位置经度
	Scale     int     `xml:"Scale"`      // 地图缩放大小
	Label     string  `xml:"Label"`      // 地理位置信息
	MsgId     int64   `xml:"MsgId"`      // 消息id，64位整型
}

// 链接消息
type LinkMessage struct {
	Message
	Title       string `xml:"Title"` // 消息标题
	Description string `xml:"Description"`
	Url         string `xml:"Url"`
	MsgId       int64  `xml:"MsgId"` // 消息id，64位整型
}

// 事件消息的基本结构
type EventMessage struct {
	Message
	Event string `xml:"Event"` // 事件类型，subscribe(订阅)、unsubscribe(取消订阅)
}

// 微信公众号扫描带参数二维码事件消息
type ScanEventMessage struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket   string `xml:"Ticket"`   // 二维码的ticket，可用来换取二维码图片
}

// 微信公众号上报地理位置事件消息
type LocationEventMessage struct {
	EventMessage
	Latitude  float64 `xml:"Latitude"`  // 地理位置纬度
	Longitude float64 `xml:"Longitude"` // 地理位置经度
	Precision float64 `xml:"Precision"` // 地理位置精度
}

// 微信公众号点击菜单拉取消息事件消息
type ClickEventMessage struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，与自定义菜单接口中KEY值对应
}

// 微信公众号点击菜单跳转链接事件消息
type ViewEventMessage struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，设置的跳转URL
}

// 微信公众号关注事件消息
type SubscribeEventMessage struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket   string `xml:"Ticket"`   // 二维码的ticket，可用来换取二维码图片
}

// 微信公众号取消关注事件消息
type UnsubscribeEventMessage struct {
	EventMessage
}

// 微信公众号用户已关注时的事件推送消息
type ScanSubscribeEventMessage struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket   string `xml:"Ticket"`   // 二维码的ticket，可用来换取二维码图片
}

// -----------------------------------------
// 微信公众号所有被动回复的消息结构
// -----------------------------------------

type TextMessageResponse struct {
	Message
	Content string `xml:"Content"`
}

type ImageMessageResponse struct {
	Message
	Image struct {
		MediaId string `xml:"MediaId"` // 通过素材管理中的接口上传多媒体文件，得到的id。
	} `xml:"Image"`
}

type VoiceMessageResponse struct {
	Message
	Voice struct {
		MediaId string `xml:"MediaId"` // 通过素材管理中的接口上传多媒体文件，得到的id。
	} `xml:"Voice"`
}

type VideoMessageResponse struct {
	Message
	Video struct {
		MediaId     string `xml:"MediaId"`     // 通过素材管理中的接口上传多媒体文件，得到的id。
		Title       string `xml:"Title"`       // 视频消息的标题
		Description string `xml:"Description"` // 视频消息的描述
	} `xml:"Video"`
}

type MusicMessageResponse struct {
	Message
	Music struct {
		Title        string `xml:"Title"`        // 音乐标题
		Description  string `xml:"Description"`  // 音乐描述
		MusicUrl     string `xml:"MusicUrl"`     // 音乐链接
		HQMusicUrl   string `xml:"HQMusicUrl"`   // 高质量音乐链接，WIFI环境优先使用该链接播放音乐
		ThumbMediaId string `xml:"ThumbMediaId"` // 缩略图的媒体id，通过素材管理中的接口上传多媒体文件，得到的id
	} `xml:"Music"`
}

type NewsMessageResponse struct {
	Message
	ArticleCount int `xml:"ArticleCount"`
	Articles     []struct {
		Title       string `xml:"Title"`
		Description string `xml:"Description"`
		PicUrl      string `xml:"PicUrl"`
		Url         string `xml:"Url"`
	} `xml:"Articles"`
}
