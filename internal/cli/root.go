package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/nitrocode/breakup/internal/block"
	"github.com/nitrocode/breakup/internal/guard"
	"github.com/nitrocode/breakup/internal/home"
	"github.com/nitrocode/breakup/internal/hook"
	"github.com/nitrocode/breakup/internal/letter"
	"github.com/nitrocode/breakup/internal/render"
	"github.com/nitrocode/breakup/internal/state"
	"github.com/nitrocode/breakup/internal/when"
	"github.com/spf13/cobra"
)

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
	nowFn            = time.Now
)

func Run() int {
	cmd := newRoot()
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, guard.ErrBlocked) {
			return 1
		}
		var silent *silentErr
		if errors.As(err, &silent) {
			return silent.code
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type silentErr struct {
	code int
}

func (s *silentErr) Error() string { return "" }

func newRoot() *cobra.Command {
	var (
		forFlag    string
		hardenFlag bool
		again      bool
		quitApps   bool
		herName    string
	)

	cmd := &cobra.Command{
		Use:           "breakup",
		Short:         "take space from your AI",
		Long:          "breakup temporarily blocks LLM CLIs and, if you ask, their APIs.\nwhen you try to come back, she has something to say.\nyou have to name her first.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       "0.1.0",
		Example:       "  breakup name june\n  breakup for 3m\n  breakup makeup --the-build-is-on-fire",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startBreak(forFlag, hardenFlag, again, quitApps, herName)
		},
	}
	cmd.CompletionOptions.DisableDefaultCmd = true

	cmd.Flags().StringVar(&forFlag, "for", "", "how long: 8h, 7d, 3m, 6m, tonight, until")
	cmd.Flags().BoolVar(&again, "again", false, "reset the clock if you're already apart")
	cmd.PersistentFlags().BoolVar(&hardenFlag, "harden", false, "also lock LLM API hosts in /etc/hosts")
	cmd.PersistentFlags().BoolVar(&quitApps, "close-the-apps", false, "quit Cursor, Claude, ChatGPT if they're open (macOS)")
	cmd.PersistentFlags().StringVar(&herName, "name", "", "what you call her")

	cmd.AddCommand(
		newForCmd(&hardenFlag, &quitApps, &herName),
		newNowCmd(&hardenFlag, &quitApps, &herName),
		newNameCmd(),
		newStatusCmd(),
		newWhyCmd(),
		newMakeupCmd(),
		newHardenCmd(),
		newSoftenCmd(),
		newInstallCmd(),
		newUninstallCmd(),
		newInterceptCmd(),
	)
	return cmd
}

func newForCmd(harden *bool, quitApps *bool, herName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "for [duration]",
		Short: "leave for a while",
		Long:  "leave for a while.\n\nm means months. that's the metaphor.\nminutes are 15min. hours are 8h. tonight is until 6am.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return startBreak(args[0], *harden, false, *quitApps, *herName)
		},
	}
}

func newNowCmd(harden *bool, quitApps *bool, herName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "now",
		Short: "leave for 3 months, no conversation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startBreak("3m", *harden, false, *quitApps, *herName)
		},
	}
}

func startBreak(duration string, harden, again, quitApps bool, nameFlag string) error {
	now := nowFn()
	cur, err := state.Load()
	if err != nil {
		return err
	}
	if cur.Effective(now) != state.Together && !again {
		sayTitle(cur)
		render.Letter(stdout, letter.AlreadyApart(cur, now))
		return nil
	}

	name, err := resolveName(cur, nameFlag)
	if errors.Is(err, errNeedName) {
		return needName()
	}
	if err != nil {
		sayTitle(cur)
		render.Letter(stdout, err.Error())
		return &silentErr{code: 1}
	}

	if duration == "" {
		d, h, err := promptBreak(harden)
		if err != nil {
			return err
		}
		duration = d
		if h {
			harden = true
		}
	}

	until, label, err := when.Parse(duration, now)
	if err != nil {
		return err
	}

	if err := ensureInstalled(); err != nil {
		return err
	}

	if harden {
		if err := doHarden(); err != nil {
			render.Meta(stderr, err.Error())
			harden = false
		}
	}

	st, err := state.Update(func(s *state.State) error {
		s.Status = state.Apart
		s.Since = now
		s.Until = until
		s.Label = label
		s.Name = name
		s.Hardened = harden
		s.PauseUntil = nil
		s.Attempts = nil
		s.GiveUpTries = 0
		s.Installed = true
		s.WalkedAway++
		return nil
	})
	if err != nil {
		return err
	}

	var closed []string
	if quitApps {
		closed = block.MacQuitApps()
	}

	sayTitle(st)
	render.Letter(stdout, letter.Breakup(st.Label, st.Name, st.Hardened))
	n := len(guard.AllShimNames())
	render.Meta(stdout, letter.Installed(n, st.Hardened))
	if !guard.ShimDirOnPATH() {
		render.Meta(stdout, "this shell is old. open a new one, or:\n  source "+home.HookFile())
	}
	if len(closed) > 0 {
		render.Meta(stdout, "i closed: "+join(closed))
	}
	return nil
}

func promptBreak(hardenAlready bool) (duration string, harden bool, err error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return "3m", hardenAlready, nil
	}
	duration = "3m"
	harden = hardenAlready
	sel := huh.NewSelect[string]().
		Title("how long.").
		Options(
			huh.NewOption("just tonight", "tonight"),
			huh.NewOption("8 hours", "8h"),
			huh.NewOption("a week", "7d"),
			huh.NewOption("3 months", "3m"),
			huh.NewOption("6 months", "6m"),
			huh.NewOption("until i say so", "until"),
		).
		Value(&duration)

	form := huh.NewForm(huh.NewGroup(sel))
	if !hardenAlready {
		form = huh.NewForm(huh.NewGroup(
			sel,
			huh.NewConfirm().
				Title("lock the door? (block LLM APIs — needs sudo)").
				Value(&harden),
		))
	}
	form.WithTheme(huh.ThemeBase())
	if err := form.Run(); err != nil {
		return "3m", hardenAlready, nil
	}
	return duration, harden, nil
}

func ensureInstalled() error {
	if _, err := guard.InstallShims(home.Executable()); err != nil {
		return err
	}
	_, err := hook.Install()
	return err
}

func join(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += ", " + ss[i]
	}
	return out
}

func doHarden() error {
	if block.NeedsRoot() {
		if err := block.ReexecSudo("harden", "--no-state"); err != nil {
			return fmt.Errorf("couldn't lock the door: %w", err)
		}
		return nil
	}
	return block.Apply()
}
