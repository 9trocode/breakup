package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BREAKUP_HOME", dir)

	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	until := now.AddDate(0, 3, 0)

	got, err := Update(func(s *State) error {
		s.Status = Apart
		s.Since = now
		s.Until = &until
		s.Label = "3 months"
		s.Record("grok", now.Add(time.Hour))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != Apart {
		t.Fatalf("status = %s", got.Status)
	}
	if got.AttemptCount() != 1 {
		t.Fatalf("attempts = %d", got.AttemptCount())
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Label != "3 months" {
		t.Errorf("label = %q", loaded.Label)
	}
	if loaded.Last() == nil || loaded.Last().Tool != "grok" {
		t.Errorf("last = %+v", loaded.Last())
	}
	if loaded.Effective(now) != Apart {
		t.Errorf("effective = %s", loaded.Effective(now))
	}
}

func TestEffectivePauseExpiry(t *testing.T) {
	now := time.Now()
	until := now.Add(-time.Minute)
	s := &State{Status: Paused, PauseUntil: &until}
	if s.Effective(now) != Apart {
		t.Errorf("expired pause should be apart, got %s", s.Effective(now))
	}
	future := now.Add(time.Hour)
	s.PauseUntil = &future
	if s.Effective(now) != Paused {
		t.Errorf("got %s", s.Effective(now))
	}
}

func TestLoadMissingIsTogether(t *testing.T) {
	t.Setenv("BREAKUP_HOME", filepath.Join(t.TempDir(), "nope"))
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != Together {
		t.Errorf("got %s", s.Status)
	}
}

func TestSameToolStreak(t *testing.T) {
	now := time.Now()
	s := &State{}
	s.Record("grok", now)
	s.Record("grok", now)
	s.Record("claude", now)
	s.Record("grok", now)
	s.Record("grok", now)
	s.Record("grok", now)
	if s.SameToolStreak("grok") != 3 {
		t.Errorf("streak = %d", s.SameToolStreak("grok"))
	}
}
