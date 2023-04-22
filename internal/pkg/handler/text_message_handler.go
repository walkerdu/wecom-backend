package handler

import (
	"log"

	"github.com/walkerdu/weixin-backend/internal/pkg/chatbot"
	"github.com/walkerdu/weixin-backend/pkg/wechat"
)

func init() {
	handler := &TextMessageHandler{}
	HandlerInst().RegisterLogicHandler(wechat.MessageTypeText, handler)
}

type TextMessageHandler struct {
}

func (t *TextMessageHandler) GetHandlerType() wechat.MessageType {
	return wechat.MessageTypeText
}

func (t *TextMessageHandler) HandleMessage(msg wechat.MessageIF) (wechat.MessageIF, error) {
	textMsg := msg.(*wechat.TextMessageReq)

	chatRsp, err := chatbot.MustChatbot().GetResponse(textMsg.FromUserName, textMsg.Content)
	if err != nil {
		log.Printf("[ERROR][HandleMessage] chatbot.GetResponse failed, err=%s", err)
		chatRsp = "chatbot something wrong, please contact owner"
	}

	textMsgRsp := wechat.TextMessageRsp{
		Content: chatRsp,
	}

	return &textMsgRsp, nil
}
