package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSandboxE2E builds the real binary and runs it in an isolated HOME.
// This is the PATH/shim/exec path unit tests never touch.
func TestSandboxE2E(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	realDir := filepath.Join(tmp, "real")
	binDir := filepath.Join(tmp, "bin")
	hosts := filepath.Join(tmp, "hosts")
	buHome := filepath.Join(home, ".breakup")
	for _, d := range []string{home, realDir, binDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	breakup := filepath.Join(binDir, "breakup")
	build := exec.Command("go", "build", "-o", breakup, ".")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	writeSh := func(name, body string) {
		p := filepath.Join(realDir, name)
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeSh("claude", "#!/bin/sh\necho REAL_CLAUDE \"$@\"\n")
	writeSh("grok", "#!/bin/sh\necho REAL_GROK \"$@\"\n")
	writeSh("npx", "#!/bin/sh\necho REAL_NPX \"$@\"\n")
	writeSh("git", "#!/bin/sh\necho REAL_GIT \"$@\"\n")

	shimPATH := func() string {
		return filepath.Join(buHome, "bin") + string(os.PathListSeparator) +
			binDir + string(os.PathListSeparator) +
			realDir + string(os.PathListSeparator) +
			"/usr/bin" + string(os.PathListSeparator) + "/bin"
	}
	baseEnv := func() []string {
		return []string{
			"HOME=" + home,
			"BREAKUP_HOME=" + buHome,
			"BREAKUP_HOSTS=" + hosts,
			"SHELL=/bin/sh",
			"NO_COLOR=1",
			"PATH=" + binDir + string(os.PathListSeparator) + realDir + string(os.PathListSeparator) + "/usr/bin:/bin",
		}
	}
	run := func(path string, env []string, args ...string) (string, int) {
		t.Helper()
		cmd := exec.Command(path, args...)
		cmd.Env = env
		cmd.Dir = tmp
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				t.Fatalf("%s %v: %v\n%s", path, args, err, buf.String())
			}
		}
		return buf.String(), code
	}
	bu := func(args ...string) (string, int) {
		return run(breakup, baseEnv(), args...)
	}

	out, code := bu("for", "3m")
	if code == 0 {
		t.Fatalf("unnamed break should fail:\n%s", out)
	}
	if !strings.Contains(out, "nobody") {
		t.Fatalf("expected nobody letter:\n%s", out)
	}

	out, code = bu("name", "please")
	if code == 0 {
		t.Fatalf("name please should fail:\n%s", out)
	}

	out, code = bu("name", "June")
	if code != 0 {
		t.Fatalf("name june: %d\n%s", code, out)
	}
	if !strings.Contains(out, "june") {
		t.Fatalf("name letter:\n%s", out)
	}

	out, code = bu("for", "8h")
	if code != 0 {
		t.Fatalf("for 8h: %d\n%s", code, out)
	}
	if !strings.Contains(out, "— june") {
		t.Fatalf("sign-off missing:\n%s", out)
	}
	claudeShim := filepath.Join(buHome, "bin", "claude")
	if _, err := os.Stat(claudeShim); err != nil {
		t.Fatal("claude shim missing")
	}
	zshrc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(zshrc, []byte(">>> breakup >>>")) {
		t.Fatalf("zshrc hook missing:\n%s", zshrc)
	}
	hookBody, err := os.ReadFile(filepath.Join(buHome, "hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(hookBody, []byte("unset -f npx")) {
		t.Fatalf("hook should strip npx function:\n%s", hookBody)
	}

	shimEnv := append(baseEnv()[:0:0], baseEnv()...)
	for i, e := range shimEnv {
		if strings.HasPrefix(e, "PATH=") {
			shimEnv[i] = "PATH=" + shimPATH()
		}
	}

	out, code = run(claudeShim, shimEnv, "hello", "world")
	if code != 1 {
		t.Fatalf("blocked claude exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "june") {
		t.Fatalf("blocked letter:\n%s", out)
	}
	if strings.Contains(out, "REAL_CLAUDE") {
		t.Fatalf("real claude ran while blocked:\n%s", out)
	}

	out, code = run(claudeShim, shimEnv, "--help")
	if code != 1 {
		t.Fatalf("claude --help should block, exit %d\n%s", code, out)
	}
	if strings.Contains(strings.ToLower(out), "usage:") && strings.Contains(out, "intercept") {
		t.Fatalf("cobra leaked intercept help:\n%s", out)
	}

	npxShim := filepath.Join(buHome, "bin", "npx")
	out, code = run(npxShim, shimEnv, "eslint", ".")
	if code != 0 {
		t.Fatalf("npx eslint should pass: %d\n%s", code, out)
	}
	if !strings.Contains(out, "REAL_NPX eslint .") {
		t.Fatalf("npx eslint passthrough:\n%s", out)
	}

	out, code = run(npxShim, shimEnv, "-y", "@anthropic-ai/claude-code")
	if code != 1 {
		t.Fatalf("npx claude-code should block: %d\n%s", code, out)
	}
	if strings.Contains(out, "REAL_NPX") {
		t.Fatalf("npx claude-code leaked:\n%s", out)
	}

	out, code = run(filepath.Join(realDir, "git"), shimEnv, "hello")
	if code != 0 || !strings.Contains(out, "REAL_GIT hello") {
		t.Fatalf("git: %d %s", code, out)
	}

	out, code = bu("status", "--json")
	if code != 0 {
		t.Fatalf("status json: %s", out)
	}
	var st map[string]any
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if st["name"] != "june" {
		t.Fatalf("json name = %v", st["name"])
	}

	out, _ = bu("makeup")
	if !strings.Contains(strings.ToLower(out), "no") {
		t.Fatalf("makeup should refuse:\n%s", out)
	}
	out, _ = bu("name", "mara")
	if !strings.Contains(out, "sand") {
		t.Fatalf("rename apart:\n%s", out)
	}

	out, _ = bu("makeup", "--i-cant-do-this")
	if !strings.Contains(out, "1 of 3") {
		t.Fatalf("giveup 1:\n%s", out)
	}
	out, _ = bu("makeup", "--i-cant-do-this", "--please")
	if !strings.Contains(out, "2 of 3") {
		t.Fatalf("giveup 2:\n%s", out)
	}
	out, _ = bu("makeup", "--i-cant-do-this")
	if !strings.Contains(out, "--please") {
		t.Fatalf("giveup 3:\n%s", out)
	}
	_, code = run(claudeShim, shimEnv, "x")
	if code != 1 {
		t.Fatalf("still blocked before please, exit %d", code)
	}
	out, code = bu("makeup", "--i-cant-do-this", "--please")
	if code != 0 || !strings.Contains(out, "we're back") {
		t.Fatalf("reunion: %d\n%s", code, out)
	}

	out, code = run(claudeShim, shimEnv, "--prompt", "hi")
	if code != 0 || !strings.Contains(out, "REAL_CLAUDE --prompt hi") {
		t.Fatalf("passthrough after makeup: %d\n%s", code, out)
	}

	makeupShim := filepath.Join(buHome, "bin", "makeup")
	out, _ = run(makeupShim, shimEnv)
	if !strings.Contains(out, "haven't even left") {
		t.Fatalf("makeup shim:\n%s", out)
	}

	if _, code = bu("for", "3m"); code != 0 {
		t.Fatal("second break")
	}
	out, _ = bu("makeup", "--the-build-is-on-fire")
	if !strings.Contains(out, "two hours") {
		t.Fatalf("emergency:\n%s", out)
	}
	out, code = run(claudeShim, shimEnv, "paused")
	if code != 0 || !strings.Contains(out, "REAL_CLAUDE paused") {
		t.Fatalf("pause passthrough: %d\n%s", code, out)
	}

	if _, code = bu("harden"); code != 0 {
		t.Fatal("harden")
	}
	hb, _ := os.ReadFile(hosts)
	if !bytes.Contains(hb, []byte("api.openai.com")) {
		t.Fatalf("hosts not locked:\n%s", hb)
	}
	if _, code = bu("soften"); code != 0 {
		t.Fatal("soften")
	}
	hb, _ = os.ReadFile(hosts)
	if bytes.Contains(hb, []byte("api.openai.com")) {
		t.Fatalf("hosts still locked:\n%s", hb)
	}

	if _, code = bu("uninstall"); code != 0 {
		t.Fatal("uninstall")
	}
	zshrc, _ = os.ReadFile(filepath.Join(home, ".zshrc"))
	if bytes.Contains(zshrc, []byte(">>> breakup >>>")) {
		t.Fatalf("hook remained:\n%s", zshrc)
	}
	if _, err := os.Stat(claudeShim); !os.IsNotExist(err) {
		t.Fatal("shim remained")
	}
}
