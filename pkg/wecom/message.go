package wecom

import (
	"encoding/xml"
)

// MessageType 是企业消息类型
type MessageType string

const (
	MessageTypeText     MessageType = "text"     // 表示文本消息类型
	MessageTypeImage    MessageType = "image"    // 表示图片消息类型
	MessageTypeVoice    MessageType = "voice"    // 表示语音消息类型
	MessageTypeVideo    MessageType = "video"    // 表示视频消息类型
	MessageTypeFile     MessageType = "file"     // 表示文件消息类型
	MessageTypeLocation MessageType = "location" // 表示地理位置消息类型
	MessageTypeLink     MessageType = "link"     // 表示链接消息类型
	MessageTypeEvent    MessageType = "event"    // 表示事件消息类型
	MessageTypeNews     MessageType = "news"     // 表示图文消息类型
	MessageTypeMarkdown MessageType = "markdown" // 表示Markdown消息类型，目前只限推送消息
)

type MessageIF interface {
	GetMessageType() MessageType
}

// 请求消息基本结构
type MessageReq struct {
	ToUserName   string      `xml:"ToUserName"`   // 企业微信CorpID，消息接收方
	FromUserName string      `xml:"FromUserName"` // 发送方帐号（一个OpenID）
	CreateTime   int64       `xml:"CreateTime"`   // 消息创建时间（整型）
	MsgType      MessageType `xml:"MsgType"`      // 消息类型，如text、image、voice、video、location、link等
	AgentID      int         `xml:"AgentID"`      // 企业应用的id，整型。可在应用的设置页面查看
}

func (m *MessageReq) GetMessageType() MessageType {
	return m.MsgType
}

// 文本请求消息
type TextMessageReq struct {
	MessageReq
	Content string `xml:"Content"` // 文本消息内容
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 图片请求消息
type ImageMessageReq struct {
	MessageReq
	PicUrl  string `xml:"PicUrl"`  // 图片链接（由系统生成）
	MediaId string `xml:"MediaId"` // 图片媒体文件id，可以调用获取媒体文件接口拉取数据
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 语音请求消息
type VoiceMessageReq struct {
	MessageReq
	MediaId string `xml:"MediaId"` // 语音媒体文件id，可以调用获取媒体文件接口拉取数据
	Format  string `xml:"Format"`  // 语音格式，如amr、speex等
	MsgId   int64  `xml:"MsgId"`   // 消息id，64位整型
}

// 视频请求消息
type VideoMessageReq struct {
	MessageReq
	MediaId      string `xml:"MediaId"`      // 视频媒体文件id，可以调用获取媒体文件接口拉取数据
	ThumbMediaId string `xml:"ThumbMediaId"` // 视频消息缩略图的媒体id，可以调用获取媒体文件接口拉取数据
	MsgId        int64  `xml:"MsgId"`        // 消息id，64位整型
}

// 地理位置请求消息
type LocationMessageReq struct {
	MessageReq
	Location_X float64 `xml:"Location_X"` // 地理位置维度
	Location_Y float64 `xml:"Location_Y"` // 地理位置经度
	Scale      int     `xml:"Scale"`      // 地图缩放大小
	Label      string  `xml:"Label"`      // 地理位置信息
	MsgId      int64   `xml:"MsgId"`      // 消息id，64位整型
}

// 链接请求消息
type LinktMessageReq struct {
	MessageReq
	Title       string `xml:"Title"`       // 消息标题
	Description string `xml:"Description"` // 消息描述
	Url         string `xml:"Url"`         // 消息链接
	PicUrl      string `xml:"PicUrl"`      // 图片链接（由系统生成）
	MsgId       int64  `xml:"MsgId"`       // 消息id，64位整型
}

// -----------------------------------------
// 企业微信所有被动回复的消息结构
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

// 回复消息基本结构
// 这里不和MessageReq公用一个通用的Message是考虑到CDATA序列化的限制
type MessageRsp struct {
	XMLName      xml.Name    `xml:"xml"`
	ToUserName   string      `xml:"ToUserName"`   // 接收方帐号（收到的OpenID）
	FromUserName string      `xml:"FromUserName"` // 开发者微信号
	CreateTime   int64       `xml:"CreateTime"`   // 消息创建时间（整型）
	MsgType      MessageType `xml:"MsgType"`      // 消息类型，如text、image、voice、video、music、news等
}

func (m *MessageRsp) GetMessageType() MessageType {
	return m.MsgType
}

// 文本回复消息
type TextMessageRsp struct {
	MessageRsp
	Content string `xml:"Content"` // 回复的消息内容（换行：在content中能够换行，微信客户端就支持换行显示）
}

// 图片回复消息
type ImageMessageRsp struct {
	MessageRsp
	Image struct {
		MediaId string `xml:"MediaId"` // 通过素材管理中的接口上传多媒体文件，得到的id
	} `xml:"Image"`
}

// 语音回复消息
type VoiceMessageRsp struct {
	MessageRsp
	Voice struct {
		MediaId string `xml:"MediaId"` // 通过素材管理中的接口上传多媒体文件，得到的id
	} `xml:"Voice"`
}

// 视频回复消息
type VideoMessageRsp struct {
	MessageRsp
	Video struct {
		MediaId     string `xml:"MediaId"`               // 通过素材管理中的接口上传多媒体文件，得到的id
		Title       string `xml:"Title,omitempty"`       // 视频消息的标题（可选）
		Description string `xml:"Description,omitempty"` // 视频消息的描述（可选）
	} `xml:"Video"`
}

// 音乐回复消息
type MusicMessageRsp struct {
	MessageRsp
	Music struct {
		Title        string `xml:"Title,omitempty"`       // 音乐标题（可选）
		Description  string `xml:"Description,omitempty"` // 音乐描述（可选）
		MusicUrl     string `xml:"MusicUrl"`              // 音乐链接
		HQMusicUrl   string `xml:"HQMusicUrl"`            // 高质量音乐链接，WIFI环境优先使用该链接播放音乐
		ThumbMediaId string `xml:"ThumbMediaId"`          // 缩略图的媒体id，通过素材管理中的接口上传多媒体文件，得到的id
	} `xml:"Music"`
}

// 图文回复消息
type NewsMessageRsp struct {
	MessageRsp
	ArticleCount int `xml:"ArticleCount"` // 图文消息个数，限制为10条以内
	Articles     []struct {
		Title       string `xml:"Title"`       // 图文消息标题
		Description string `xml:"Description"` // 图文消息描述
		PicUrl      string `xml:"PicUrl"`      // 图片链接，支持JPG、PNG格式，较好的效果为大图640*320，小图80*80
		Url         string `xml:"Url"`         // 点击图文消息跳转链接
	} `xml:"Articles>item"`
}
