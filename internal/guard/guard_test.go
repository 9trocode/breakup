package guard

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nitrocode/breakup/internal/state"
)

func TestDecide(t *testing.T) {
	now := time.Now()
	apart := &state.State{Status: state.Apart, Since: now}
	together := &state.State{Status: state.Together}
	pauseUntil := now.Add(time.Hour)
	paused := &state.State{Status: state.Paused, PauseUntil: &pauseUntil}

	tests := []struct {
		name string
		s    *state.State
		tool string
		args []string
		want Decision
	}{
		{"together passes", together, "claude", nil, Pass},
		{"apart blocks claude", apart, "claude", nil, Block},
		{"paused passes", paused, "claude", nil, PassPaused},
		{"nil state passes", nil, "claude", nil, Pass},
		{"npx eslint passes while apart", apart, "npx", []string{"eslint", "."}, Pass},
		{"npx claude-code blocks", apart, "npx", []string{"-y", "@anthropic-ai/claude-code"}, Block},
		{"bunx aider blocks", apart, "bunx", []string{"aider-chat"}, Block},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.s, tt.tool, tt.args, now)
			if got != tt.want {
				t.Errorf("Decide = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCLI(t *testing.T) {
	if !IsCLI("claude") || !IsCLI("/opt/homebrew/bin/grok") {
		t.Fatal("expected known clis")
	}
	if IsCLI("git") || IsCLI("go") {
		t.Fatal("should not block git/go")
	}
}

func TestInstallShims(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BREAKUP_HOME", dir)
	n, err := InstallShims("/usr/bin/breakup")
	if err != nil {
		t.Fatal(err)
	}
	if n < 10 {
		t.Fatalf("installed %d shims", n)
	}
	claude := filepath.Join(dir, "bin", "claude")
	body, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !contains(s, "intercept --tool \"claude\"") {
		t.Errorf("shim body: %s", s)
	}
	makeup := filepath.Join(dir, "bin", "makeup")
	if _, err := os.Stat(makeup); err != nil {
		t.Fatal(err)
	}
}

func TestRealPathSkipsShimDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BREAKUP_HOME", filepath.Join(root, ".breakup"))
	shimDir := filepath.Join(root, ".breakup", "bin")
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shimDir, "claude"), []byte("shim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "claude"), []byte("real"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+realDir)
	got, err := RealPath("claude")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(realDir, "claude") {
		t.Errorf("got %s", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
