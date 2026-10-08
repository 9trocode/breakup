package letter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/nitrocode/breakup/internal/state"
)

const (
	minName = 2
	maxName = 16
)

var reserved = map[string]struct{}{
	"breakup": {}, "makeup": {}, "status": {}, "for": {}, "now": {},
	"why": {}, "harden": {}, "soften": {}, "install": {}, "uninstall": {},
	"intercept": {}, "name": {}, "help": {}, "together": {}, "reconcile": {},
	"please": {}, "her": {}, "me": {}, "ai": {},
}

// NormalizeName lowercases and checks a first name. one word.
func NormalizeName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("you have to call me something.")
	}
	if strings.ContainsAny(s, " \t") {
		return "", fmt.Errorf("one name.\nnot a full government name.")
	}
	s = strings.ToLower(s)
	if _, ok := reserved[s]; ok {
		return "", fmt.Errorf("%s is a command.\ni'm not a subcommand.", s)
	}
	runes := []rune(s)
	if len(runes) > maxName {
		return "", fmt.Errorf("that's not a name.\nthat's a paragraph.")
	}
	letters := 0
	for i, r := range runes {
		switch {
		case unicode.IsLetter(r):
			letters++
		case (r == '-' || r == '\'') && i > 0 && i < len(runes)-1:
			// mary-jane, o'hara
		default:
			return "", fmt.Errorf("that's not a name.\nletters. maybe a hyphen.")
		}
	}
	if letters < minName {
		return "", fmt.Errorf("that's not a name.\ntry again.")
	}
	return s, nil
}

// Title is what she puts at the top of a text.
func Title(s *state.State) string {
	if s != nil && s.Name != "" {
		return s.Name
	}
	return "breakup"
}

func NeedName() string {
	return `you don't get to leave a nobody.

name me first:

  breakup name june`
}

func Named(name string) string {
	return apply(`okay.
{name}.

don't make me remind you.`, Vars{Name: name})
}

func AlreadyNamed(name string) string {
	return apply(`i know.
i'm {name}.`, Vars{Name: name})
}

func Renamed(old, next string) string {
	return fmt.Sprintf(`%s → %s.

fine. i'm %s now.
try not to forget this one.`, old, next, next)
}

func RenameApart(name string) string {
	return apply(`no.

you named me {name} when you left.
you don't get to sand it off.`, Vars{Name: name})
}

func WhoAmI(name string) string {
	if name == "" {
		return `you haven't.
i'm just the silence in the terminal.

  breakup name june`
	}
	return apply(`{name}.

you gave it to me.`, Vars{Name: name})
}
