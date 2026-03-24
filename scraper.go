package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL = "https://www.gochang.go.kr/reserve/index.gochang"
	menuCd  = "DOM_000002402002000000"
)

// Room represents a single room entry from the reservation page.
type Room struct {
	Name      string `json:"name"`      // e.g. "2인실(트윈)"
	Size      string `json:"size"`      // e.g. "26.44m2"
	Capacity  string `json:"capacity"`  // e.g. "2명"
	Price     string `json:"price"`     // e.g. "60,000원"
	Available bool   `json:"available"` // true if bookable
	RoomSid   string `json:"room_sid"`  // room ID from writeFunc (only when available)
}

// Key returns a unique identifier for this room (using RoomSid if available, else name+index).
func (r Room) Key() string {
	if r.RoomSid != "" {
		return r.RoomSid
	}
	return r.Name
}

// String returns a human-readable description of the room.
func (r Room) String() string {
	return fmt.Sprintf("%s | %s | %s | %s", r.Name, r.Size, r.Capacity, r.Price)
}

var writeFuncRe = regexp.MustCompile(`writeFunc\([^,]+,[^,]+,'(\d+)'`)

// FetchRooms fetches the room list for the given date and stay duration.
func FetchRooms(date string, nights int) ([]Room, error) {
	// date format: "2026-04-05"
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", date)
	}
	year, month, day := parts[0], parts[1], parts[2]
	resvSche := year + "-" + month

	url := fmt.Sprintf("%s?menuCd=%s&searchDay=%s&resvSche=%s&reservDay=%d&roomType=1001",
		baseURL, menuCd, day, resvSche, nights)

	log.Printf("[scraper] GET %s", url)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML parse failed: %w", err)
	}

	var rooms []Room

	doc.Find("table.bbs_table tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() < 5 {
			return
		}

		name := strings.TrimSpace(tds.Eq(0).Text())
		if name == "" || name == "\u00a0" {
			return
		}

		room := Room{
			Name:     name,
			Size:     strings.TrimSpace(tds.Eq(1).Text()),
			Capacity: strings.TrimSpace(tds.Eq(2).Text()),
			Price:    strings.TrimSpace(tds.Eq(3).Text()),
		}

		// Check availability by CSS class on the button span
		statusCell := tds.Eq(4)
		span := statusCell.Find("span.cal_btn")
		cls, _ := span.Attr("class")

		if strings.Contains(cls, "cal01") {
			room.Available = true
			// Extract roomSid from onclick
			onclick, _ := statusCell.Find("a").Attr("onclick")
			if m := writeFuncRe.FindStringSubmatch(onclick); len(m) > 1 {
				room.RoomSid = m[1]
			}
		}

		rooms = append(rooms, room)
	})

	log.Printf("[scraper] %s: %d rooms found", date, len(rooms))
	return rooms, nil
}

// BuildReservationURL returns the direct URL for the reservation page.
func BuildReservationURL(date string, nights int) string {
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return baseURL
	}
	return fmt.Sprintf("%s?menuCd=%s&searchDay=%s&resvSche=%s-%s&reservDay=%d&roomType=1001",
		baseURL, menuCd, parts[2], parts[0], parts[1], nights)
}
