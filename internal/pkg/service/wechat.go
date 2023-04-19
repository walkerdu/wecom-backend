package service

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/walkerdu/weixin-backend/configs"
	"github.com/walkerdu/weixin-backend/internal/pkg/handler"
	"github.com/walkerdu/weixin-backend/pkg/wechat"
)

type WeChatServer struct {
	httpSvr *http.Server
	wx      *wechat.Wechat
}

func NewWeChatServer(config *configs.WeChatConfig) (*WeChatServer, error) {
	log.Printf("[INFO] NewWeChatServer|WeChatConfig=%v", config)

	svr := &WeChatServer{}

	// 初始化微信公众号API
	svr.wx = wechat.NewWechat(config.AppID, config.AppSecret, config.Token, config.EncodingKey)

	mux := http.NewServeMux()
	mux.Handle("/wechat", svr.wx)

	svr.httpSvr = &http.Server{
		Addr:    config.Addr,
		Handler: mux,
	}

	svr.InitHandler()

	return svr, nil
}

func (svr *WeChatServer) InitHandler() error {
	for msgType, handler := range handler.HandlerInst().GetLogicHandlerMap() {
		svr.wx.RegisterLogicMsgHandler(msgType, handler.HandleMessage)
	}

	return nil
}

func (svr *WeChatServer) Serve() error {
	log.Printf("[INFO] Server()")

	if err := svr.httpSvr.ListenAndServe(); nil != err {
		log.Fatalf("httpSvr ListenAndServe() failed, err=%s", err)
		return err
	}

	return nil
}

func (svr *WeChatServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svr.httpSvr.Shutdown(ctx); err != nil {
		log.Printf("httpSvr ListenAndServe() failed, err=%s", err)
		return err
	}

	log.Println("[INFO]close httpSvr success")
	return nil
}
