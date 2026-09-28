package tui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// Palette: opencode's built-in dark theme, from
// packages/opencode/src/cli/cmd/tui/context/theme/opencode.json
var (
	colBg      = lipgloss.Color("#0a0a0a") // background
	colPanel   = lipgloss.Color("#141414") // backgroundPanel
	colElement = lipgloss.Color("#1e1e1e") // backgroundElement — opencode's prompt-input fill; our selection
	colAccent  = lipgloss.Color("#fab283") // primary — focused pane border + title
	colText    = lipgloss.Color("#eeeeee") // text
	colMuted   = lipgloss.Color("#808080") // textMuted
	colFaint   = lipgloss.Color("#606060") // darkStep8 — completed tasks
	colRed     = lipgloss.Color("#e06c75") // error
)

// borderBar marks the focused pane's left edge. It is always drawn on both
// panes (panel-colored when blurred, i.e. invisible) so focus changes never
// shift the layout.
const borderBar = "▏"

var (
	styleBorderFocused = lipgloss.NewStyle().Background(colPanel).Foreground(colAccent)
	styleBorderBlurred = lipgloss.NewStyle().Background(colPanel).Foreground(colPanel)

	// Every row style carries an explicit background: rows are padded inside
	// the styled segment (see padRight), so backgrounds span the full width.
	styleRow                       = lipgloss.NewStyle().Background(colPanel).Foreground(colText)
	styleRowCompleted              = lipgloss.NewStyle().Background(colPanel).Foreground(colFaint).Strikethrough(true)
	styleRowCursor                 = lipgloss.NewStyle().Background(colElement).Foreground(colText).Bold(true)
	styleRowCursorCompleted        = lipgloss.NewStyle().Background(colElement).Foreground(colFaint).Strikethrough(true).Bold(true)
	styleRowCursorBlurred          = lipgloss.NewStyle().Background(colElement).Foreground(colMuted)
	styleRowCursorBlurredCompleted = lipgloss.NewStyle().Background(colElement).Foreground(colFaint).Strikethrough(true)

	styleTitleFocused = lipgloss.NewStyle().Background(colPanel).Foreground(colAccent).Bold(true)
	styleTitleBlurred = lipgloss.NewStyle().Background(colPanel).Foreground(colMuted).Bold(true)
	styleMeta         = lipgloss.NewStyle().Background(colPanel).Foreground(colMuted)
	styleMutedPanel   = lipgloss.NewStyle().Background(colPanel).Foreground(colMuted)

	styleStatusBar    = lipgloss.NewStyle().Background(colElement).Foreground(colMuted)
	styleStatusBarErr = lipgloss.NewStyle().Background(colElement).Foreground(colRed)

	styleHelpBox = lipgloss.NewStyle().Background(colPanel).Foreground(colText).Padding(1, 2)
	styleApp     = lipgloss.NewStyle().Background(colBg) // help-screen whitespace
)

func newInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.CharLimit = 512
	ti.SetWidth(40) // sensible default; resized on tea.WindowSizeMsg
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	s.Focused.Text = lipgloss.NewStyle().Foreground(colText)
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(colMuted)
	ti.SetStyles(s)
	return ti
}
