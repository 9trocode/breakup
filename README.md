# breakup

take space from your AI.

a local CLI that temporarily blocks LLM CLIs and, if you ask, their APIs.
when you reach for them anyway, you get the text you'd get if you
messaged someone three months too soon.

some breaks are needed. especially with this.

## install

```bash
go install github.com/nitrocode/breakup@latest
```

or from this repo:

```bash
make install
```

then:

```bash
breakup name june
breakup for 3m
```

that writes shims to `~/.breakup/bin`, prepends them to PATH from your
shell rc, and starts the break. **open a new terminal.**

you name her. she won't pick it. after that, every blocked `claude` and
`grok` opens with her name at the top — not `breakup`.

## use it

```bash
breakup name june      # you have to call her something
breakup                # interactive. default is 3 months.
breakup now            # 3 months, no conversation
breakup for 8h         # a work block
breakup for 7d
breakup for 3m         # the classic
breakup for 6m
breakup for tonight
breakup for until      # until you say so

breakup --harden       # also lock LLM APIs in /etc/hosts (sudo)
breakup --close-the-apps   # quit Cursor / Claude / ChatGPT if they're open

breakup status
breakup why
```

`m` means **months**. that's the metaphor. minutes are `15min`.

## coming back

```bash
breakup makeup
```

if the time isn't up, she says no.

```bash
breakup makeup --the-build-is-on-fire   # pause. 2 hours, then 1, then 30 minutes.
breakup makeup --i-cant-do-this         # ask to end it early. she grills you three times. then you beg.
makeup                                  # same as breakup makeup, after install
```

`--i-cant-do-this` is not a skip. in a real terminal she asks why, what you tried,
and makes you say please. from a script she refuses twice, demands `--please` on
the third, and only then lets you in.

## what actually gets blocked

**CLIs** (PATH shims): `claude`, `grok`, `cursor`, `codex`, `aider`,
`gemini`, `ollama`, `llm`, `goose`, `opencode`, `amp`, `npx`/`bunx`
when they launch those tools, and a list of the usual suspects.

**APIs** (optional, `breakup harden`): `api.openai.com`,
`api.anthropic.com`, `api.x.ai`, cursor, copilot, groq, openrouter,
chatgpt.com, claude.ai, and the rest of the usual doorways.

a full path to a binary still exists. without `harden` it can still
call home. with `harden` it can't.

this cannot close a tab you already have open. quit the app. that's
part of it.

## uninstall

```bash
breakup uninstall
breakup uninstall --forget   # delete the history too
```

## how

state lives in `~/.breakup/state.json`.
shims live in `~/.breakup/bin` and sit in front of PATH.
the emergency hatch always exists — she just won't make it easy.
locking you out with no door would be a different product.
# breakup
