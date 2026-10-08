// Package hook installs the PATH snippet into the user's shell rc.
package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nitrocode/breakup/internal/guard"
	"github.com/nitrocode/breakup/internal/home"
)

const (
	begin = "# >>> breakup >>>"
	end   = "# <<< breakup <<<"
)

func Script() string {
	var b strings.Builder
	b.WriteString("# managed by breakup. do not edit.\n")
	fmt.Fprintf(&b, "export PATH=%q:$PATH\n", home.BinDir())
	// nvm and friends define npx/claude as functions. those beat PATH.
	b.WriteString("# aliases/functions beat PATH. knock them down.\n")
	for _, name := range guard.AllShimNames() {
		fmt.Fprintf(&b, "unalias %s >/dev/null 2>&1 || true\n", name)
		fmt.Fprintf(&b, "unset -f %s >/dev/null 2>&1 || true\n", name)
	}
	return b.String()
}

func RCSnippet() string {
	return fmt.Sprintf("%s\n[ -f %q ] && . %q\n%s\n", begin, home.HookFile(), home.HookFile(), end)
}

func WriteHook() error {
	if err := os.MkdirAll(home.Dir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(home.HookFile(), []byte(Script()), 0o644)
}

func RCFiles() []string {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var files []string
	shell := filepath.Base(os.Getenv("SHELL"))
	switch shell {
	case "zsh":
		files = []string{filepath.Join(h, ".zshrc")}
	case "bash":
		files = []string{filepath.Join(h, ".bashrc"), filepath.Join(h, ".bash_profile")}
	default:
		files = []string{
			filepath.Join(h, ".zshrc"),
			filepath.Join(h, ".bashrc"),
		}
	}
	return files
}

func Install() (string, error) {
	if err := WriteHook(); err != nil {
		return "", err
	}
	snippet := RCSnippet()
	var touched string
	for _, rc := range RCFiles() {
		if err := ensureSnippet(rc, snippet); err != nil {
			continue
		}
		touched = rc
		break
	}
	if touched == "" {
		// last resort: create zshrc
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		rc := filepath.Join(h, ".zshrc")
		if err := ensureSnippet(rc, snippet); err != nil {
			return "", err
		}
		touched = rc
	}
	return touched, nil
}

func Uninstall() error {
	_ = os.Remove(home.HookFile())
	for _, rc := range candidateRCs() {
		_ = stripSnippet(rc)
	}
	return nil
}

func candidateRCs() []string {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(h, ".zshrc"),
		filepath.Join(h, ".bashrc"),
		filepath.Join(h, ".bash_profile"),
	}
}

func ensureSnippet(path, snippet string) error {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	body := string(raw)
	if strings.Contains(body, begin) {
		next := replaceBlock(body, snippet)
		if next == body {
			return nil
		}
		return write(path, next)
	}
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += "\n" + snippet
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return write(path, body)
}

func stripSnippet(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	body := string(raw)
	if !strings.Contains(body, begin) {
		return nil
	}
	next := replaceBlock(body, "")
	next = strings.TrimRight(next, "\n") + "\n"
	return write(path, next)
}

func replaceBlock(body, snippet string) string {
	start := strings.Index(body, begin)
	if start < 0 {
		return body
	}
	endIdx := strings.Index(body[start:], end)
	if endIdx < 0 {
		return body
	}
	endIdx = start + endIdx + len(end)
	for endIdx < len(body) && (body[endIdx] == '\n' || body[endIdx] == '\r') {
		endIdx++
	}
	// eat a leading extra newline if we are deleting
	out := body[:start] + snippet + body[endIdx:]
	return out
}

func write(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	fi, err := os.Stat(path)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = fi.Mode()
	}
	return os.WriteFile(path, []byte(body), mode)
}
