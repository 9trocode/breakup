package cli

import (
	"errors"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/nitrocode/breakup/internal/letter"
	"github.com/nitrocode/breakup/internal/render"
	"github.com/nitrocode/breakup/internal/state"
	"github.com/spf13/cobra"
)

func sayTitle(s *state.State) {
	render.Title(stdout, letter.Title(s))
}

func newNameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "name [name]",
		Short: "what you call her",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := state.Load()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				if ttyIn() {
					n, err := promptName()
					if err != nil {
						return err
					}
					return setName(s, n, nowFn())
				}
				sayTitle(s)
				render.Letter(stdout, letter.WhoAmI(s.Name))
				return nil
			}
			n, err := letter.NormalizeName(args[0])
			if err != nil {
				sayTitle(s)
				render.Letter(stdout, err.Error())
				return &silentErr{code: 1}
			}
			return setName(s, n, nowFn())
		},
	}
}

func setName(s *state.State, n string, now time.Time) error {
	if s.Name == n {
		sayTitle(s)
		render.Letter(stdout, letter.AlreadyNamed(n))
		return nil
	}
	if s.Name != "" && s.Effective(now) != state.Together {
		sayTitle(s)
		render.Letter(stdout, letter.RenameApart(s.Name))
		return nil
	}
	old := s.Name
	st, err := state.Update(func(cur *state.State) error {
		cur.Name = n
		return nil
	})
	if err != nil {
		return err
	}
	sayTitle(st)
	if old == "" {
		render.Letter(stdout, letter.Named(n))
		return nil
	}
	render.Letter(stdout, letter.Renamed(old, n))
	return nil
}

func resolveName(cur *state.State, flag string) (string, error) {
	if flag != "" {
		return letter.NormalizeName(flag)
	}
	if cur != nil && cur.Name != "" {
		return cur.Name, nil
	}
	if ttyIn() {
		return promptName()
	}
	return "", errNeedName
}

var errNeedName = errors.New("need name")

func promptName() (string, error) {
	var raw string
	input := huh.NewInput().
		Title("what do you call me.").
		CharLimit(16).
		Value(&raw).
		Validate(func(s string) error {
			_, err := letter.NormalizeName(s)
			return err
		})
	form := huh.NewForm(huh.NewGroup(input)).WithTheme(huh.ThemeBase())
	if err := form.Run(); err != nil {
		return "", errNeedName
	}
	return letter.NormalizeName(raw)
}

func needName() error {
	render.Title(stdout, "breakup")
	render.Letter(stdout, letter.NeedName())
	return &silentErr{code: 1}
}
