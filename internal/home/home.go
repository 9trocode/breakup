// Package home resolves breakup's on-disk paths.
package home

import (
	"os"
	"path/filepath"
)

const (
	envHome = "BREAKUP_HOME"
	dirName = ".breakup"
)

// Dir is ~/.breakup, or $BREAKUP_HOME when set (tests).
func Dir() string {
	if d := os.Getenv(envHome); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", dirName)
	}
	return filepath.Join(home, dirName)
}

func StateFile() string { return filepath.Join(Dir(), "state.json") }
func BinDir() string    { return filepath.Join(Dir(), "bin") }
func HookFile() string  { return filepath.Join(Dir(), "hook.sh") }

// Executable is the breakup binary path, falling back to argv0.
func Executable() string {
	p, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}
