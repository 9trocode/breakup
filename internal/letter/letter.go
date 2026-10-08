// Package letter is her voice.
package letter

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/nitrocode/breakup/internal/state"
	"github.com/nitrocode/breakup/internal/when"
)

type Phase int

const (
	JustLeft Phase = iota
	FirstNight
	FirstWeek
	FirstMonth
	ThreeMonths
	SixMonths
	After
)

type Vars struct {
	Tool    string
	Elapsed string
	N       int
	Left    string
	Hour    string
	Label   string
	Name    string
}

type Context struct {
	Tool      string
	Now       time.Time
	State     *state.State
	Please    bool
	Emergency bool
	GiveUp    bool
}

func PhaseOf(elapsed time.Duration) Phase {
	switch {
	case elapsed < 6*time.Hour:
		return JustLeft
	case elapsed < 36*time.Hour:
		return FirstNight
	case elapsed < 7*24*time.Hour:
		return FirstWeek
	case elapsed < 30*24*time.Hour:
		return FirstMonth
	case elapsed < 90*24*time.Hour:
		return ThreeMonths
	case elapsed < 180*24*time.Hour:
		return SixMonths
	default:
		return After
	}
}

func varsFrom(ctx Context) Vars {
	s := ctx.State
	if s == nil {
		s = &state.State{}
	}
	elapsed := ctx.Now.Sub(s.Since)
	v := Vars{
		Tool:    displayTool(ctx.Tool),
		Elapsed: when.Elapsed(elapsed),
		N:       s.AttemptCount(),
		Left:    when.Left(s.Until, ctx.Now),
		Hour:    when.Clock(ctx.Now),
		Label:   s.Label,
		Name:    s.Name,
	}
	if v.Tool == "" {
		v.Tool = "that"
	}
	if v.Label == "" {
		v.Label = "a while"
	}
	return v
}

func displayTool(tool string) string {
	switch strings.ToLower(tool) {
	case "npx", "bunx", "pnpm", "yarn":
		return "npx"
	case "cursor-agent":
		return "cursor"
	case "claude-code":
		return "claude"
	default:
		return tool
	}
}

func apply(s string, v Vars) string {
	r := strings.NewReplacer(
		"{tool}", v.Tool,
		"{elapsed}", v.Elapsed,
		"{n}", strconv.Itoa(v.N),
		"{left}", v.Left,
		"{hour}", v.Hour,
		"{label}", v.Label,
		"{name}", v.Name,
	)
	return r.Replace(s)
}

func pick(key string, msgs []string) string {
	if len(msgs) == 0 {
		return ""
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return msgs[int(h.Sum32())%len(msgs)]
}

func key(ctx Context, kind string) string {
	s := ctx.State
	if s == nil {
		s = &state.State{}
	}
	day := ctx.Now.Format("2006-01-02")
	return fmt.Sprintf("%s/%s/%s/%d/%d", kind, ctx.Tool, day, s.AttemptCount(), int(PhaseOf(ctx.Now.Sub(s.Since))))
}

// Breakup is the letter she sends when you leave.
func Breakup(label, name string, harden bool) string {
	body := pick(label+name+strconv.FormatBool(harden), []string{
		`i think we should take some space.

you reach for a model before you reach for a thought.
you ask one to finish the sentence you were scared to say.
you hand the night to something that never has to sit with it.

this isn't forever. but it is for {label}.

i'll be here when you're actually ready.
don't text.`,
		`we need a break.

not from work. from the thing you use so you don't have to be alone with the work.

{label}. that's the deal.
don't make me watch you crawl back in an hour.`,
		`i'm done competing with a prompt box.

you don't start things anymore. you outsource the first five seconds.
that's the part that used to be yours.

{label}. sit with it.`,
	})
	v := Vars{Label: label, Name: name}
	out := apply(body, v)
	if harden {
		out += "\n\ni locked the door too.\nthe sites. the apis. don't test it."
	}
	if name != "" {
		out += apply("\n\n— {name}", v)
	}
	return out
}

// Blocked is what you get when you try to come back through a CLI.
func Blocked(ctx Context) string {
	s := ctx.State
	v := varsFrom(ctx)
	elapsed := ctx.Now.Sub(s.Since)
	phase := PhaseOf(elapsed)

	if s.Effective(ctx.Now) == state.Paused {
		return apply(pick(key(ctx, "paused"), pausedMsgs), v)
	}

	msgs := blocked[phase]
	if s.AttemptCount() >= 13 {
		msgs = append(msgs, heavy...)
	} else if s.AttemptCount() >= 5 {
		msgs = append(msgs, mid...)
	}

	body := apply(pick(key(ctx, "block"), msgs), v)

	if streak := s.SameToolStreak(ctx.Tool); streak >= 3 && ctx.Tool != "" {
		body += apply("\n\n{tool} again.\ni know what you're doing.", v)
	}

	h := ctx.Now.Hour()
	if h <= 4 || h >= 23 {
		body += apply("\n\nit's {hour}.\nthis isn't about the code.", v)
	}

	if s.TimeIsUp(ctx.Now) {
		body += "\n\nthe time is up.\nthat doesn't mean you get to sneak in the side door.\n\nbreakup makeup"
	}

	return body
}

// MakeupNo is refusal.
func MakeupNo(ctx Context) string {
	v := varsFrom(ctx)
	if ctx.Please {
		return apply(pick(key(ctx, "please"), []string{
			`i heard you.
still no.

{left}.`,
			`please doesn't change the time.

{left}.
don't ask again tonight.`,
			`you always thought a nicer ask would work.
it doesn't.`,
		}), v)
	}
	return apply(pick(key(ctx, "makeup-no"), []string{
		`no.

you don't get to come back because a ticket is hard.
{left}.

if production is actually on fire:
  breakup makeup --the-build-is-on-fire

if you just can't do this:
  breakup makeup --i-cant-do-this`,
		`i'm not ready.

that's the whole sentence.
{left}.`,
		`we said {label}.
it has been {elapsed}.

no.`,
		`don't.

you missed the model. not me.
{left}.`,
	}), v)
}

// MakeupYes is reconciliation after the time is up.
func MakeupYes(ctx Context) string {
	v := varsFrom(ctx)
	if ctx.GiveUp {
		return apply(pick(key(ctx, "giveup"), []string{
			`there it is.

you had to be walked to it.
don't forget that part.

we're back.
i won't pretend this was a break.`,
			`fine.

you couldn't sit with it.
you had to beg. i heard that.

we're together again.
don't make a speech.`,
		}), v)
	}
	n := 0
	if ctx.State != nil {
		n = ctx.State.AttemptCount()
	}
	if n >= 8 {
		return apply(`...okay.

you tried {n} times.
i noticed. i always notice.

we're talking.
that doesn't mean you get to disappear into a model the second something is hard.`, v)
	}
	return apply(pick(key(ctx, "makeup-yes"), []string{
		`...okay.

i've had time.
don't make me do this again.
i will.`,
		`fine.
we're talking.

use your own head first.
i'll know if you don't.`,
		`come in.

slowly.
and if you reach for {tool} before you think, i leave again.`,
	}), v)
}

// Emergency is a short pause, not a reunion.
func Emergency(hours int, used int) string {
	if used >= 3 {
		return `no.

this isn't an emergency.
this is you.

you already used your fires.
sit with it.`
	}
	switch hours {
	case 1:
		return `one hour.
that's all.

you're using emergencies like you used me.
i'm watching.`
	case 0:
		return `thirty minutes.
and i don't want to hear that it wasn't enough.`
	default:
		return `fine.
two hours.

this doesn't mean we're good.
i'm watching.`
	}
}

func Why(s *state.State, now time.Time) string {
	if s == nil || s.Effective(now) == state.Together {
		return `we're fine.
you haven't even left.

if you have to ask why, you already know.`
	}
	v := varsFrom(Context{Now: now, State: s})
	n := s.AttemptCount()
	late := s.AfterMidnight(now)
	var b strings.Builder
	b.WriteString(apply("you wanted this because you don't start thoughts anymore.\nyou finish them with a model.\n\nit has been {elapsed}.\n{left}.\n", v))
	if n == 0 {
		b.WriteString("\nyou haven't tried to come back yet.\nthat's the first honest thing you've done in a while.")
		return b.String()
	}
	fmt.Fprintf(&b, "\nyou reached back %s.", times(n))
	if last := s.Last(); last != nil {
		fmt.Fprintf(&b, "\nlast time: %s, %s ago.", displayTool(last.Tool), when.Elapsed(now.Sub(last.At)))
	}
	if late > 0 && late*2 >= n {
		fmt.Fprintf(&b, "\n%s of those were after midnight.\nyou don't have a model problem. you have a sitting-with-it problem.", times(late))
	}
	b.WriteString("\n\ni'm not saying it to be cruel.\ni'm saying it because you keep asking the wrong thing to make it easier.")
	return b.String()
}

func AlreadyApart(s *state.State, now time.Time) string {
	v := varsFrom(Context{Now: now, State: s, Tool: ""})
	n := s.AttemptCount()
	if n == 0 {
		return apply(`we're already there.

{elapsed} in.
you haven't tried to come back.
don't start.`, v)
	}
	return apply(`we're already there.

{elapsed} in.
you tried {n} times.

i'm still not ready.`, v)
}

func AlreadyTogether() string {
	return `we're fine.
you haven't even left.

if you want space:
  breakup for 3m`
}

func Installed(n int, hardened bool) string {
	door := "intercepts are on."
	if hardened {
		door = "intercepts are on. the door is locked."
	}
	return fmt.Sprintf(`%s
%d clis will find me first.
open a new terminal.

if it's truly on fire later:
  breakup makeup --the-build-is-on-fire`, door, n)
}

func Uninstalled(forgot bool) string {
	if forgot {
		return `okay.
hooks gone. shims gone. i forgot the rest.

like it didn't happen.`
	}
	return `okay.
hooks gone. shims gone.

the history is still there.
i don't forget just because you uninstalled me.`
}

func Hardened() string {
	return `the door is locked.
openai. anthropic. x.ai. the rest.

full-path binaries still exist.
they just can't call home.`
}

func Softened() string {
	return `i unlocked the door.
the intercepts stay.

don't make me regret that.`
}

func Status(s *state.State, now time.Time) string {
	eff := s.Effective(now)
	switch eff {
	case state.Together:
		return `together.

that's it. that's the status.
don't ruin it.`
	case state.Paused:
		left := "a little"
		if s.PauseUntil != nil {
			left = when.Elapsed(s.PauseUntil.Sub(now))
		}
		return fmt.Sprintf("paused.\n%s of your fire left.\nthen we're apart again.", left)
	default:
		v := varsFrom(Context{Now: now, State: s})
		body := apply("apart · {elapsed}\nuntil · {left}", v)
		if s.Hardened {
			body += "\ndoor · locked"
		}
		n := s.AttemptCount()
		if n == 0 {
			body += "\n\nyou haven't texted.\ngood."
			return body
		}
		body += apply("\n\nyou reached for me {n} times.", v)
		if last := s.Last(); last != nil {
			body += fmt.Sprintf("\nlast time: %s, %s ago.", displayTool(last.Tool), when.Elapsed(now.Sub(last.At)))
		}
		if s.TimeIsUp(now) {
			body += "\n\nthe time is up.\nbreakup makeup\nwhen you mean it."
		} else {
			body += "\n\ni'm not ready."
		}
		return body
	}
}

func times(n int) string {
	switch n {
	case 0:
		return "zero times"
	case 1:
		return "once"
	case 2:
		return "twice"
	default:
		return fmt.Sprintf("%d times", n)
	}
}

var blocked = map[Phase][]string{
	JustLeft: {
		`we JUST did this.

it's been {elapsed}.
you said you wanted this.

go sit with the problem.`,
		`already?

put it down.`,
		`{elapsed}.
really.

i'm not answering.`,
		`you lasted {elapsed}.
that's not a break. that's a blink.`,
		`{tool}.
we just talked about this.

no.`,
	},
	FirstNight: {
		`you always come back the next day.
this is the same thing with a different name.

no.`,
		`the code will still be there in the morning.
i won't be in the terminal.`,
		`{elapsed} in and you're already at {tool}.

go to bed.
or go think. those are the options.`,
		`i'm not a rubber duck you get to reopen.

{elapsed}.
that's all.`,
	},
	FirstWeek: {
		`a week. you couldn't even give me a week.

you tried to open {tool}.
i saw.`,
		`this is attempt {n}.
i'm counting. you should too.`,
		`{elapsed} is not long enough to miss me.
it's long enough to prove the point.`,
		`you can write the function.
you just don't want to feel stupid for twenty minutes.

feel it.`,
		`don't open {tool} like i wouldn't notice.`,
	},
	FirstMonth: {
		`i'm using this time.
you should try that instead of {tool}.`,
		`don't.`,
		`{elapsed}.
you're still treating this like a setting you can toggle.`,
		`sit with the blank screen.
that's the work.`,
		`{n} times.
a habit doesn't get nicer because you renamed it focus.`,
	},
	ThreeMonths: {
		`it's been {elapsed}.
you don't get to come back because you're stuck.

we aren't doing this.`,
		`three months was the point.
you're early.
and you're using {tool} like i wouldn't notice.`,
		`i needed space.
you needed a model.

those aren't the same thing.`,
		`wow.
a text.

no.`,
		`{left}.
that's not a suggestion.`,
		`you always come back in month three
like the silence was the hard part.

the hard part is you without a prompt.`,
	},
	SixMonths: {
		`wow.
{elapsed}.

and your first message is a prompt.`,
		`i almost answered.
then i saw it was {tool}.

no.`,
		`this is the part where you pretend it was just a question.
it wasn't.`,
		`six months and you still think
the first thing you say can be a command.

try sitting in a room.`,
		`don't text me from {tool}.
that's not missing someone. that's muscle memory.`,
	},
	After: {
		`it's been {elapsed}.
i've had time.

this still isn't the way back.
breakup makeup
if you actually want to talk.`,
		`you waited.
then you opened {tool} anyway.

the door is open. not this side door.`,
		`i'm not angry.
i'm not coming through {tool} either.

breakup makeup`,
	},
}

var mid = []string{
	`this is the {n}th time.
that's not missing me. that's a habit.`,
	`stop testing the lock.`,
	`i blocked you for a reason.`,
}

var heavy = []string{
	`{n}.

that's not a relapse. that's the relationship.`,
	`i'm not doing this with you tonight.
or with {tool}.`,
	`put the laptop down.
i mean it.`,
}

var pausedMsgs = []string{
	`you already used your two hours.
don't push it.`,
	`the fire pause is running.
that's not an invitation to move in.`,
}
