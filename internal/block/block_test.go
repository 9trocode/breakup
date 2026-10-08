package block

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlockStripRoundTrip(t *testing.T) {
	orig := "127.0.0.1 localhost\n::1 localhost\n"
	blocked := Block(orig)
	if !HasBlock(blocked) {
		t.Fatal("expected marker")
	}
	if !strings.Contains(blocked, "api.openai.com") {
		t.Fatal("expected openai")
	}
	if !strings.Contains(blocked, "api.anthropic.com") {
		t.Fatal("expected anthropic")
	}
	if !strings.Contains(blocked, "api.x.ai") {
		t.Fatal("expected x.ai")
	}
	back := Strip(blocked)
	if HasBlock(back) {
		t.Fatal("marker survived strip")
	}
	if !strings.Contains(back, "localhost") {
		t.Fatal("lost original")
	}
	if strings.Contains(back, "api.openai.com") {
		t.Fatal("domain survived strip")
	}
}

func TestBlockIdempotent(t *testing.T) {
	orig := "127.0.0.1 localhost\n"
	once := Block(orig)
	twice := Block(once)
	if strings.Count(twice, begin) != 1 {
		t.Fatalf("markers = %d", strings.Count(twice, begin))
	}
}

func TestApplyTempHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	if err := os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BREAKUP_HOSTS", path)
	if err := Apply(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !HasBlock(string(raw)) {
		t.Fatal("not applied")
	}
	if err := Remove(); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(path)
	if HasBlock(string(raw)) {
		t.Fatal("not removed")
	}
}
