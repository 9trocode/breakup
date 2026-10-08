package letter

import (
	"strings"
	"testing"
	"time"

	"github.com/nitrocode/breakup/internal/state"
)

func TestPhaseOf(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want Phase
	}{
		{time.Hour, JustLeft},
		{20 * time.Hour, FirstNight},
		{3 * 24 * time.Hour, FirstWeek},
		{20 * 24 * time.Hour, FirstMonth},
		{60 * 24 * time.Hour, ThreeMonths},
		{120 * 24 * time.Hour, SixMonths},
		{200 * 24 * time.Hour, After},
	}
	for _, tt := range tests {
		if got := PhaseOf(tt.d); got != tt.want {
			t.Errorf("PhaseOf(%s) = %d, want %d", tt.d, got, tt.want)
		}
	}
}

func TestBlockedMentionsTheReach(t *testing.T) {
	now := time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC)
	since := now.Add(-80 * 24 * time.Hour) // ~3 months
	st := &state.State{
		Status: state.Apart,
		Since:  since,
		Label:  "3 months",
		Attempts: []state.Attempt{
			{At: now.Add(-time.Hour), Tool: "grok"},
		},
	}
	body := Blocked(Context{Tool: "grok", Now: now, State: st})
	if body == "" {
		t.Fatal("empty letter")
	}
	// should not look like a help page
	if strings.Contains(strings.ToLower(body), "usage:") {
		t.Errorf("letter looks like usage: %s", body)
	}
}

func TestMakeupNoWhileEarly(t *testing.T) {
	now := time.Now()
	until := now.Add(60 * 24 * time.Hour)
	st := &state.State{
		Status: state.Apart,
		Since:  now.Add(-2 * 24 * time.Hour),
		Until:  &until,
		Label:  "3 months",
	}
	body := MakeupNo(Context{Now: now, State: st})
	if !strings.Contains(body, "no") && !strings.Contains(body, "don't") && !strings.Contains(body, "ready") {
		t.Errorf("expected a refusal, got %q", body)
	}
}

func TestBreakupContainsDuration(t *testing.T) {
	body := Breakup("3 months", "june", false)
	if !strings.Contains(body, "3 months") {
		t.Errorf("expected duration in letter: %s", body)
	}
	if !strings.Contains(body, "— june") {
		t.Errorf("expected sign-off: %s", body)
	}
	hard := Breakup("3 months", "june", true)
	if !strings.Contains(hard, "locked") && !strings.Contains(hard, "door") {
		t.Errorf("expected harden note: %s", hard)
	}
}

func TestWhyCountsAttempts(t *testing.T) {
	now := time.Now()
	st := &state.State{
		Status: state.Apart,
		Since:  now.Add(-5 * 24 * time.Hour),
		Attempts: []state.Attempt{
			{At: now.Add(-time.Hour), Tool: "claude"},
			{At: now.Add(-time.Minute), Tool: "grok"},
		},
	}
	body := Why(st, now)
	if !strings.Contains(body, "twice") && !strings.Contains(body, "2") {
		t.Errorf("expected attempt count: %s", body)
	}
}

func TestStatusTogether(t *testing.T) {
	st := &state.State{Status: state.Together}
	body := Status(st, time.Now())
	if !strings.Contains(body, "together") {
		t.Errorf("got %q", body)
	}
}

func TestEmergencyExhausted(t *testing.T) {
	body := Emergency(2, 3)
	if !strings.Contains(body, "isn't an emergency") {
		t.Errorf("got %q", body)
	}
}
