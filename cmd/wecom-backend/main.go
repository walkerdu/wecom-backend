package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/walkerdu/wecom-backend/configs"
	"github.com/walkerdu/wecom-backend/internal/pkg/chatbot"
	"github.com/walkerdu/wecom-backend/internal/pkg/service"
)

var (
	usage = `Usage: %s [options] [URL...]
Options:
	--corp_id <wecom corpID>
	--agent_id <wecom agent id>
	--agent_secret <wecom agent secret>
	--agent_token <wecom agent token>
	--agent_encoding_aes_key <wecom agent encoding aes key>
	--addr <wecom listen addr>
	--openai_apikey <openai api key>
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

	config := &configs.Config{}

	flag.StringVar(&config.WeCom.CorpID, "corp_id", "", "wecom corporation id")
	flag.IntVar(&config.WeCom.AgentID, "agent_id", 0, "wecom agent id")
	flag.StringVar(&config.WeCom.AgentSecret, "agent_secret", "", "wecom agent secret")
	flag.StringVar(&config.WeCom.AgentToken, "agent_token", "", "wecom agent token")
	flag.StringVar(&config.WeCom.AgentEncodingAESKey, "agent_encoding_aes_key", "", "wecom agent encoding aes key")
	flag.StringVar(&config.WeCom.Addr, "addr", ":80", "wecom listen addr")

	flag.StringVar(&config.OpenAI.ApiKey, "openai_apikey", "", "openai api key")

	flag.Parse()

	log.Printf("[INFO] starup config:%v", config)

	chatbot.NewChatbot(config)

	ws, err := service.NewWeComServer(&config.WeCom)
	if err != nil {
		log.Fatal("[ALERT] NewWeComServer() failed")
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