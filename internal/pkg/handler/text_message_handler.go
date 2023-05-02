package handler

import (
	"log"

	"github.com/walkerdu/wecom-backend/internal/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

const WeChatTimeOutSecs = 5

func init() {
	handler := &TextMessageHandler{}

	HandlerInst().RegisterLogicHandler(wecom.MessageTypeText, handler)
}

type TextMessageHandler struct {
}

func (t *TextMessageHandler) GetHandlerType() wecom.MessageType {
	return wecom.MessageTypeText
}

func (t *TextMessageHandler) HandleMessage(msg wecom.MessageIF) (wecom.MessageIF, error) {
	textMsg := msg.(*wecom.TextMessageReq)

	chatRsp, err := chatbot.MustChatbot().GetResponse(textMsg.FromUserName, textMsg.Content)
	if err != nil {
		log.Printf("[ERROR][HandleMessage] chatbot.GetResponse failed, err=%s", err)
		chatRsp = "chatbot something wrong, please contact owner"
	}

	textMsgRsp := wecom.TextMessageRsp{
		Content: chatRsp,
	}

	return &textMsgRsp, nil
}
