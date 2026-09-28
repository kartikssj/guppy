# guppy

A fast, keyboard-driven terminal UI for Google Tasks.

```
 ▏                                ▏
 ▏   Lists (2)                    ▏   Personal                       2/4
 ▏                                ▏
 ▏    Personal                    ▏   ☑  Buy milk
 ▏    Work                        ▏   ☐  Write report
 ▏    Shopping                    ▏     └ ☐  Draft outline
 ▏                                ▏   ☐  Call mom

  tab switch · space toggle · a add · A subtask · ? help · q quit
```

Borderless panels with background fills and a thin accent bar marking the
focused pane, themed after [opencode](https://opencode.ai)'s dark palette.

## Features

- Browse all your task lists side-by-side with their tasks
- Subtasks, rendered as an indented tree
- Complete/reopen, add, rename, delete tasks (deleting a parent deletes its subtasks)
- Everything is keyboard-driven; changes sync immediately

## Install

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/kartikssj/guppy/main/install.sh | sh
```

The script downloads the latest release, verifies its SHA-256 checksum, and
installs to `~/.local/bin` (override with `GUPPY_INSTALL_DIR`, pin a version
with `GUPPY_VERSION`). Prefer to inspect it first? Download the script, read
it, then run it.

Or download a prebuilt binary manually from [Releases](../../releases) — these
ship with guppy's shared OAuth credentials and work out of the box. On macOS a
browser-downloaded binary is quarantined by Gatekeeper; allow it with:

```sh
xattr -d com.apple.quarantine guppy
```

Or build from source:

```sh
git clone <this repo> && cd guppy
go build -o guppy .
```

Or install into your `$GOBIN`:

```sh
go install .
```

Source builds embed no OAuth credentials; set up your own first (see
[Bring your own Google project](#bring-your-own-google-project-advanced)).

## Quick start

Just run it:

```sh
./guppy
```

On first run your browser opens to Google's sign-in page. Click **Allow**,
come back to the terminal, and your tasks are there. The token is cached in
`~/.config/guppy/` and refreshes automatically, so you only do this once.

To sign in with a different account, run `guppy --reauth`.

## Keys

| Key | Action |
|---|---|
| `↑`/`↓` (or `k`/`j`) | Move cursor |
| `tab`, `←`/`→` | Switch pane |
| `enter` | Open the selected list |
| `space` | Toggle task complete |
| `a` | Add task |
| `A` | Add subtask under selected task |
| `e` | Rename task |
| `d` then `y` | Delete task |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

While typing a task title: `enter` commits, `esc` cancels.

## Bring your own Google project (advanced)

Release binaries ship with shared OAuth client credentials so they work out
of the box. A desktop app can't keep a client secret confidential (see
[RFC 8252](https://oauth.net/2/native-apps/)), so these are public by design
and the flow is protected by PKCE. (They're injected into release builds via
`-ldflags -X` rather than committed, because GitHub's push protection blocks
pushes containing them.) If you'd rather use your own Google Cloud project —
your own quota, no unverified-app warning — do this once:

1. Go to [Google Cloud Console](https://console.cloud.google.com/) and create a project.
2. **APIs & Services → Library** → enable **Google Tasks API**.
3. **APIs & Services → OAuth consent screen** → choose **External**, fill in
   the required fields (no scopes need to be added here), publish or add
   yourself as a test user.
4. **APIs & Services → Credentials → Create Credentials → OAuth client ID**
   → application type **Desktop app** → download the JSON.
5. Either save it as `~/.config/guppy/credentials.json`, point
   `GUPPY_CREDENTIALS` at it, or set `GUPPY_CLIENT_ID` / `GUPPY_CLIENT_SECRET`.

guppy picks credentials in this order: env vars → `credentials.json` →
embedded (release binaries only).

## Files

| Path | Contents |
|---|---|
| `~/.config/guppy/token.json` | Cached OAuth token (mode `0600`) |
| `~/.config/guppy/credentials.json` | Optional BYO OAuth client config |

`$XDG_CONFIG_HOME` is respected when set.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

The TUI talks to Google Tasks through the `internal/gtasks.Service`
interface, so all UI logic is tested against an in-memory fake; the API
client itself is tested against an `httptest` server.

See [DESIGN.md](DESIGN.md) for the architecture and the reasoning behind it,
and [AGENTS.md](AGENTS.md) for contributor/agent guidelines.

## Legal

- [Privacy Policy](PRIVACY.md)
- [Terms of Service](TERMS.md)

## Roadmap

Due dates, search, hiding/clearing completed tasks, multiple accounts.

