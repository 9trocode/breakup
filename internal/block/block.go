// Package block locks the door: LLM API hosts in /etc/hosts.
package block

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	begin = "# BEGIN breakup"
	end   = "# END breakup"
)

var Domains = []string{
	"api.openai.com",
	"api.anthropic.com",
	"api.x.ai",
	"inference.x.ai",
	"api.groq.com",
	"openrouter.ai",
	"api.openrouter.ai",
	"api.mistral.ai",
	"api.together.xyz",
	"api.deepseek.com",
	"api.perplexity.ai",
	"api.cohere.ai",
	"api.cohere.com",
	"api.fireworks.ai",
	"generativelanguage.googleapis.com",
	"api.githubcopilot.com",
	"copilot-proxy.githubusercontent.com",
	"api2.cursor.sh",
	"api3.cursor.sh",
	"api4.cursor.sh",
	"repo42.cursor.sh",
	"chatgpt.com",
	"chat.openai.com",
	"claude.ai",
	"console.anthropic.com",
	"grok.x.ai",
	"gemini.google.com",
}

func HostsPath() string {
	if p := os.Getenv("BREAKUP_HOSTS"); p != "" {
		return p
	}
	return "/etc/hosts"
}

func Block(contents string) string {
	stripped := Strip(contents)
	var b strings.Builder
	b.WriteString(strings.TrimRight(stripped, "\n"))
	b.WriteString("\n\n")
	b.WriteString(begin)
	b.WriteByte('\n')
	for _, d := range Domains {
		fmt.Fprintf(&b, "127.0.0.1 %s\n", d)
		fmt.Fprintf(&b, "::1 %s\n", d)
	}
	b.WriteString(end)
	b.WriteByte('\n')
	return b.String()
}

func Strip(contents string) string {
	lines := strings.Split(contents, "\n")
	var out []string
	in := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == begin {
			in = true
			continue
		}
		if trim == end {
			in = false
			continue
		}
		if in {
			continue
		}
		out = append(out, line)
	}
	s := strings.Join(out, "\n")
	return strings.TrimRight(s, "\n") + "\n"
}

func HasBlock(contents string) bool {
	return strings.Contains(contents, begin)
}

func Apply() error {
	path := HostsPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	next := Block(string(raw))
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return fmt.Errorf("write %s: %w\ntry: sudo breakup harden", path, err)
	}
	flushDNS()
	return nil
}

func Remove() error {
	path := HostsPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if !HasBlock(string(raw)) {
		return nil
	}
	next := Strip(string(raw))
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return fmt.Errorf("write %s: %w\ntry: sudo breakup soften", path, err)
	}
	flushDNS()
	return nil
}

func flushDNS() {
	if runtime.GOOS != "darwin" {
		return
	}
	_ = exec.Command("dscacheutil", "-flushcache").Run()
	_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
}

func NeedsRoot() bool {
	path := HostsPath()
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return true
	}
	_ = f.Close()
	return false
}

// ReexecSudo runs the current binary with sudo and the given args.
func ReexecSudo(args ...string) error {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	cmd := exec.Command("sudo", append([]string{"-p", "i need to lock the door. password: ", self}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func MacQuitApps() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	apps := []string{"Cursor", "Claude", "ChatGPT", "Windsurf", "Perplexity", "Copilot"}
	var quit []string
	for _, a := range apps {
		err := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to quit`, a)).Run()
		if err == nil {
			quit = append(quit, a)
		}
	}
	return quit
}
