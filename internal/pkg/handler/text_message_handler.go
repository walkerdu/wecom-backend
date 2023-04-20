package handler

import (
	"github.com/walkerdu/weixin-backend/pkg/wechat"
)

func init() {
	handler := &TextMessageHandler{}
	HandlerInst().RegisterLogicHandler(wechat.MessageTypeText, handler)
}

type TextMessageHandler struct {
	//chatbot        *chatbot.Chatbot
}

func (t *TextMessageHandler) GetHandlerType() wechat.MessageType {
	return wechat.MessageTypeText
}

func (t *TextMessageHandler) HandleMessage(msg wechat.MessageIF) (wechat.MessageIF, error) {
	textMsg := msg.(*wechat.TextMessage)

	textMsgRsp := wechat.TextMessageResponse{
		Content: textMsg.Content + " response",
	}

	return &textMsgRsp, nil
}
