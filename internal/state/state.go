// Package state is the relationship, on disk.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nitrocode/breakup/internal/home"
	"golang.org/x/sys/unix"
)

type Status string

const (
	Together Status = "together"
	Apart    Status = "apart"
	Paused   Status = "paused"
)

const maxAttempts = 80

type Attempt struct {
	At   time.Time `json:"at"`
	Tool string    `json:"tool"`
}

type State struct {
	Status      Status     `json:"status"`
	Since       time.Time  `json:"since,omitempty"`
	Until       *time.Time `json:"until,omitempty"`
	Label       string     `json:"label,omitempty"`
	Hardened    bool       `json:"hardened,omitempty"`
	PauseUntil  *time.Time `json:"pause_until,omitempty"`
	Emergencies int        `json:"emergencies,omitempty"`
	Attempts    []Attempt  `json:"attempts,omitempty"`
	Installed   bool       `json:"installed,omitempty"`
	WalkedAway  int        `json:"walked_away,omitempty"` // how many times we made up then left again
	GiveUpTries int        `json:"give_up_tries,omitempty"`
	Name        string     `json:"name,omitempty"`
}

func empty() *State {
	return &State{Status: Together}
}

func (s *State) Effective(now time.Time) Status {
	if s == nil {
		return Together
	}
	switch s.Status {
	case Paused:
		if s.PauseUntil != nil && now.Before(*s.PauseUntil) {
			return Paused
		}
		// pause expired — still apart
		return Apart
	case Apart:
		return Apart
	default:
		return Together
	}
}

func (s *State) TimeIsUp(now time.Time) bool {
	if s == nil || s.Until == nil {
		return false
	}
	return !now.Before(*s.Until)
}

func (s *State) Record(tool string, now time.Time) {
	s.Attempts = append(s.Attempts, Attempt{At: now, Tool: tool})
	if len(s.Attempts) > maxAttempts {
		s.Attempts = s.Attempts[len(s.Attempts)-maxAttempts:]
	}
}

func (s *State) AttemptCount() int {
	return len(s.Attempts)
}

func (s *State) Last() *Attempt {
	if len(s.Attempts) == 0 {
		return nil
	}
	a := s.Attempts[len(s.Attempts)-1]
	return &a
}

func (s *State) SameToolStreak(tool string) int {
	n := 0
	for i := len(s.Attempts) - 1; i >= 0; i-- {
		if s.Attempts[i].Tool != tool {
			break
		}
		n++
	}
	return n
}

func (s *State) AfterMidnight(now time.Time) int {
	n := 0
	for _, a := range s.Attempts {
		h := a.At.Hour()
		if h <= 5 || h >= 23 {
			n++
		}
	}
	_ = now
	return n
}

// Load reads state. Missing file means together.
func Load() (*State, error) {
	path := home.StateFile()
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return empty(), nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var s State
	dec := json.NewDecoder(f)
	if err := dec.Decode(&s); err != nil {
		if errors.Is(err, io.EOF) {
			return empty(), nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	if s.Status == "" {
		s.Status = Together
	}
	return &s, nil
}

// Update exclusive-locks the state file, applies fn, writes atomically.
func Update(fn func(*State) error) (*State, error) {
	if err := os.MkdirAll(home.Dir(), 0o755); err != nil {
		return nil, err
	}
	path := home.StateFile()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		return nil, fmt.Errorf("lock state: %w", err)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN) //nolint:errcheck

	var s State
	dec := json.NewDecoder(f)
	if err := dec.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read state: %w", err)
	}
	if s.Status == "" {
		s.Status = Together
	}

	if err := fn(&s); err != nil {
		return &s, err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		return nil, err
	}
	if err := f.Truncate(int64(len(data))); err != nil {
		return nil, err
	}
	return &s, nil
}

func Save(s *State) error {
	_, err := Update(func(cur *State) error {
		*cur = *s
		return nil
	})
	return err
}

// EnsureDir makes ~/.breakup.
func EnsureDir() error {
	return os.MkdirAll(home.Dir(), 0o755)
}

func Path() string { return home.StateFile() }

func Dir() string { return home.Dir() }
