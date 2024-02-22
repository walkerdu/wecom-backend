package service

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/walkerdu/wecom-backend/configs"
	"github.com/walkerdu/wecom-backend/internal/pkg/handler"
	"github.com/walkerdu/wecom-backend/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/pkg/wecom"
)

type WeComServer struct {
	httpSvr *http.Server
	wc      *wecom.WeCom
}

func NewWeComServer(config *configs.WeComConfig) (*WeComServer, error) {
	log.Printf("[INFO] NewWeComServer")

	svr := &WeComServer{}

	// 初始化微信公众号API
	svr.wc = wecom.NewWeCom(&config.AgentConfig)

	mux := http.NewServeMux()
	mux.Handle("/wecom", svr.wc)

	svr.httpSvr = &http.Server{
		Addr:    config.Addr,
		Handler: mux,
	}

	svr.InitHandler()

	// 注册聊天消息的异步推送回调
	chatbot.MustChatbot().RegsiterMessagePublish(svr.wc.PushTextMessage)

	return svr, nil
}

func (svr *WeComServer) InitHandler() error {
	for msgType, handler := range handler.HandlerInst().GetLogicHandlerMap() {
		svr.wc.RegisterLogicMsgHandler(msgType, handler.HandleMessage)
	}

	return nil
}

func (svr *WeComServer) Serve() error {
	log.Printf("[INFO] Server()")

	if err := svr.httpSvr.ListenAndServe(); nil != err {
		log.Fatalf("httpSvr ListenAndServe() failed, err=%s", err)
		return err
	}

	return nil
}

func (svr *WeComServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svr.httpSvr.Shutdown(ctx); err != nil {
		log.Printf("httpSvr ListenAndServe() failed, err=%s", err)
		return err
	}

	log.Println("[INFO]close httpSvr success")
	return nil
}
