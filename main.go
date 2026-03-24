package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[hostel] ")

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN 환경변수를 설정해주세요.")
	}

	log.Println("🏨 선운산유스호스텔 예약 알림봇 시작")

	state := NewState()

	// Restrict to specific channels (comma-separated IDs, env overrides default)
	channelStr := os.Getenv("DISCORD_CHANNEL_IDS")
	if channelStr == "" {
		channelStr = "1485933351494750378"
	}
	channels := strings.Split(channelStr, ",")
	log.Printf("📌 허용 채널: %v", channels)

	bot, err := NewBot(token, state, channels)
	if err != nil {
		log.Fatalf("봇 생성 실패: %v", err)
	}

	if err := bot.Start(); err != nil {
		log.Fatalf("봇 시작 실패: %v", err)
	}
	defer bot.Close()

	log.Println("✅ 디스코드 봇 연결 완료 — !help 으로 사용법 확인")

	// Watcher goroutine
	ctx, cancel := context.WithCancel(context.Background())
	watcher := NewWatcher(state, bot)
	go watcher.Run(ctx)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Printf("🛑 종료 신호 수신: %v", sig)
	cancel()
	log.Println("👋 정상 종료")
}
