// Package render prints her voice to a terminal without looking like a dashboard.
package render

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

func Enabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func Letter(w io.Writer, body string) {
	body = strings.TrimRight(body, "\n")
	color := Enabled(w)
	bar := "│"
	voice := lipgloss.NewStyle().Foreground(lipgloss.Color("#E8A0BF"))
	mute := lipgloss.NewStyle().Foreground(lipgloss.Color("#6E6762"))

	fmt.Fprintln(w)
	for _, line := range strings.Split(body, "\n") {
		if !color {
			if line == "" {
				fmt.Fprintf(w, "%s\n", bar)
				continue
			}
			fmt.Fprintf(w, "%s  %s\n", bar, line)
			continue
		}
		if line == "" {
			fmt.Fprintf(w, "%s\n", mute.Render(bar))
			continue
		}
		fmt.Fprintf(w, "%s  %s\n", mute.Render(bar), voice.Render(line))
	}
	fmt.Fprintln(w)
}

func Meta(w io.Writer, body string) {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return
	}
	color := Enabled(w)
	mute := lipgloss.NewStyle().Foreground(lipgloss.Color("#8A837C"))
	for _, line := range strings.Split(body, "\n") {
		if color {
			fmt.Fprintln(w, mute.Render(line))
		} else {
			fmt.Fprintln(w, line)
		}
	}
}

func Title(w io.Writer, s string) {
	color := Enabled(w)
	st := lipgloss.NewStyle().Foreground(lipgloss.Color("#C45C78")).Bold(true)
	if color {
		fmt.Fprintln(w, st.Render(s))
		return
	}
	fmt.Fprintln(w, s)
}

func Prompt(w io.Writer) {
	mute := lipgloss.NewStyle().Foreground(lipgloss.Color("#8A837C"))
	if Enabled(w) {
		fmt.Fprint(w, mute.Render("> "))
		return
	}
	fmt.Fprint(w, "> ")
}
