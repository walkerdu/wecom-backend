// internal/handler/handler.go
// 定义了Message消息处理的基础LogicHandler的interface
// 实现了全局Handler实例，用于管理所有业务的LogicHandler
// 所有具体的业务的LogicHandler只需要注册到全局的Handler实例中就可以了

package handler

import (
	"sync"

	"github.com/walkerdu/weixin-backend/pkg/wechat"
)

var once sync.Once
var handler *Handler

type LogicHandler interface {
	GetHandlerType() wechat.MessageType
	HandleMessage(wechat.MessageIF) (wechat.MessageIF, error)
}

// Handler 是所有HTTP处理器的基础结构体
type Handler struct {
	//middleware.AuthMiddleware
	logicHandlerMap map[wechat.MessageType]LogicHandler
}

// NewHandler 返回一个新的Handler实例
func HandlerInst() *Handler {
	once.Do(func() {
		handler = &Handler{
			logicHandlerMap: make(map[wechat.MessageType]LogicHandler),
		}
	})

	return handler
}

func (h *Handler) RegisterLogicHandler(msgType wechat.MessageType, logicHandler LogicHandler) {
	h.logicHandlerMap[msgType] = logicHandler
}

func (h *Handler) GetLogicHandlerMap() map[wechat.MessageType]LogicHandler {
	return h.logicHandlerMap
}
