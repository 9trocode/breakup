package letter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	rePlease  = regexp.MustCompile(`(?i)\bplease\b`)
	reBegging = regexp.MustCompile(`(?i)\bi['’]?m begging\b`)
	rePls     = regexp.MustCompile(`(?i)\bpl[sz]\b`)
	reSorry   = regexp.MustCompile(`(?i)\b(i['’]?m )?sorry\b`)
	reAI      = regexp.MustCompile(`(?i)\b(grok|claude|chatgpt|copilot|aider|cursor|codex|gemini|ollama|model|llm|prompt|openai|anthropic)\b`)
	reWork    = regexp.MustCompile(`(?i)\b(stuck|bug|error|deadline|ticket|prod|production|crash|broken|ship|deploy|incident|on fire|blocked)\b`)
	reNothing = regexp.MustCompile(`(?i)\b(nothing|idk|dunno|n/?a|no idea|i don['’]?t know)\b`)
	reVague   = regexp.MustCompile(`(?i)\b(everything|whatever|because|just because|i need to)\b`)
)

const MinGiveUpTries = 3

// IsBeg is the bar. "pls" is not the bar.
func IsBeg(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if rePls.MatchString(s) && !rePlease.MatchString(s) && !reBegging.MatchString(s) {
		return false
	}
	return rePlease.MatchString(s) || reBegging.MatchString(s)
}

func kindOf(answer string) string {
	a := strings.TrimSpace(answer)
	if a == "" {
		return "empty"
	}
	letters := 0
	for _, r := range a {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	if letters < 8 {
		return "shrug"
	}
	al := strings.ToLower(a)
	if IsBeg(al) {
		return "early-please"
	}
	if reAI.MatchString(al) {
		return "named-it"
	}
	if reWork.MatchString(al) {
		return "the-work"
	}
	if reNothing.MatchString(al) {
		return "nothing"
	}
	if reSorry.MatchString(al) {
		return "sorry"
	}
	if reVague.MatchString(al) {
		return "vague"
	}
	return "generic"
}

func throwback(answer string) string {
	s := strings.Join(strings.Fields(strings.TrimSpace(answer)), " ")
	if s == "" || len(s) > 42 {
		return ""
	}
	return s
}

// GrillOpen starts the interrogation. ends on "why."
func GrillOpen(ctx Context) string {
	v := varsFrom(ctx)
	n := 0
	if ctx.State != nil {
		n = ctx.State.AttemptCount()
	}
	body := apply(`you lasted {elapsed}.
{left} on the clock.

and you show up with --i-cant-do-this
like a white flag you didn't earn.`, v)
	if n > 0 {
		body += apply("\n\nyou already reached back {n} times.\nthis is just you asking out loud.", v)
	}
	body += "\n\nwhy."
	return body
}

func GrillRoastWhy(answer string, ctx Context) string {
	v := varsFrom(ctx)
	k := kindOf(answer)
	var roast string
	switch k {
	case "empty":
		roast = `you can't even say it.
that's not mysterious. that's you folding.`
	case "shrug":
		roast = roastShrug(answer, `that's not a reason. that's a shrug.`)
	case "named-it":
		roast = `you named the model.
that's the addiction talking.
you didn't even last long enough to be ashamed of it.`
	case "the-work":
		roast = apply(`that's the work.
that's the whole point of {label}.

you don't get to skip the part where it's hard
because the hard part has a compiler error.`, v)
	case "early-please":
		roast = `please already?
i asked why. you skipped to the apology
like a man who thinks manners are a cheat code.`
	case "sorry":
		roast = `i didn't ask for sorry.
i asked why.
those aren't the same sentence.`
	case "nothing":
		roast = `nothing.
so you felt the blank and came here.
of course you did.`
	case "vague":
		roast = roastShrug(answer, `that's fog. i asked for a reason.`)
	default:
		roast = apply(pick(key(ctx, "why-generic")+kindOf(answer), []string{
			`you lasted {elapsed} and that's what you came up with.`,
			`say it plainer. you want the thing you promised to put down.`,
			`that's a story you tell yourself so you don't have to sit still.`,
		}), v)
	}
	return roast + "\n\nwhat did you try before you opened this."
}

func GrillRoastTried(answer string, ctx Context) string {
	v := varsFrom(ctx)
	k := kindOf(answer)
	var roast string
	switch k {
	case "empty", "nothing", "shrug":
		roast = `so nothing.
you sat down, felt stupid for ten seconds,
and looked for a model to hold your hand.

that's not stuck. that's a reflex.`
	case "named-it":
		roast = `you tried the other one.
that's not trying. that's switching dealers.`
	case "the-work":
		roast = `you touched the problem
and then came here the second it didn't yield.

that used to be called the job.`
	case "early-please":
		roast = `save please.
you haven't earned the asking yet.`
	case "vague":
		roast = `everything is not a thing you did.
name one. you can't. that's why you're here.`
	default:
		roast = apply(pick(key(ctx, "tried-generic"), []string{
			`and then you still came back.
so you didn't try long.`,
			`that lasted less than {elapsed} of a break.
don't dress it up as effort.`,
			`you did the smallest possible thing
so you could say you tried.
i can tell.`,
		}), v)
	}
	return roast + "\n\nsay it."
}

func GrillBegNo(answer string, ctx Context) string {
	v := varsFrom(ctx)
	k := kindOf(answer)
	switch {
	case rePls.MatchString(answer) && !IsBeg(answer):
		return `pls.

i'm not a search box.
come back when you can spell the word.`
	case k == "empty" || k == "shrug":
		return apply(`that's not asking.
that's you waiting for me to do it for you.
again.

come back when you can say please
without making it a commit message.

{left}.`, v)
	case reSorry.MatchString(answer) && !IsBeg(answer):
		return `sorry isn't asking.
sorry is you hoping i'll do the rest.

the word is please.
say it next time. mean it.`
	case k == "the-work" || k == "named-it" || k == "vague" || k == "generic":
		return apply(`i heard the speech.
i didn't hear please.

you can explain a bug to a wall.
you don't get to explain your way back in.

{left}.`, v)
	default:
		return apply(`that's not begging.
that's negotiating.

i don't negotiate with a flag.

{left}.`, v)
	}
}

func GrillFold(ctx Context, round int) string {
	v := varsFrom(ctx)
	switch round {
	case 1:
		return apply(`you asked to come back
and then went quiet on why.

i'm not doing this if you won't.
{left}.`, v)
	case 2:
		return apply(`you couldn't name a single thing you tried.
because you didn't.

sit with that.
{left}.`, v)
	default:
		return apply(`you couldn't even say it.
that's the whole problem, boxed.

{left}.`, v)
	}
}

// GiveUpTurn is the non-interactive version: 3 refusals, then --please.
// try is 1-based, already incremented.
func GiveUpTurn(try int, please bool, ctx Context) (body string, allowed bool) {
	v := varsFrom(ctx)
	v.N = try
	switch {
	case try == 1:
		msg := `no.

you don't get a flag to skip the feeling.
that's a tantrum with a longer name.

that's 1 of 3.`
		if please {
			msg = `please already?
you haven't even sat in it.

that's 1 of 3.
don't skip to the apology.`
		}
		return apply(msg+"\n\ncome back. i'll still say no.\n{left}.", v), false
	case try == 2:
		msg := `still no.

repeating the command isn't persistence.
it's just you, louder.

that's 2 of 3.`
		if please {
			msg = `you brought please to round two
like i wouldn't notice you trying to skip round three.

that's 2 of 3.
i'm counting.`
		}
		return apply(msg+"\n\n{left}.", v), false
	case try >= MinGiveUpTries && !please:
		return apply(`that's 3.

i grilled you.
you still didn't beg.

add --please
and mean it.

  breakup makeup --i-cant-do-this --please

{left}.`, v), false
	default:
		return apply(`there it is.

three times, then please.
you had to be walked to it.

we're back.
i won't pretend this was a break.`, v), true
	}
}

func roastShrug(answer, tail string) string {
	if s := throwback(answer); s != "" {
		return fmt.Sprintf("%s\n\n%s", s, tail)
	}
	return tail
}
