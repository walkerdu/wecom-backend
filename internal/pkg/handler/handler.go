// internal/handler/handler.go
// 定义了Message消息处理的基础LogicHandler的interface
// 实现了全局Handler实例，用于管理所有业务的LogicHandler
// 所有具体的业务的LogicHandler只需要注册到全局的Handler实例中就可以了

package handler

import (
	"sync"

	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

var once sync.Once
var handler *Handler

type LogicHandler interface {
	GetHandlerType() wecom.MessageType
	HandleMessage(wecom.MessageIF) (wecom.MessageIF, error)
}

// Handler 是所有HTTP处理器的基础结构体
type Handler struct {
	//middleware.AuthMiddleware
	logicHandlerMap map[wecom.MessageType]LogicHandler
}

// NewHandler 返回一个新的Handler实例
func HandlerInst() *Handler {
	once.Do(func() {
		handler = &Handler{
			logicHandlerMap: make(map[wecom.MessageType]LogicHandler),
		}
	})

	return handler
}

func (h *Handler) RegisterLogicHandler(msgType wecom.MessageType, logicHandler LogicHandler) {
	h.logicHandlerMap[msgType] = logicHandler
}

func (h *Handler) GetLogicHandlerMap() map[wecom.MessageType]LogicHandler {
	return h.logicHandlerMap
}
