package guard

import "strings"

// CLIs we stand in front of. denylist, not a moral inventory.
var CLIs = []string{
	"claude",
	"claude-code",
	"grok",
	"cursor",
	"cursor-agent",
	"codex",
	"aider",
	"gemini",
	"llm",
	"sgpt",
	"ollama",
	"copilot",
	"openai",
	"aichat",
	"mods",
	"goose",
	"crush",
	"opencode",
	"amp",
	"continue",
	"fabric",
	"chatgpt",
	"windsurf",
	"cline",
	"cody",
	"tabby",
	"droid",
	"openhands",
	"aider-chat",
	"xai",
	"groq",
	"together",
	"perplexity",
	"phind",
	"qwen",
	"ollama-chat",
	"codeium",
	"supercode",
	"aider-install",
}

// Wrappers that can launch those CLIs.
var Wrappers = []string{
	"npx",
	"bunx",
	"pnpx",
}

var npxPackages = []string{
	"@anthropic-ai/claude-code",
	"claude-code",
	"@google/gemini-cli",
	"@openai/codex",
	"aider-chat",
	"open-interpreter",
	"@github/copilot",
	"@continue.dev/cli",
	"opencode-ai",
}

func IsCLI(name string) bool {
	base := baseName(name)
	for _, t := range CLIs {
		if base == t {
			return true
		}
	}
	for _, t := range Wrappers {
		if base == t {
			return true
		}
	}
	return false
}

func WrapperBlocks(tool string, args []string) bool {
	base := baseName(tool)
	switch base {
	case "npx", "bunx", "pnpx":
		blob := strings.ToLower(strings.Join(args, " "))
		for _, p := range npxPackages {
			if strings.Contains(blob, strings.ToLower(p)) {
				return true
			}
		}
		// bare names people pass to npx
		for _, t := range []string{"claude", "aider", "codex", "gemini"} {
			if containsToken(args, t) {
				return true
			}
		}
		return false
	default:
		return IsCLI(base) && !isWrapper(base)
	}
}

func isWrapper(name string) bool {
	for _, w := range Wrappers {
		if name == w {
			return true
		}
	}
	return false
}

func containsToken(args []string, tok string) bool {
	for _, a := range args {
		if strings.EqualFold(a, tok) {
			return true
		}
	}
	return false
}

func baseName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndexAny(name, "/\\"); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func AllShimNames() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, t := range CLIs {
		add(t)
	}
	for _, t := range Wrappers {
		add(t)
	}
	add("makeup")
	return out
}
