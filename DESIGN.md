# DESIGN.md

Architecture and design decisions for guppy, a terminal UI for Google Tasks.
Read this before making non-trivial changes.

## Stack

| Layer | Choice | Notes |
|---|---|---|
| Language | Go 1.26 | Single static binary |
| TUI framework | `charm.land/bubbletea/v2` | **v2 API** — see "Bubble Tea v2" below |
| Components | `charm.land/bubbles/v2` (textinput only) | |
| Styling | `charm.land/lipgloss/v2` + `github.com/charmbracelet/x/ansi` | |
| Google API | `google.golang.org/api/tasks/v1` + `golang.org/x/oauth2` | |

## Package layout

```
main.go                  entrypoint: config → auth → TUI
internal/
  config/                config dir resolution (~/.config/guppy, XDG-aware)
  auth/                  OAuth2: credentials resolution, PKCE loopback flow, token cache
  gtasks/                Service interface + Google implementation + domain types
  tui/                   Bubble Tea model: model.go (state), update.go (input),
                         view.go (rendering), tree.go (task tree), styles.go,
                         keys.go, messages.go
```

The one hard boundary: **`gtasks.Service` is an interface** (`Lists`, `Tasks`,
`Add`, `Rename`, `SetCompleted`, `Delete`). The TUI never sees the Google API
types — only `gtasks.Task`/`gtasks.TaskList`. All TUI tests run against an
in-memory fake; the Google implementation is tested against an `httptest`
server. Keep it that way: new API operations go on the interface, the fake in
`update_test.go` gets a matching method, and UI logic stays testable.

## Bubble Tea v2

This project uses the **v2** API, which differs from most examples on the
internet (v1):

- Imports are vanity domains: `charm.land/bubbletea/v2`, not `github.com/charmbracelet/bubbletea`.
- `View() tea.View` (a struct), not `View() string`. We set `v.AltScreen`,
  `v.WindowTitle`, and `v.BackgroundColor` on it.
- Key events are `tea.KeyPressMsg` (not `tea.KeyMsg`); space is `"space"`,
  not `" "`.
- The model is a pointer (`*Model`) registered with `tea.NewProgram` —
  mutation in `Update` is fine, no value-copy semantics.
- Do not request keyboard enhancement protocols; with them, `shift+a` reports
  as `"shift+a"` instead of `"A"` and breaks the single-letter bindings.

All network/API calls are `tea.Cmd`s returning typed messages
(`listsLoadedMsg`, `tasksLoadedMsg`, `mutationDoneMsg`). Nothing blocks the
update loop.

## Rendering: the golden rule

**Every line is built at its exact final width as a single styled segment,
and no style with `Width` ever wraps styled content.**

Why: lipgloss v2 word-wraps content when a style has `Width` set, and its
wrapping **strips trailing whitespace** — including the padding that carries
a nested row's background color. A full-width selection bar nested inside a
background-filled pane silently collapses to a text-only highlight. (Found
empirically; the padding spaces arrive as an empty styled segment.)

So `view.go` composes the screen manually:

- Rows/titles: build the raw string, `ansi.Truncate` to width, `padRight`
  (spaces) to width, then apply **one** style that carries the background.
  The padding lives *inside* the styled segment, so the background spans the
  full pane width.
- Lines with multiple segments (tasks title = name + right-aligned counter)
  must place the variable segment last, or pad with a bg-colored segment
  (`padLine`) — after an inner ANSI reset, outer styling is gone.
- Blank pane lines are explicit full-width `colPanel` segments; the pane gap
  and top margin are plain spaces (terminal default background, which we set
  to `colBg` via `view.BackgroundColor`).
- The focused-pane left bar (`▏`) is prepended per line, drawn on **both**
  panes (panel-colored when blurred) so focus changes never shift the layout.
- Pane height is fixed by construction (`paneHeight()`), with explicit filler
  lines — no reliance on lipgloss `Height` padding.

`view_test.go`'s `checkScreen` enforces the invariant mechanically: every
rendered mode must produce exactly `height` lines, each exactly `width`
cells. If you change rendering, keep it green. For visual debugging, a
throwaway test logging `ansi.Strip(m.render())` (plus `%q` for raw bytes) is
the established pattern — don't commit it.

## Theming

Colors are opencode's built-in dark theme (`opencode.json`), mapped in
`styles.go`:

| Token | Hex | opencode field | Used for |
|---|---|---|---|
| `colBg` | `#0a0a0a` | background | screen, pane gap |
| `colPanel` | `#141414` | backgroundPanel | pane fill, title rows |
| `colElement` | `#1e1e1e` | backgroundElement | selection bars, status bar |
| `colAccent` | `#fab283` | primary | focused pane border + title, input prompt |
| `colText` / `colMuted` / `colFaint` | `#eeeeee` / `#808080` / `#606060` | text / textMuted / darkStep8 | normal, hints, completed |
| `colRed` | `#e06c75` | error | status bar errors |

Focus is conveyed by color alone (accent title + accent `▏` bar); the
selection bar is soft (`colElement`) in both panes, brighter text when
focused. The app forces this dark palette regardless of terminal theme by
filling every cell — there is no light variant.

## Task tree

The Google Tasks API returns a **flat** list; hierarchy is derived from
`parent` + `position` fields (`tree.go`). Rules: sort siblings by `position`
(lexicographic), render children indented under their parent, promote orphans
(missing parent) to the top level rather than dropping them. Google Tasks
supports exactly one nesting level; adding a subtask to a subtask creates a
sibling under the same top-level parent (`commitInput`).

Other API behaviors encoded in the app:

- Lists/tasks list calls paginate (`MaxResults(100)` + `Pages`).
- `Tasks.List` uses `ShowCompleted(true).ShowHidden(false)`: crossed-out
  completed tasks are shown, "cleared" ones are not.
- Deleting a parent deletes its subtasks — the delete confirmation says so.
- Insert accepts `parent`/`previous` as **query params** (`.Parent(...)` on
  the call), not just body fields.

## Sync model

- Toggle-complete is **optimistic**: flip local state, fire the mutation,
  refetch the list when it finishes (a failure sets a generic error in the
  status bar; the refetch restores truth).
- Add/rename/delete just refetch on completion.
- Raw tasks are cached per list (`Model.cache`) for instant pane switches;
  the cached render is immediately refreshed in the background
  (stale-while-revalidate).
- `tasksLoadedMsg` carries the list ID and is discarded if it arrives after
  the user switched lists (stale-response guard).

## Auth

- Release binaries embed shared OAuth client credentials
  (`internal/auth/credentials.go`), injected via `-ldflags -X` from repository
  secrets by `.github/workflows/release.yml`. The values aren't committed —
  GitHub push protection blocks pushes containing them — but they are
  **public by design**: desktop apps are public clients under RFC 8252; PKCE
  protects the flow. Never commit a *user's* token or `credentials.json`.
- Resolution order: `GUPPY_CLIENT_ID`/`GUPPY_CLIENT_SECRET` env vars →
  `$GUPPY_CREDENTIALS` / `~/.config/guppy/credentials.json` → embedded
  (release builds only; empty in source builds).
- Flow: loopback HTTP server on a random `127.0.0.1` port, browser opened
  automatically (URL printed as fallback), 3-minute timeout, state validated,
  PKCE verifier required at exchange.
- Token: cached at `~/.config/guppy/token.json` with `0600`; refreshed tokens
  are persisted by `persistSource`, preserving the refresh token (Google
  omits it on refresh responses).
- Scope is minimal: `tasks.TasksScope` only.
- Token lifetime: ~1h access token, refresh token is permanent **only if** the
  Google Cloud project is published (Production). In Testing status it
  expires after 7 days. `--reauth` discards the cached token.

## Conventions

- User-facing errors are generic ("Couldn't load tasks — press r to retry");
  wrapped internal errors (`fmt.Errorf("...: %w")) are fine internally but
  never rendered raw to the UI.
- Keys are plain strings matched on `msg.String()`; the full table lives in
  `update.go:onBrowseKey` and `keys.go` (help text) — update both plus the
  README when adding bindings.
- Tests: table-driven where possible; TUI tests use `feed`/`press` helpers
  (synchronously run commands and feed messages back). `tea.Batch` is not
  expected in update paths — keep it to `Init`.
