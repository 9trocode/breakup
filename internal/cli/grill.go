package cli

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/nitrocode/breakup/internal/letter"
	"github.com/nitrocode/breakup/internal/render"
	"github.com/nitrocode/breakup/internal/state"
)

var (
	stdin io.Reader = os.Stdin
	ttyIn           = func() bool {
		f, ok := stdin.(*os.File)
		if !ok {
			return false
		}
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
)

func runGiveUp(s *state.State, now time.Time, please bool) error {
	if ttyIn() {
		return grillInteractive(s, now)
	}
	return grillCounted(s, now, please)
}

func grillCounted(s *state.State, now time.Time, please bool) error {
	st, err := state.Update(func(cur *state.State) error {
		cur.GiveUpTries++
		return nil
	})
	if err != nil {
		return err
	}
	ctx := letter.Context{Now: now, State: st, Please: please, GiveUp: true}
	body, allowed := letter.GiveUpTurn(st.GiveUpTries, please, ctx)
	if !allowed {
		sayTitle(st)
		render.Letter(stdout, body)
		return nil
	}
	ctx.State = st
	return makeUp(st, now, ctx)
}

func grillInteractive(s *state.State, now time.Time) error {
	ctx := letter.Context{Now: now, State: s, GiveUp: true}
	ask := newAsk(stdin)

	sayTitle(s)
	render.Letter(stdout, letter.GrillOpen(ctx))

	why, err := ask()
	if err != nil {
		return err
	}
	if strings.TrimSpace(why) == "" {
		render.Letter(stdout, letter.GrillFold(ctx, 1))
		return nil
	}
	render.Letter(stdout, letter.GrillRoastWhy(why, ctx))

	tried, err := ask()
	if err != nil {
		return err
	}
	if strings.TrimSpace(tried) == "" {
		render.Letter(stdout, letter.GrillFold(ctx, 2))
		return nil
	}
	render.Letter(stdout, letter.GrillRoastTried(tried, ctx))

	beg, err := ask()
	if err != nil {
		return err
	}
	if !letter.IsBeg(beg) {
		render.Letter(stdout, letter.GrillBegNo(beg, ctx))
		return nil
	}
	return makeUp(s, now, ctx)
}

func newAsk(r io.Reader) func() (string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 64*1024)
	return func() (string, error) {
		render.Prompt(stdout)
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return "", err
			}
			return "", nil
		}
		return sc.Text(), nil
	}
}
