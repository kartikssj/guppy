# AGENTS.md

Guidance for AI agents and contributors working on guppy. Read
[DESIGN.md](DESIGN.md) before changing architecture or rendering; it explains
the *why* behind the non-obvious parts.

## What this is

guppy is a keyboard-driven terminal UI for Google Tasks. Go 1.26, single
binary, Bubble Tea **v2** (not v1 — check imports: `charm.land/bubbletea/v2`).

## Build, test, verify

```sh
gofmt -w . && go vet ./... && go test ./... -count=1 && go build -o guppy .
```

All must pass before a change is done. No linter config beyond `go vet`.

## Hard rules

1. **Rendering:** never nest styled content inside a lipgloss style that has
   `Width` set — wrapping strips trailing whitespace and breaks background
   fills. Build every line at exact width as a single styled segment
   (`padRight`/`padLine` helpers in `view.go`). `checkScreen` in
   `view_test.go` must stay green. Details: DESIGN.md "Rendering".
2. **API boundary:** the TUI only talks to `internal/gtasks.Service` (an
   interface). New API operations: add to the interface, extend the fake in
   `update_test.go`, keep Google types inside `internal/gtasks`.
3. **Security:**
   - The OAuth client ID/secret are public metadata (RFC 8252 public client)
     — do not rotate or treat as a leaked secret. They are not committed
     (GitHub push protection blocks them); release builds inject them via
     `-ldflags -X` from the `GUPPY_CLIENT_ID`/`GUPPY_CLIENT_SECRET`
     repository secrets (`.github/workflows/release.yml`).
   - Never commit user tokens or a user's `credentials.json`
     (`.gitignore`d); token file permissions stay `0600`.
   - Keep the OAuth scope minimal (`tasks.TasksScope`). Adding scopes
     requires README + consent-screen changes.
   - UI error messages stay generic; no raw API errors, tokens, or stack
     traces in the UI.
   - Data practices promised to users live in PRIVACY.md/TERMS.md (no
     telemetry, no disk cache of task data, local-only). Any feature that
     changes data handling must update both files first.
4. **Bubble Tea v2 idioms:** `View() tea.View`, `tea.KeyPressMsg`,
   `"space"` not `" "`. Don't enable keyboard enhancement protocols (breaks
   `A` binding by reporting `shift+a`).
5. **Minimal diffs:** match the existing style; don't refactor adjacent code
   unprompted; no new dependencies without a clear need.

## When changing…

- **Keybindings:** update `update.go` (handler), `keys.go` (help text), and
  README.md (keys table).
- **Colors/layout:** `styles.go` + `view.go` only; keep the opencode palette
  mapping table in DESIGN.md in sync.
- **Google Tasks behavior** (e.g. due dates, clearing completed): check the
  API notes in DESIGN.md "Task tree" first — several non-obvious behaviors
  (parent query param, one nesting level, hidden vs completed) are encoded
  there.
- **Auth/credentials:** keep the resolution order (env → file → embedded)
  and the public-client rationale comment intact.

## Testing expectations

- New update-loop logic needs tests via the `feed`/`press` helpers against
  the fake service (see `update_test.go`).
- API client changes need `httptest` tests (see `client_test.go`).
- Rendering changes must keep `view_test.go` passing; add modes to it when
  new screens appear.

## House style

- Comments explain *why*, not what; package docs on every package.
- Errors wrapped with context via `fmt.Errorf("...: %w", err)`.
- Plain strings for key matches; no key-binding abstraction library.
