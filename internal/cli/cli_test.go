package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nitrocode/breakup/internal/guard"
	"github.com/nitrocode/breakup/internal/state"
)

func setupHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("BREAKUP_HOME", filepath.Join(dir, ".breakup"))
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("BREAKUP_HOSTS", filepath.Join(dir, "hosts"))
	if err := os.WriteFile(filepath.Join(dir, "hosts"), []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nowFn = time.Now
	ttyIn = func() bool { return false }
	t.Cleanup(func() {
		nowFn = time.Now
		ttyIn = func() bool { return false }
		stdin = os.Stdin
	})
	if _, err := run(t, "name", "june"); err != nil {
		t.Fatalf("name june: %v", err)
	}
	return dir
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf
	cmd := newRoot()
	cmd.SetArgs(args)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	err := cmd.Execute()
	return buf.String(), err
}

func TestBreakupForAndStatus(t *testing.T) {
	setupHome(t)
	now := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }
	t.Cleanup(func() { nowFn = time.Now })

	out, err := run(t, "for", "3m")
	if err != nil {
		t.Fatalf("breakup for 3m: %v\n%s", err, out)
	}
	if !strings.Contains(out, "space") && !strings.Contains(out, "break") && !strings.Contains(out, "prompt") {
		t.Errorf("expected a breakup letter, got:\n%s", out)
	}

	st, err := state.Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != state.Apart {
		t.Fatalf("status = %s", st.Status)
	}
	if st.Label != "3 months" {
		t.Errorf("label = %q", st.Label)
	}

	out, err = run(t, "status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "apart") {
		t.Errorf("status: %s", out)
	}

	out, err = run(t, "makeup")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(out), "no") && !strings.Contains(out, "ready") {
		t.Errorf("expected refusal: %s", out)
	}
	st, _ = state.Load()
	if st.Status != state.Apart {
		t.Fatalf("makeup should not have worked, status=%s", st.Status)
	}
}

func TestMakeupAfterTime(t *testing.T) {
	setupHome(t)
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return start }
	if _, err := run(t, "for", "8h"); err != nil {
		t.Fatal(err)
	}
	nowFn = func() time.Time { return start.Add(9 * time.Hour) }
	t.Cleanup(func() { nowFn = time.Now })

	out, err := run(t, "makeup")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "okay") && !strings.Contains(out, "fine") && !strings.Contains(out, "come in") {
		t.Errorf("expected yes: %s", out)
	}
	st, _ := state.Load()
	if st.Status != state.Together {
		t.Fatalf("status = %s", st.Status)
	}
}

func TestGiveUpTakesThreeThenPlease(t *testing.T) {
	setupHome(t)
	ttyIn = func() bool { return false }
	if _, err := run(t, "for", "6m"); err != nil {
		t.Fatal(err)
	}

	out, err := run(t, "makeup", "--i-cant-do-this")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1 of 3") {
		t.Errorf("round 1: %s", out)
	}
	st, _ := state.Load()
	if st.Effective(time.Now()) != state.Apart {
		t.Fatalf("should still be apart after 1, got %s", st.Status)
	}

	out, err = run(t, "makeup", "--i-cant-do-this", "--please")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "2 of 3") {
		t.Errorf("round 2 with please still refused: %s", out)
	}
	st, _ = state.Load()
	if st.Status != state.Apart {
		t.Fatalf("please should not skip round 2, status=%s", st.Status)
	}

	out, err = run(t, "makeup", "--i-cant-do-this")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "--please") {
		t.Errorf("round 3 should demand please: %s", out)
	}
	st, _ = state.Load()
	if st.Status != state.Apart {
		t.Fatalf("round 3 without please should fail, status=%s", st.Status)
	}

	out, err = run(t, "makeup", "--i-cant-do-this", "--please")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "we're back") && !strings.Contains(out, "please") {
		t.Errorf("expected yes after begging: %s", out)
	}
	st, _ = state.Load()
	if st.Effective(time.Now()) != state.Together {
		t.Fatalf("status = %s", st.Status)
	}
}

func TestGiveUpInteractiveRequiresBeg(t *testing.T) {
	setupHome(t)
	if _, err := run(t, "for", "3m"); err != nil {
		t.Fatal(err)
	}

	ttyIn = func() bool { return true }
	t.Cleanup(func() { ttyIn = func() bool { return false } })

	stdin = strings.NewReader("the bug is hard\ni googled it\nidk\n")
	out, err := run(t, "makeup", "--i-cant-do-this")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "why.") {
		t.Errorf("expected grill: %s", out)
	}
	if !strings.Contains(out, "please") && !strings.Contains(out, "beg") && !strings.Contains(out, "asking") {
		t.Errorf("expected a refused beg: %s", out)
	}
	st, _ := state.Load()
	if st.Status != state.Apart {
		t.Fatalf("weak beg should not work, status=%s", st.Status)
	}

	stdin = strings.NewReader("stuck on a ticket\nnothing really\nplease let me back\n")
	out, err = run(t, "makeup", "--i-cant-do-this")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "we're back") && !strings.Contains(out, "beg") && !strings.Contains(out, "together") {
		t.Errorf("expected yes after please: %s", out)
	}
	st, _ = state.Load()
	if st.Effective(time.Now()) != state.Together {
		t.Fatalf("status = %s", st.Status)
	}
}

func TestInterceptBlocks(t *testing.T) {
	setupHome(t)
	now := time.Date(2026, 4, 1, 3, 14, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }
	t.Cleanup(func() { nowFn = time.Now })
	if _, err := run(t, "for", "3m"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "intercept", "--tool", "grok", "--", "hello")
	if !errors.Is(err, guard.ErrBlocked) {
		t.Fatalf("err = %v", err)
	}
	if out == "" {
		t.Fatal("expected a letter")
	}
	if !strings.Contains(out, "june") {
		t.Errorf("expected her name on the intercept: %s", out)
	}
	st, _ := state.Load()
	if st.AttemptCount() != 1 {
		t.Fatalf("attempts = %d", st.AttemptCount())
	}
	if last := st.Last(); last == nil || last.Tool != "grok" {
		t.Errorf("last = %+v", last)
	}
}

func TestAlreadyApart(t *testing.T) {
	setupHome(t)
	if _, err := run(t, "now"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "for", "3m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "already") {
		t.Errorf("expected already apart: %s", out)
	}
}

func TestEmergencyPause(t *testing.T) {
	setupHome(t)
	now := time.Now()
	nowFn = func() time.Time { return now }
	t.Cleanup(func() { nowFn = time.Now })
	if _, err := run(t, "for", "3m"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "makeup", "--the-build-is-on-fire")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "two hours") && !strings.Contains(out, "watching") {
		t.Errorf("expected emergency letter: %s", out)
	}
	st, _ := state.Load()
	if st.Effective(now) != state.Paused {
		t.Fatalf("effective = %s", st.Effective(now))
	}
}

func TestUninstallLeavesRcClean(t *testing.T) {
	dir := setupHome(t)
	if _, err := run(t, "install"); err != nil {
		t.Fatal(err)
	}
	zshrc := filepath.Join(dir, ".zshrc")
	raw, _ := os.ReadFile(zshrc)
	if !strings.Contains(string(raw), "breakup") {
		t.Fatalf("expected hook in zshrc: %s", raw)
	}
	if _, err := run(t, "uninstall"); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(zshrc)
	if strings.Contains(string(raw), ">>> breakup >>>") {
		t.Fatalf("hook remained: %s", raw)
	}
}

func TestNeedNameBeforeBreak(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("BREAKUP_HOME", filepath.Join(dir, ".breakup"))
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("BREAKUP_HOSTS", filepath.Join(dir, "hosts"))
	if err := os.WriteFile(filepath.Join(dir, "hosts"), []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ttyIn = func() bool { return false }
	t.Cleanup(func() { ttyIn = func() bool { return false } })

	out, err := run(t, "for", "3m")
	if err == nil {
		t.Fatal("expected to refuse an unnamed break")
	}
	if !strings.Contains(out, "nobody") && !strings.Contains(out, "name me") {
		t.Errorf("expected a name demand: %s", out)
	}
}

func TestNameAndRename(t *testing.T) {
	setupHome(t) // already june
	out, err := run(t, "name")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "june") {
		t.Errorf("who: %s", out)
	}

	out, err = run(t, "name", "mara")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "mara") {
		t.Errorf("rename while together: %s", out)
	}

	if _, err := run(t, "for", "3m"); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "name", "ada")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sand") && !strings.Contains(out, "no") {
		t.Errorf("rename while apart should fail: %s", out)
	}
	st, _ := state.Load()
	if st.Name != "mara" {
		t.Fatalf("name changed while apart: %s", st.Name)
	}
}

func TestNameFlagOnBreak(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("BREAKUP_HOME", filepath.Join(dir, ".breakup"))
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("BREAKUP_HOSTS", filepath.Join(dir, "hosts"))
	if err := os.WriteFile(filepath.Join(dir, "hosts"), []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ttyIn = func() bool { return false }

	out, err := run(t, "for", "8h", "--name", "June")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "june") {
		t.Errorf("expected sign-off/title: %s", out)
	}
	st, _ := state.Load()
	if st.Name != "june" {
		t.Fatalf("name = %q", st.Name)
	}
}
