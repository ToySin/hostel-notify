package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const cmdPrefix = "!"

// Bot handles Discord message commands.
type Bot struct {
	session    *discordgo.Session
	state      *State
	channelIDs map[string]bool // allowed channels (empty = all)
}

// NewBot creates a new Bot and registers message handlers.
// allowedChannels can be nil or empty to allow all channels.
func NewBot(token string, state *State, allowedChannels []string) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord session creation failed: %w", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	chMap := make(map[string]bool, len(allowedChannels))
	for _, id := range allowedChannels {
		if id != "" {
			chMap[id] = true
		}
	}

	bot := &Bot{session: dg, state: state, channelIDs: chMap}
	dg.AddHandler(bot.onMessage)

	// Log reconnect events
	dg.AddHandler(func(s *discordgo.Session, _ *discordgo.Resumed) {
		log.Println("[bot] 🔄 Discord 세션 재연결 완료 (Resumed)")
	})
	dg.AddHandler(func(s *discordgo.Session, d *discordgo.Disconnect) {
		log.Println("[bot] ⚠️  Discord 연결 끊김 — 자동 재연결 시도 중...")
	})
	dg.AddHandler(func(s *discordgo.Session, c *discordgo.Connect) {
		log.Println("[bot] ✅ Discord 웹소켓 연결됨")
	})

	return bot, nil
}

// Start opens the Discord connection.
func (b *Bot) Start() error {
	return b.session.Open()
}

// Close cleanly disconnects.
func (b *Bot) Close() {
	b.session.Close()
}

// Session returns the underlying discordgo session for sending messages.
func (b *Bot) Session() *discordgo.Session {
	return b.session
}

func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Channel filter: ignore messages from non-allowed channels
	if len(b.channelIDs) > 0 && !b.channelIDs[m.ChannelID] {
		return
	}
	if !strings.HasPrefix(m.Content, cmdPrefix) {
		return
	}

	content := strings.TrimPrefix(m.Content, cmdPrefix)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "watch":
		b.handleWatch(s, m, args)
	case "unwatch":
		b.handleUnwatch(s, m, args)
	case "list":
		b.handleList(s, m)
	case "check":
		b.handleCheck(s, m, args)
	case "help":
		b.handleHelp(s, m)
	}
}

func (b *Bot) handleWatch(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		b.reply(s, m, "사용법: `!watch YYYY-MM-DD [박수]`\n예: `!watch 2026-04-05` 또는 `!watch 2026-04-05 2` (2박3일)")
		return
	}

	date := args[0]
	if !isValidDate(date) {
		b.reply(s, m, "날짜 형식이 올바르지 않습니다. `YYYY-MM-DD` 형식으로 입력해주세요.")
		return
	}

	nights := 1
	if len(args) >= 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 6 {
			b.reply(s, m, "박수는 1~6 사이의 숫자로 입력해주세요.")
			return
		}
		nights = n
	}

	if !b.state.AddWatch(date, nights, m.ChannelID) {
		b.reply(s, m, fmt.Sprintf("이미 **%s** (%d박)을 감시 중입니다.", date, nights))
		return
	}

	label := fmt.Sprintf("%d박%d일", nights, nights+1)
	b.reply(s, m, fmt.Sprintf("👀 **%s** (%s) 감시를 시작합니다. 현재 상태를 조회할게요...", date, label))

	// Fetch current state immediately and show it
	rooms, err := FetchRooms(date, nights)
	if err != nil {
		b.reply(s, m, fmt.Sprintf("❌ 초기 조회 실패: %v\n감시는 계속됩니다. 변경사항이 생기면 알려드릴게요.", err))
		return
	}

	// Record initial snapshot so watcher skips its first-poll
	entry := b.state.GetEntry(date, nights)
	if entry != nil {
		b.state.ComputeDiff(entry, rooms)
	}

	// Show current availability
	var available int
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏨 **선운산유스호스텔** | %s (%s)\n\n", date, label))

	for _, r := range rooms {
		if r.Available {
			available++
		}
	}

	if available > 0 {
		sb.WriteString(fmt.Sprintf("✅ **현재 예약 가능** (%d개):\n", available))
		for _, r := range rooms {
			if r.Available {
				sb.WriteString(fmt.Sprintf("• %s\n", r.String()))
			}
		}
	} else {
		sb.WriteString("😔 현재 예약 가능한 객실이 없습니다.\n")
	}

	sb.WriteString(fmt.Sprintf("\n📊 전체 %d개 | 예약가능 %d | 예약완료 %d\n", len(rooms), available, len(rooms)-available))
	sb.WriteString("변경사항이 생기면 알려드릴게요.")

	b.reply(s, m, sb.String())
}

func (b *Bot) handleUnwatch(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		b.reply(s, m, "사용법: `!unwatch YYYY-MM-DD [박수]`")
		return
	}

	date := args[0]
	nights := 1
	if len(args) >= 2 {
		n, _ := strconv.Atoi(args[1])
		if n >= 1 {
			nights = n
		}
	}

	if b.state.RemoveWatch(date, nights) {
		b.reply(s, m, fmt.Sprintf("🔕 **%s** (%d박) 감시를 해제했습니다.", date, nights))
	} else {
		b.reply(s, m, fmt.Sprintf("**%s** (%d박)은 감시 목록에 없습니다.", date, nights))
	}
}

func (b *Bot) handleList(s *discordgo.Session, m *discordgo.MessageCreate) {
	watches := b.state.ListWatches()
	if len(watches) == 0 {
		b.reply(s, m, "현재 감시 중인 날짜가 없습니다. `!watch YYYY-MM-DD`로 추가하세요.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 **감시 목록**\n")
	for _, w := range watches {
		status := "🔍 폴링 중"
		if w.FirstPoll {
			status = "⏳ 대기 중"
		}
		availCount := len(w.PrevAvailable)
		sb.WriteString(fmt.Sprintf("• **%s** (%s) — %s (예약가능 %d개)\n",
			w.Date, w.NightsLabel(), status, availCount))
	}
	b.reply(s, m, sb.String())
}

func (b *Bot) handleCheck(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		b.reply(s, m, "사용법: `!check YYYY-MM-DD [박수]`")
		return
	}

	date := args[0]
	if !isValidDate(date) {
		b.reply(s, m, "날짜 형식이 올바르지 않습니다. `YYYY-MM-DD` 형식으로 입력해주세요.")
		return
	}

	nights := 1
	if len(args) >= 2 {
		n, err := strconv.Atoi(args[1])
		if err == nil && n >= 1 && n <= 6 {
			nights = n
		}
	}

	b.reply(s, m, fmt.Sprintf("🔍 **%s** (%d박%d일) 조회 중...", date, nights, nights+1))

	rooms, err := FetchRooms(date, nights)
	if err != nil {
		b.reply(s, m, fmt.Sprintf("❌ 조회 실패: %v", err))
		return
	}

	if len(rooms) == 0 {
		b.reply(s, m, "조회된 객실이 없습니다.")
		return
	}

	var available, booked int
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏨 **선운산유스호스텔** | %s (%d박%d일)\n\n", date, nights, nights+1))

	var availRooms []Room
	for _, r := range rooms {
		if r.Available {
			available++
			availRooms = append(availRooms, r)
		} else {
			booked++
		}
	}

	if available > 0 {
		sb.WriteString(fmt.Sprintf("✅ **예약 가능** (%d개):\n", available))
		for _, r := range availRooms {
			sb.WriteString(fmt.Sprintf("• %s\n", r.String()))
		}
	} else {
		sb.WriteString("😔 예약 가능한 객실이 없습니다.\n")
	}

	sb.WriteString(fmt.Sprintf("\n📊 전체 %d개 | 예약가능 %d | 예약완료 %d\n", len(rooms), available, booked))
	sb.WriteString(fmt.Sprintf("🔗 %s", BuildReservationURL(date, nights)))

	b.reply(s, m, sb.String())
}

func (b *Bot) handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	help := `📖 **선운산유스호스텔 예약 알림봇**

• ` + "`!watch YYYY-MM-DD [박수]`" + ` — 감시 시작 (기본 1박)
• ` + "`!unwatch YYYY-MM-DD [박수]`" + ` — 감시 해제
• ` + "`!list`" + ` — 감시 목록 조회
• ` + "`!check YYYY-MM-DD [박수]`" + ` — 즉시 1회 조회
• ` + "`!help`" + ` — 이 도움말

예시:
` + "```" + `
!watch 2026-04-05       # 4/5 1박2일 감시
!watch 2026-04-05 2     # 4/5 2박3일 감시
!check 2026-04-01       # 4/1 즉시 조회
!unwatch 2026-04-05     # 감시 해제
` + "```"
	b.reply(s, m, help)
}

func (b *Bot) reply(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	_, err := s.ChannelMessageSend(m.ChannelID, content)
	if err != nil {
		log.Printf("[bot] message send failed: %v", err)
	}
}

// SendToChannel sends a message to a specific channel.
func (b *Bot) SendToChannel(channelID, content string) {
	// Discord message limit is 2000 chars
	if len(content) > 1990 {
		content = content[:1990] + "..."
	}
	_, err := b.session.ChannelMessageSend(channelID, content)
	if err != nil {
		log.Printf("[bot] message send failed: %v", err)
	}
}

func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
