package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAndStripSnippet(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rc, []byte("export EDITOR=vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snippet := RCSnippet()
	if err := ensureSnippet(rc, snippet); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(rc)
	if !strings.Contains(string(body), begin) {
		t.Fatal("missing marker")
	}
	if !strings.Contains(string(body), "export EDITOR=vim") {
		t.Fatal("lost existing rc")
	}
	// idempotent
	if err := ensureSnippet(rc, snippet); err != nil {
		t.Fatal(err)
	}
	body, _ = os.ReadFile(rc)
	if strings.Count(string(body), begin) != 1 {
		t.Fatalf("duplicated: %s", body)
	}
	if err := stripSnippet(rc); err != nil {
		t.Fatal(err)
	}
	body, _ = os.ReadFile(rc)
	if strings.Contains(string(body), begin) {
		t.Fatal("marker remained")
	}
	if !strings.Contains(string(body), "export EDITOR=vim") {
		t.Fatal("lost existing rc after strip")
	}
}

func TestScriptExportsBin(t *testing.T) {
	t.Setenv("BREAKUP_HOME", "/tmp/fake-breakup")
	s := Script()
	if !strings.Contains(s, "/tmp/fake-breakup/bin") {
		t.Errorf("script = %s", s)
	}
}
