package when

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	now := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		in       string
		want     time.Duration
		months   int
		label    string
		untilNil bool
	}{
		{in: "3m", months: 3, label: "3 months"},
		{in: "6 months", months: 6, label: "6 months"},
		{in: "8h", want: 8 * time.Hour, label: "8 hours"},
		{in: "7d", want: 7 * 24 * time.Hour, label: "7 days"},
		{in: "a week", want: 7 * 24 * time.Hour, label: "a week"},
		{in: "until", untilNil: true, label: "until you mean it"},
		{in: "15min", want: 15 * time.Minute, label: "15 minutes"},
		{in: "for 2w", want: 14 * 24 * time.Hour, label: "2 weeks"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			until, label, err := Parse(tt.in, now)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.in, err)
			}
			if label != tt.label {
				t.Errorf("label = %q, want %q", label, tt.label)
			}
			if tt.untilNil {
				if until != nil {
					t.Fatalf("expected nil until")
				}
				return
			}
			if until == nil {
				t.Fatalf("expected until")
			}
			var want time.Time
			if tt.months > 0 {
				want = now.AddDate(0, tt.months, 0)
			} else {
				want = now.Add(tt.want)
			}
			if !until.Equal(want) {
				t.Errorf("until = %s, want %s", until, want)
			}
		})
	}
}

func TestParseTonight(t *testing.T) {
	now := time.Date(2026, 3, 15, 22, 0, 0, 0, time.UTC)
	until, label, err := Parse("tonight", now)
	if err != nil {
		t.Fatal(err)
	}
	if label != "tonight" {
		t.Errorf("label = %q", label)
	}
	if until.Hour() != 6 || until.Day() != 16 {
		t.Errorf("until = %s, want next 6am", until)
	}
}

func TestParseBad(t *testing.T) {
	_, _, err := Parse("banana", time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestElapsed(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "barely a minute"},
		{5 * time.Minute, "5 minutes"},
		{time.Hour, "an hour"},
		{10 * time.Hour, "10 hours"},
		{24 * time.Hour, "a day"},
		{3 * 24 * time.Hour, "3 days"},
		{21 * 24 * time.Hour, "3 weeks"},
		{30 * 24 * time.Hour, "a month"},
		{95 * 24 * time.Hour, "3 months 5 days"},
	}
	for _, tt := range tests {
		got := Elapsed(tt.d)
		if got != tt.want {
			t.Errorf("Elapsed(%s) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestClock(t *testing.T) {
	tm := time.Date(2026, 1, 1, 2, 14, 0, 0, time.UTC)
	if got := Clock(tm); got != "2:14am" {
		t.Errorf("Clock = %q", got)
	}
	tm = time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC)
	if got := Clock(tm); got != "12:05am" {
		t.Errorf("Clock = %q", got)
	}
}
