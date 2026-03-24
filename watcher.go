package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

const basePollInterval = 60 * time.Second

// Watcher periodically checks watched dates for availability changes.
type Watcher struct {
	state *State
	bot   *Bot
}

// NewWatcher creates a new Watcher.
func NewWatcher(state *State, bot *Bot) *Watcher {
	return &Watcher{state: state, bot: bot}
}

// jitteredInterval returns basePollInterval ± 40%.
func jitteredInterval() time.Duration {
	jitter := time.Duration(rand.Int63n(int64(basePollInterval)*8/10)) - basePollInterval*4/10
	d := basePollInterval + jitter
	if d < 10*time.Second {
		d = 10 * time.Second
	}
	return d
}

// Run starts the polling loop. Blocks until ctx is cancelled.
// Only polls when there are active watches; sleeps otherwise.
func (w *Watcher) Run(ctx context.Context) {
	log.Printf("[watcher] started (base interval: %v ±40%%)", basePollInterval)

	for {
		wait := jitteredInterval()

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Println("[watcher] stopped")
			return
		case <-timer.C:
		}

		// Skip if nothing to watch
		if len(w.state.GetActiveWatches()) == 0 {
			continue
		}

		w.pollAll(ctx)
	}
}

func (w *Watcher) pollAll(ctx context.Context) {
	// Prune expired watches
	if n := w.state.PruneExpired(); n > 0 {
		log.Printf("[watcher] pruned %d expired watch(es)", n)
	}

	watches := w.state.GetActiveWatches()
	if len(watches) == 0 {
		return
	}

	for _, entry := range watches {
		select {
		case <-ctx.Done():
			return
		default:
		}

		w.pollOne(ctx, entry)

		// Delay between requests when watching multiple dates
		if len(watches) > 1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func (w *Watcher) pollOne(ctx context.Context, entry *WatchEntry) {
	rooms, err := FetchRooms(entry.Date, entry.Nights)
	if err != nil {
		log.Printf("[watcher] %s fetch error: %v", entry.WatchKey(), err)
		return
	}

	diff := w.state.ComputeDiff(entry, rooms)

	if entry.FirstPoll {
		// FirstPoll was just cleared by ComputeDiff, this was the initial snapshot
		availCount := len(entry.PrevAvailable)
		log.Printf("[watcher] %s initial snapshot: %d available rooms", entry.WatchKey(), availCount)
		return
	}

	if !diff.HasChanges() {
		return
	}

	log.Printf("[watcher] %s changes detected: +%d -%d",
		entry.WatchKey(), len(diff.Added), len(diff.Removed))

	msg := formatDiffMessage(entry, diff)
	w.bot.SendToChannel(entry.ChannelID, msg)
}

func formatDiffMessage(entry *WatchEntry, diff Diff) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🏨 **선운산유스호스텔 예약 변동 알림**\n"))
	sb.WriteString(fmt.Sprintf("📅 %s (%s)\n\n", entry.Date, entry.NightsLabel()))

	if len(diff.Added) > 0 {
		sb.WriteString(fmt.Sprintf("✅ **새로 예약 가능** (%d개):\n", len(diff.Added)))
		for _, r := range diff.Added {
			sb.WriteString(fmt.Sprintf("• %s\n", r.String()))
		}
		sb.WriteString("\n")
	}

	if len(diff.Removed) > 0 {
		sb.WriteString(fmt.Sprintf("❌ **예약 마감됨** (%d개):\n", len(diff.Removed)))
		for _, r := range diff.Removed {
			sb.WriteString(fmt.Sprintf("• %s\n", r.String()))
		}
		sb.WriteString("\n")
	}

	// Current availability summary
	availCount := len(entry.PrevAvailable)
	sb.WriteString(fmt.Sprintf("📊 현재 예약가능: %d개\n", availCount))
	sb.WriteString(fmt.Sprintf("🔗 %s", BuildReservationURL(entry.Date, entry.Nights)))

	return sb.String()
}
