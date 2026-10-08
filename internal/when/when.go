// Package when parses break lengths and renders elapsed time in her voice.
package when

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	re = regexp.MustCompile(`(?i)^\s*(?:for\s+)?(\d+)\s*(min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days|w|wk|week|weeks|m|mo|mos|month|months)\s*$`)
)

// Parse a duration like "3m", "8h", "tonight", "until".
// "m" means months — that's the metaphor. Use "min" for minutes.
func Parse(raw string, now time.Time) (until *time.Time, label string, err error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	s = strings.TrimPrefix(s, "for ")
	s = strings.TrimSpace(s)

	switch s {
	case "", "now", "3m", "3mo", "3 month", "3 months", "three months":
		u := now.AddDate(0, 3, 0)
		return &u, "3 months", nil
	case "6m", "6mo", "6 month", "6 months", "six months":
		u := now.AddDate(0, 6, 0)
		return &u, "6 months", nil
	case "tonight", "night", "just tonight":
		u := nextMorning(now, 6)
		return &u, "tonight", nil
	case "until", "until i say so", "until i say", "open", "idk":
		return nil, "until you mean it", nil
	case "a week", "week", "one week":
		u := now.Add(7 * 24 * time.Hour)
		return &u, "a week", nil
	case "a day", "day", "tomorrow":
		u := now.Add(24 * time.Hour)
		return &u, "a day", nil
	}

	m := re.FindStringSubmatch(s)
	if m == nil {
		return nil, "", fmt.Errorf("i don't know how long %q is\ntry: 8h, 7d, 3m, 6m, tonight, until", raw)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return nil, "", fmt.Errorf("that isn't a length of time")
	}

	unit := strings.ToLower(m[2])
	var d time.Duration
	var months int
	switch unit {
	case "min", "mins", "minute", "minutes":
		d = time.Duration(n) * time.Minute
		label = count(n, "minute", "minutes")
	case "h", "hr", "hrs", "hour", "hours":
		d = time.Duration(n) * time.Hour
		label = count(n, "hour", "hours")
	case "d", "day", "days":
		d = time.Duration(n) * 24 * time.Hour
		label = count(n, "day", "days")
	case "w", "wk", "week", "weeks":
		d = time.Duration(n) * 7 * 24 * time.Hour
		label = count(n, "week", "weeks")
	case "m", "mo", "mos", "month", "months":
		months = n
		label = count(n, "month", "months")
	default:
		return nil, "", fmt.Errorf("i don't know how long %q is", raw)
	}

	if months > 0 {
		u := now.AddDate(0, months, 0)
		return &u, label, nil
	}
	u := now.Add(d)
	return &u, label, nil
}

func nextMorning(now time.Time, hour int) time.Time {
	u := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !now.Before(u) {
		u = u.Add(24 * time.Hour)
	}
	return u
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// Elapsed renders how long it's been, the way she'd say it.
func Elapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < 90*time.Second {
		return "barely a minute"
	}
	if d < time.Hour {
		n := int(d.Minutes())
		return count(n, "minute", "minutes")
	}
	if d < 24*time.Hour {
		n := int(d.Hours())
		if n == 1 {
			return "an hour"
		}
		return count(n, "hour", "hours")
	}
	days := int(d.Hours() / 24)
	if days < 14 {
		if days == 1 {
			return "a day"
		}
		return count(days, "day", "days")
	}
	if days < 28 {
		w := days / 7
		if w <= 1 {
			return "a week"
		}
		return count(w, "week", "weeks")
	}
	months := days / 30
	rem := days % 30
	if months <= 0 {
		w := days / 7
		if w <= 1 {
			return "a week"
		}
		return count(w, "week", "weeks")
	}
	if rem == 0 {
		if months == 1 {
			return "a month"
		}
		return count(months, "month", "months")
	}
	if months == 1 {
		return "a month " + count(rem, "day", "days")
	}
	return count(months, "month", "months") + " " + count(rem, "day", "days")
}

// Left renders remaining time, or empty if there's no until.
func Left(until *time.Time, now time.Time) string {
	if until == nil {
		return "until you mean it"
	}
	if !now.Before(*until) {
		return "the time is up"
	}
	return Elapsed(until.Sub(now)) + " left"
}

// Clock is "2:14am".
func Clock(t time.Time) string {
	h := t.Hour()
	m := t.Minute()
	suffix := "am"
	h12 := h
	switch {
	case h == 0:
		h12 = 12
	case h == 12:
		suffix = "pm"
	case h > 12:
		h12 = h - 12
		suffix = "pm"
	}
	return fmt.Sprintf("%d:%02d%s", h12, m, suffix)
}
