package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WatchEntry represents a date being watched for availability changes.
type WatchEntry struct {
	Date      string `json:"date"`       // "2026-04-05"
	Nights    int    `json:"nights"`     // 1 = 1박2일, 2 = 2박3일, ...
	ChannelID string `json:"channel_id"` // Discord channel to notify

	// Previous snapshot of available room SIDs → Room
	PrevAvailable map[string]Room `json:"prev_available"`
	FirstPoll     bool            `json:"first_poll"` // true if never polled yet
}

// WatchKey returns a unique key for this watch entry.
func (w *WatchEntry) WatchKey() string {
	return fmt.Sprintf("%s:%d", w.Date, w.Nights)
}

// NightsLabel returns a human-readable label like "1박2일".
func (w *WatchEntry) NightsLabel() string {
	return fmt.Sprintf("%d박%d일", w.Nights, w.Nights+1)
}

// Diff represents changes between two availability snapshots.
type Diff struct {
	Added   []Room // newly available
	Removed []Room // no longer available
}

// HasChanges returns true if there are any differences.
func (d Diff) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Removed) > 0
}

// stateData is the JSON-serializable form of State.
type stateData struct {
	Watches map[string]*WatchEntry `json:"watches"`
}

// State manages all active watch entries (thread-safe).
type State struct {
	mu       sync.RWMutex
	watches  map[string]*WatchEntry // key = "date:nights"
	filePath string
}

// NewState creates an empty state.
func NewState(filePath string) *State {
	return &State{
		watches:  make(map[string]*WatchEntry),
		filePath: filePath,
	}
}

// Load reads state from the JSON file. Silently starts fresh if file doesn't exist.
func (s *State) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[state] ⚠️  state.json 읽기 실패: %v", err)
		}
		return
	}

	var sd stateData
	if err := json.Unmarshal(data, &sd); err != nil {
		log.Printf("[state] ⚠️  state.json 파싱 실패: %v", err)
		return
	}

	if sd.Watches == nil {
		return
	}

	// Filter out expired entries on load
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	loaded := 0
	expired := 0
	for key, w := range sd.Watches {
		if w.Date < today {
			expired++
			continue
		}
		s.watches[key] = w
		loaded++
	}

	if loaded > 0 || expired > 0 {
		log.Printf("[state] 📂 state.json 로드: %d개 복원, %d개 만료 제거", loaded, expired)
	}
}

// Save writes the current state to the JSON file.
func (s *State) Save() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.watches) == 0 {
		// Clean up file if nothing to save
		os.Remove(s.filePath)
		return
	}

	sd := stateData{Watches: s.watches}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		log.Printf("[state] ⚠️  state.json 직렬화 실패: %v", err)
		return
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[state] ⚠️  디렉토리 생성 실패: %v", err)
		return
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		log.Printf("[state] ⚠️  state.json 저장 실패: %v", err)
	}
}

// AddWatch adds a date to the watch list. Returns false if already watching.
func (s *State) AddWatch(date string, nights int, channelID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%d", date, nights)
	if _, exists := s.watches[key]; exists {
		return false
	}

	s.watches[key] = &WatchEntry{
		Date:          date,
		Nights:        nights,
		ChannelID:     channelID,
		PrevAvailable: nil,
		FirstPoll:     true,
	}
	return true
}

// GetEntry returns the watch entry for the given date/nights, or nil if not found.
func (s *State) GetEntry(date string, nights int) *WatchEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.watches[fmt.Sprintf("%s:%d", date, nights)]
}

// RemoveWatch removes a date from the watch list. Returns false if not found.
func (s *State) RemoveWatch(date string, nights int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%d", date, nights)
	if _, exists := s.watches[key]; !exists {
		return false
	}
	delete(s.watches, key)
	return true
}

// ListWatches returns a copy of all active watch entries.
func (s *State) ListWatches() []*WatchEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*WatchEntry, 0, len(s.watches))
	for _, w := range s.watches {
		entries = append(entries, w)
	}
	return entries
}

// GetActiveWatches returns watches that haven't expired (date not in the past).
func (s *State) GetActiveWatches() []*WatchEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	var active []*WatchEntry
	for _, w := range s.watches {
		if w.Date >= today {
			active = append(active, w)
		}
	}
	return active
}

// PruneExpired removes watch entries for dates that have passed.
func (s *State) PruneExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	count := 0
	for key, w := range s.watches {
		if w.Date < today {
			delete(s.watches, key)
			count++
		}
	}
	return count
}

// ComputeDiff compares the current available rooms with the previous snapshot
// and updates the entry's state. Returns the diff.
func (s *State) ComputeDiff(entry *WatchEntry, currentRooms []Room) Diff {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build current available map
	currentAvail := make(map[string]Room)
	for _, r := range currentRooms {
		if r.Available {
			currentAvail[r.Key()] = r
		}
	}

	var diff Diff

	if entry.FirstPoll {
		// First poll: no diff, just record the state
		entry.PrevAvailable = currentAvail
		entry.FirstPoll = false
		return diff
	}

	// Find newly available (in current but not in prev)
	for key, room := range currentAvail {
		if _, existed := entry.PrevAvailable[key]; !existed {
			diff.Added = append(diff.Added, room)
		}
	}

	// Find no longer available (in prev but not in current)
	for key, room := range entry.PrevAvailable {
		if _, exists := currentAvail[key]; !exists {
			diff.Removed = append(diff.Removed, room)
		}
	}

	// Update snapshot
	entry.PrevAvailable = currentAvail

	return diff
}
