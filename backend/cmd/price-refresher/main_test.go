package main

import (
	"testing"
	"time"
)

func TestNextRunUsesTenAMPlusSeven(t *testing.T) {
	now := time.Date(2026, 6, 22, 2, 59, 0, 0, time.UTC)
	got := nextRun(now, jakartaOffset)
	want := time.Date(2026, 6, 22, 10, 0, 0, 0, jakartaOffset)
	if !got.Equal(want) {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestNextRunMovesToTomorrowAfterTenAM(t *testing.T) {
	now := time.Date(2026, 6, 22, 3, 1, 0, 0, time.UTC)
	got := nextRun(now, jakartaOffset)
	want := time.Date(2026, 6, 23, 10, 0, 0, 0, jakartaOffset)
	if !got.Equal(want) {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
