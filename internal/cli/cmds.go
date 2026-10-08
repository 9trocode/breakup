package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nitrocode/breakup/internal/block"
	"github.com/nitrocode/breakup/internal/guard"
	"github.com/nitrocode/breakup/internal/home"
	"github.com/nitrocode/breakup/internal/hook"
	"github.com/nitrocode/breakup/internal/letter"
	"github.com/nitrocode/breakup/internal/render"
	"github.com/nitrocode/breakup/internal/state"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "how long, how many times you tried",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := state.Load()
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(s)
			}
			sayTitle(s)
			render.Letter(stdout, letter.Status(s, nowFn()))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "raw state")
	return cmd
}

func newWhyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "why",
		Short: "she tells you why",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := state.Load()
			if err != nil {
				return err
			}
			sayTitle(s)
			render.Letter(stdout, letter.Why(s, nowFn()))
			return nil
		},
	}
}

func newMakeupCmd() *cobra.Command {
	var (
		please    bool
		onFire    bool
		giveUp    bool
		forceTime bool
	)
	cmd := &cobra.Command{
		Use:     "makeup",
		Aliases: []string{"together", "reconcile"},
		Short:   "try to get back together",
		RunE: func(cmd *cobra.Command, args []string) error {
			now := nowFn()
			s, err := state.Load()
			if err != nil {
				return err
			}
			if s.Effective(now) == state.Together {
				sayTitle(s)
				render.Letter(stdout, letter.AlreadyTogether())
				return nil
			}

			ctx := letter.Context{Now: now, State: s, Please: please, Emergency: onFire, GiveUp: giveUp}

			if onFire {
				return emergencyPause(s, now)
			}
			if giveUp {
				return runGiveUp(s, now, please)
			}
			if forceTime || s.TimeIsUp(now) {
				return makeUp(s, now, ctx)
			}
			sayTitle(s)
			render.Letter(stdout, letter.MakeupNo(ctx))
			return nil
		},
	}
	cmd.Flags().BoolVar(&please, "please", false, "beg. required after she grills you.")
	cmd.Flags().BoolVar(&onFire, "the-build-is-on-fire", false, "pause the break for a short emergency")
	cmd.Flags().BoolVar(&giveUp, "i-cant-do-this", false, "ask to end it early. she will grill you. then you beg.")
	cmd.Flags().BoolVar(&forceTime, "i-know", false, "end the break as if the time were up")
	_ = cmd.Flags().MarkHidden("i-know")
	return cmd
}

func emergencyPause(s *state.State, now time.Time) error {
	used := s.Emergencies
	hours := 2
	switch {
	case used >= 3:
		sayTitle(s)
		render.Letter(stdout, letter.Emergency(hours, used))
		return nil
	case used == 1:
		hours = 1
	case used >= 2:
		hours = 0 // 30 min
	}
	d := 2 * time.Hour
	switch hours {
	case 1:
		d = time.Hour
	case 0:
		d = 30 * time.Minute
	}
	until := now.Add(d)
	_, err := state.Update(func(cur *state.State) error {
		cur.Status = state.Paused
		cur.PauseUntil = &until
		cur.Emergencies++
		return nil
	})
	if err != nil {
		return err
	}
	sayTitle(s)
	render.Letter(stdout, letter.Emergency(hours, used))
	render.Meta(stdout, fmt.Sprintf("paused until %s", until.Format(time.Kitchen)))
	return nil
}

func makeUp(s *state.State, now time.Time, ctx letter.Context) error {
	if s.Hardened {
		if block.NeedsRoot() {
			_ = block.ReexecSudo("soften", "--no-state")
		} else {
			_ = block.Remove()
		}
	}
	_, err := state.Update(func(cur *state.State) error {
		cur.Status = state.Together
		cur.Hardened = false
		cur.PauseUntil = nil
		cur.Until = nil
		cur.GiveUpTries = 0
		return nil
	})
	if err != nil {
		return err
	}
	sayTitle(s)
	render.Letter(stdout, letter.MakeupYes(ctx))
	return nil
}

func newHardenCmd() *cobra.Command {
	var noState bool
	cmd := &cobra.Command{
		Use:   "harden",
		Short: "lock LLM API hosts in /etc/hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if block.NeedsRoot() && os.Geteuid() != 0 {
				return block.ReexecSudo("harden")
			}
			if err := block.Apply(); err != nil {
				return err
			}
			if !noState {
				_, _ = state.Update(func(s *state.State) error {
					s.Hardened = true
					return nil
				})
			}
			s, _ := state.Load()
			sayTitle(s)
			render.Letter(stdout, letter.Hardened())
			return nil
		},
	}
	cmd.Flags().BoolVar(&noState, "no-state", false, "only edit hosts")
	_ = cmd.Flags().MarkHidden("no-state")
	return cmd
}

func newSoftenCmd() *cobra.Command {
	var noState bool
	cmd := &cobra.Command{
		Use:   "soften",
		Short: "unlock the hosts file. intercepts stay.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if block.NeedsRoot() && os.Geteuid() != 0 {
				return block.ReexecSudo("soften")
			}
			if err := block.Remove(); err != nil {
				return err
			}
			if !noState {
				_, _ = state.Update(func(s *state.State) error {
					s.Hardened = false
					return nil
				})
			}
			s, _ := state.Load()
			sayTitle(s)
			render.Letter(stdout, letter.Softened())
			return nil
		},
	}
	cmd.Flags().BoolVar(&noState, "no-state", false, "only edit hosts")
	_ = cmd.Flags().MarkHidden("no-state")
	return cmd
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "install PATH shims and a shell hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := guard.InstallShims(home.Executable())
			if err != nil {
				return err
			}
			rc, err := hook.Install()
			if err != nil {
				return err
			}
			_, _ = state.Update(func(s *state.State) error {
				s.Installed = true
				return nil
			})
			s, _ := state.Load()
			sayTitle(s)
			render.Letter(stdout, fmt.Sprintf("i'm in.\n%d intercepts.\nhook in %s.\n\nopen a new terminal.", n, rc))
			if !guard.ShimDirOnPATH() {
				render.Meta(stdout, "this shell is old:\n  source "+home.HookFile())
			}
			return nil
		},
	}
}

func newUninstallCmd() *cobra.Command {
	var forget bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "remove shims and the shell hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _ := state.Load()
			if s != nil && s.Hardened {
				if block.NeedsRoot() && os.Geteuid() != 0 {
					_ = block.ReexecSudo("soften", "--no-state")
				} else {
					_ = block.Remove()
				}
			}
			_ = guard.RemoveShims()
			_ = hook.Uninstall()
			if forget {
				_ = os.Remove(home.StateFile())
				render.Title(stdout, "breakup")
			} else {
				_, _ = state.Update(func(cur *state.State) error {
					cur.Status = state.Together
					cur.Hardened = false
					cur.PauseUntil = nil
					cur.Installed = false
					return nil
				})
				sayTitle(s)
			}
			render.Letter(stdout, letter.Uninstalled(forget))
			return nil
		},
	}
	cmd.Flags().BoolVar(&forget, "forget", false, "delete the history too")
	return cmd
}

func newInterceptCmd() *cobra.Command {
	var tool string
	cmd := &cobra.Command{
		Use:    "intercept",
		Hidden: true,
		Short:  "internal: run from a shim",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tool == "" {
				return fmt.Errorf("missing --tool")
			}
			// makeup shim
			if tool == "makeup" {
				root := newRoot()
				root.SetArgs(append([]string{"makeup"}, args...))
				root.SetOut(stdout)
				root.SetErr(stderr)
				return root.Execute()
			}
			now := nowFn()
			d, st, err := guard.Intercept(tool, args, now)
			if d == guard.Block {
				sayTitle(st)
				render.Letter(stdout, letter.Blocked(letter.Context{
					Tool:  tool,
					Now:   now,
					State: st,
				}))
				return guard.ErrBlocked
			}
			if err != nil {
				// pass-through failed (not found). mimic a shell.
				fmt.Fprintf(stderr, "breakup: %v\n", err)
				return &silentErr{code: 127}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tool, "tool", "", "shim name")
	return cmd
}
