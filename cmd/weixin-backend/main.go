package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/walkerdu/weixin-backend/configs"
	"github.com/walkerdu/weixin-backend/internal/pkg/service"
)

var (
	usage = `Usage: %s [options] [URL...]
Options:
	--appid <wechat appid>
	--app_secret <wechat app secret>
	--token <wechat token>
	--encoding_key <wechat encoding key>
	--addr <wechat listen addr>
`
	Usage = func() {
		//fmt.Println(fmt.Sprintf("Usage of %s:\n", os.Args[0]))
		fmt.Printf(usage, os.Args[0])
	}
)

func main() {
	flag.Usage = Usage
	if len(os.Args) <= 1 {
		flag.Usage()
		os.Exit(1)
	}

	wechatConfig := &configs.WeChatConfig{}

	flag.StringVar(&wechatConfig.AppID, "appid", "", "wechat appid")
	flag.StringVar(&wechatConfig.AppSecret, "app_secret", "", "wechat app secret")
	flag.StringVar(&wechatConfig.Token, "token", "", "wechat token")
	flag.StringVar(&wechatConfig.EncodingKey, "encoding_key", "", "wechat encoding key")
	flag.StringVar(&wechatConfig.Addr, "addr", ":80", "wechat listen addr")

	flag.Parse()

	ws, err := service.NewWeChatServer(wechatConfig)
	if err != nil {
		log.Fatal("[ALERT] NewWeChatServer() failed")
	}

	log.Printf("[INFO] start Serve()")
	ws.Serve()

	// 优雅退出
	exitc := make(chan struct{})
	setupGracefulExitHook(exitc)
}

func setupGracefulExitHook(exitc chan struct{}) {
	log.Printf("[INFO] setupGracefulExitHook()")
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		sig := <-signalCh
		log.Printf("Got %s signal", sig)

		close(exitc)
	}()
}