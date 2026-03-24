package main

import (
	"fmt"
	"testing"
)

func TestFetchRooms(t *testing.T) {
	rooms, err := FetchRooms("2026-04-01", 1)
	if err != nil {
		t.Fatalf("FetchRooms failed: %v", err)
	}

	if len(rooms) == 0 {
		t.Fatal("expected rooms, got 0")
	}

	var available, booked int
	for _, r := range rooms {
		if r.Available {
			available++
		} else {
			booked++
		}
	}

	fmt.Printf("Total: %d, Available: %d, Booked: %d\n", len(rooms), available, booked)

	// Print first few rooms
	for i, r := range rooms {
		if i >= 5 {
			break
		}
		status := "❌"
		if r.Available {
			status = "✅"
		}
		fmt.Printf("  %s %s (sid=%s)\n", status, r.String(), r.RoomSid)
	}
}

func TestFetchRoomsFullyBooked(t *testing.T) {
	rooms, err := FetchRooms("2026-04-04", 1)
	if err != nil {
		t.Fatalf("FetchRooms failed: %v", err)
	}

	var available int
	for _, r := range rooms {
		if r.Available {
			available++
		}
	}

	fmt.Printf("Apr 4 (Sat): Total=%d, Available=%d\n", len(rooms), available)
}
