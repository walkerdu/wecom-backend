package wechat

import (
	"encoding/xml"
)

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
type MessageReq struct {
	ToUserName   string      `xml:"ToUserName"`   // 开发者微信号
	FromUserName string      `xml:"FromUserName"` // 发送方帐号（一个OpenID）
	CreateTime   int64       `xml:"CreateTime"`   // 消息创建时间 （整型）
	MsgType      MessageType `xml:"MsgType"`      // 消息类型
}

func (m *MessageReq) GetMessageType() MessageType {
	return m.MsgType
}

// 文本消息
type TextMessageReq struct {
	MessageReq
	Content string `xml:"Content"` // 文本消息内容
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 图片消息
type ImageMessageReq struct {
	MessageReq
	PicUrl  string `xml:"PicUrl"`  // 图片链接（由系统生成）
	MediaId string `xml:"MediaId"` // 图片消息媒体id，可以调用多媒体文件下载接口拉取数据。
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 语音消息
type VoiceMessageReq struct {
	MessageReq
	MediaId     string `xml:"MediaId"`               // 语音消息媒体id，可以调用多媒体文件下载接口拉取数据。
	Format      string `xml:"Format"`                // 语音格式，如amr，speex等
	Recognition string `xml:"Recognition,omitempty"` // 开通语音识别后，会多出这个字段，表示语音识别结果
	MsgId       int64  `xml:"MsgId"`                 // 消息id，64位整型
}

// 视频消息
type VideoMessageReq struct {
	MessageReq
	MediaId      string `xml:"MediaId"`      // 视频消息媒体id，可以调用多媒体文件下载接口拉取数据。
	ThumbMediaId string `xml:"ThumbMediaId"` // 视频消息缩略图的媒体id，可以调用多媒体文件下载接口拉取数据。
	MsgId        int64  `xml:"MsgId"`        // 消息id，64位整型
}

// 小视频消息
type ShortVideoMessageReq struct {
	MessageReq
	MediaId      string `xml:"MediaId"`      // 视频消息媒体id，可以调用多媒体文件下载接口拉取数据。
	ThumbMediaId string `xml:"ThumbMediaId"` // 视频消息缩略图的媒体id，可以调用多媒体文件下载接口拉取数据。
	MsgId        int64  `xml:"MsgId"`        // 消息id，64位整型
}

// 地理位置消息
type LocationMessageReq struct {
	MessageReq
	LocationX float64 `xml:"Location_X"` // 地理位置维度
	LocationY float64 `xml:"Location_Y"` // 地理位置经度
	Scale     int     `xml:"Scale"`      // 地图缩放大小
	Label     string  `xml:"Label"`      // 地理位置信息
	MsgId     int64   `xml:"MsgId"`      // 消息id，64位整型
}

// 链接消息
type LinkMessageReq struct {
	MessageReq
	Title       string `xml:"Title"` // 消息标题
	Description string `xml:"Description"`
	Url         string `xml:"Url"`
	MsgId       int64  `xml:"MsgId"` // 消息id，64位整型
}

// 事件消息的基本结构
type EventMessage struct {
	MessageReq
	Event string `xml:"Event"` // 事件类型，subscribe(订阅)、unsubscribe(取消订阅)
}

// 微信公众号扫描带参数二维码事件消息
type ScanEventMessageReq struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket   string `xml:"Ticket"`   // 二维码的ticket，可用来换取二维码图片
}

// 微信公众号上报地理位置事件消息
type LocationEventMessageReq struct {
	EventMessage
	Latitude  float64 `xml:"Latitude"`  // 地理位置纬度
	Longitude float64 `xml:"Longitude"` // 地理位置经度
	Precision float64 `xml:"Precision"` // 地理位置精度
}

// 微信公众号点击菜单拉取消息事件消息
type ClickEventMessageReq struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，与自定义菜单接口中KEY值对应
}

// 微信公众号点击菜单跳转链接事件消息
type ViewEventMessageReq struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，设置的跳转URL
}

// 微信公众号关注事件消息
type SubscribeEventMessageReq struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket   string `xml:"Ticket"`   // 二维码的ticket，可用来换取二维码图片
}

// 微信公众号取消关注事件消息
type UnsubscribeEventMessageReq struct {
	EventMessage
}

// 微信公众号用户已关注时的事件推送消息
type ScanSubscribeEventMessageReq struct {
	EventMessage
	EventKey string `xml:"EventKey"` // 事件KEY值，qrscene_为前缀，后面为二维码的参数值
	Ticket   string `xml:"Ticket"`   // 二维码的ticket，可用来换取二维码图片
}

// -----------------------------------------
// 微信公众号所有被动回复的消息结构
// -----------------------------------------

// 如何需要将回包中的string包裹在xml的CDATA标签中，需要将成员用CDATA结构定义
type CDATA struct {
	Value string `xml:",cdata"`
}

func SToCDATA(str string) CDATA {
	return CDATA{
		Value: str,
	}
}

// 这里不和MessageReq公用一个通用的Message是考虑到CDATA序列化的限制
type MessageRsp struct {
	XMLName      xml.Name    `xml:"xml"`
	ToUserName   string      `xml:"ToUserName"`   // 开发者微信号
	FromUserName string      `xml:"FromUserName"` // 发送方帐号（一个OpenID）
	CreateTime   int64       `xml:"CreateTime"`   // 消息创建时间 （整型）
	MsgType      MessageType `xml:"MsgType"`      // 消息类型
}

func (m *MessageRsp) GetMessageType() MessageType {
	return m.MsgType
}

type TextMessageRsp struct {
	MessageRsp
	Content string `xml:"Content"`
}

type ImageMessageRsp struct {
	MessageRsp
	Image struct {
		MediaId string `xml:"MediaId"` // 通过素材管理中的接口上传多媒体文件，得到的id。
	} `xml:"Image"`
}

type VoiceMessageRsp struct {
	MessageRsp
	Voice struct {
		MediaId string `xml:"MediaId"` // 通过素材管理中的接口上传多媒体文件，得到的id。
	} `xml:"Voice"`
}

type VideoMessageRsp struct {
	MessageRsp
	Video struct {
		MediaId     string `xml:"MediaId"`     // 通过素材管理中的接口上传多媒体文件，得到的id。
		Title       string `xml:"Title"`       // 视频消息的标题
		Description string `xml:"Description"` // 视频消息的描述
	} `xml:"Video"`
}

type MusicMessageRsp struct {
	MessageRsp
	Music struct {
		Title        string `xml:"Title"`        // 音乐标题
		Description  string `xml:"Description"`  // 音乐描述
		MusicUrl     string `xml:"MusicUrl"`     // 音乐链接
		HQMusicUrl   string `xml:"HQMusicUrl"`   // 高质量音乐链接，WIFI环境优先使用该链接播放音乐
		ThumbMediaId string `xml:"ThumbMediaId"` // 缩略图的媒体id，通过素材管理中的接口上传多媒体文件，得到的id
	} `xml:"Music"`
}

type NewsMessageRsp struct {
	MessageRsp
	ArticleCount int `xml:"ArticleCount"`
	Articles     []struct {
		Title       string `xml:"Title"`
		Description string `xml:"Description"`
		PicUrl      string `xml:"PicUrl"`
		Url         string `xml:"Url"`
	} `xml:"Articles"`
}
