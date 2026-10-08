// Package guard stands in front of AI CLIs.
package guard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nitrocode/breakup/internal/home"
	"github.com/nitrocode/breakup/internal/state"
	"golang.org/x/sys/unix"
)

var ErrBlocked = errors.New("blocked")

type Decision int

const (
	Pass Decision = iota
	PassPaused
	Block
)

func Decide(s *state.State, tool string, args []string, now time.Time) Decision {
	if s == nil {
		return Pass
	}
	if isWrapper(baseName(tool)) && !WrapperBlocks(tool, args) {
		return Pass
	}
	switch s.Effective(now) {
	case state.Together:
		return Pass
	case state.Paused:
		return PassPaused
	default:
		return Block
	}
}

// Intercept is what every shim runs.
// Blocked: caller should print a letter and exit 1.
// Passed: this process has been replaced via exec, or the real binary was not found.
func Intercept(tool string, args []string, now time.Time) (decision Decision, st *state.State, err error) {
	st, err = state.Load()
	if err != nil {
		return Pass, st, err
	}
	d := Decide(st, tool, args, now)
	if d == Block {
		st, err = state.Update(func(s *state.State) error {
			s.Record(tool, now)
			return nil
		})
		return Block, st, err
	}
	return d, st, ExecReal(tool, args)
}

// ExecReal finds the next binary on PATH after our shim dir and execs it.
func ExecReal(tool string, args []string) error {
	real, err := RealPath(tool)
	if err != nil {
		return err
	}
	argv := append([]string{real}, args...)
	return unix.Exec(real, argv, os.Environ())
}

func RealPath(tool string) (string, error) {
	name := baseName(tool)
	shimDir, err := filepath.Abs(home.BinDir())
	if err != nil {
		shimDir = home.BinDir()
	}
	self, _ := filepath.Abs(home.Executable())

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		if sameDir(abs, shimDir) {
			continue
		}
		cand := filepath.Join(abs, name)
		if !isExe(cand) {
			continue
		}
		resolved := cand
		if r, err := filepath.EvalSymlinks(cand); err == nil {
			resolved = r
		}
		if self != "" && sameFile(resolved, self) {
			continue
		}
		return cand, nil
	}
	return "", fmt.Errorf("%s: command not found", name)
}

func sameDir(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	return a == b
}

func sameFile(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func isExe(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}
	return fi.Mode()&0o111 != 0
}

const shimTemplate = `#!/bin/sh
# managed by breakup. do not edit.
exec "%s" intercept --tool "%s" -- "$@"
`

func InstallShims(bin string) (int, error) {
	if err := os.MkdirAll(home.BinDir(), 0o755); err != nil {
		return 0, err
	}
	if bin == "" {
		bin = home.Executable()
	}
	n := 0
	for _, name := range AllShimNames() {
		path := filepath.Join(home.BinDir(), name)
		body := fmt.Sprintf(shimTemplate, bin, name)
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func RemoveShims() error {
	dir := home.BinDir()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
	return nil
}

func ShimDirOnPATH() bool {
	shim, err := filepath.Abs(home.BinDir())
	if err != nil {
		shim = home.BinDir()
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		if sameDir(abs, shim) {
			return true
		}
	}
	return false
}
