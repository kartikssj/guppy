package tui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// paneGap is the number of app-background columns between the two panes.
const paneGap = 2

// topMargin is the number of app-background lines above the panes.
const topMargin = 1

// paneHeaderLines is the number of pane lines above the first item:
// top padding + title + spacing.
const paneHeaderLines = 3

// padLeft is the left padding inside each pane (after the border bar).
const padLeft = "   "

func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "guppy"
	v.BackgroundColor = colBg
	return v
}

func (m *Model) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.width < 40 || m.height < 8 {
		return "  guppy — terminal too small, please resize"
	}
	if m.mode == modeHelp {
		return m.renderHelp()
	}

	// Every pane line is built at its exact width with explicit backgrounds.
	// We deliberately avoid lipgloss Width/Height composition here: lipgloss
	// word-wraps when a style has a width, and wrapping strips the trailing
	// whitespace that carries nested backgrounds — which would break the
	// full-width selection bars.
	listTotal, taskTotal := m.paneWidths()
	paneH := m.paneHeight()
	listLines := m.listsPaneLines(listTotal-1, paneH) // -1: border column
	taskLines := m.tasksPaneLines(taskTotal-1, paneH)
	borderL := m.borderPrefix(paneLists)
	borderT := m.borderPrefix(paneTasks)
	gap := strings.Repeat(" ", paneGap)

	var lines []string
	lines = append(lines, strings.Repeat(" ", m.width)) // top margin
	for i := 0; i < paneH; i++ {
		lines = append(lines, borderL+listLines[i]+gap+borderT+taskLines[i])
	}
	if m.mode == modeInput {
		lines = append(lines, padLine(" "+m.input.View(), m.width, colBg))
	}
	lines = append(lines, m.statusLine())
	return strings.Join(lines, "\n")
}

// paneWidths returns the total widths of the two panes, border included.
func (m *Model) paneWidths() (listTotal, taskTotal int) {
	listTotal = m.width / 4
	if listTotal < 22 {
		listTotal = 22
	}
	if listTotal > 32 {
		listTotal = 32
	}
	if maxList := m.width - paneGap - 24; listTotal > maxList {
		listTotal = maxList
	}
	taskTotal = m.width - paneGap - listTotal
	return listTotal, taskTotal
}

func (m *Model) borderPrefix(p pane) string {
	if m.focus == p {
		return styleBorderFocused.Render(borderBar)
	}
	return styleBorderBlurred.Render(borderBar)
}

// padRight pads a line with plain spaces to width w. The result is meant to
// be wrapped in a single style, so the padding inherits that style's
// background — this is what makes selection bars span the full pane width.
func padRight(s string, w int) string {
	if gw := lipgloss.Width(s); gw < w {
		return s + strings.Repeat(" ", w-gw)
	}
	return s
}

// padLine pads a line that already contains styled segments by appending a
// background-colored segment, so the line still spans the full width.
func padLine(s string, w int, bg color.Color) string {
	if gw := lipgloss.Width(s); gw < w {
		return s + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", w-gw))
	}
	return s
}

func blankLine(w int) string {
	return styleRow.Render(strings.Repeat(" ", w))
}

// listsPaneLines returns exactly h lines, each contentW wide.
func (m *Model) listsPaneLines(contentW, h int) []string {
	lines := []string{
		blankLine(contentW),
		m.titleLine(paneLists, fmt.Sprintf("Lists (%d)", len(m.lists)), contentW),
		blankLine(contentW),
	}
	itemsH := m.itemHeight()
	shown := 0
	for i := m.listOffset; i < len(m.lists) && shown < itemsH; i++ {
		lines = append(lines, m.listRowLine(i, contentW))
		shown++
	}
	if len(m.lists) == 0 && !m.loading {
		lines = append(lines, styleMutedPanel.Render(padRight(padLeft+"No lists yet", contentW)))
	}
	for len(lines) < h {
		lines = append(lines, blankLine(contentW))
	}
	return lines[:h]
}

func (m *Model) titleLine(p pane, text string, w int) string {
	st := styleTitleBlurred
	if m.focus == p {
		st = styleTitleFocused
	}
	return st.Render(padRight(ansi.Truncate(padLeft+text, w, "…"), w))
}

func (m *Model) listRowLine(i, w int) string {
	raw := padRight(ansi.Truncate(padLeft+m.lists[i].Title, w, "…"), w)
	if i == m.listCursor {
		if m.focus == paneLists {
			return styleRowCursor.Render(raw)
		}
		return styleRowCursorBlurred.Render(raw)
	}
	return styleRow.Render(raw)
}

// tasksPaneLines returns exactly h lines, each contentW wide.
func (m *Model) tasksPaneLines(contentW, h int) []string {
	lines := []string{
		blankLine(contentW),
		m.tasksTitleLine(contentW),
		blankLine(contentW),
	}
	itemsH := m.itemHeight()
	shown := 0
	for i := m.taskOffset; i < len(m.rows) && shown < itemsH; i++ {
		lines = append(lines, m.taskRowLine(i, contentW))
		shown++
	}
	switch {
	case m.loading && len(m.rows) == 0:
		lines = append(lines, styleMutedPanel.Render(padRight(padLeft+"Loading…", contentW)))
	case len(m.rows) == 0 && m.activeListID != "":
		lines = append(lines, styleMutedPanel.Render(padRight(padLeft+"No tasks — press a to add one", contentW)))
	}
	for len(lines) < h {
		lines = append(lines, blankLine(contentW))
	}
	return lines[:h]
}

// tasksTitleLine composes the list name with a muted "done/total" counter.
// The counter must be the last segment on the line so no unstyled padding
// can follow its reset.
func (m *Model) tasksTitleLine(w int) string {
	st := styleTitleBlurred
	if m.focus == paneTasks {
		st = styleTitleFocused
	}
	meta := ""
	if len(m.rows) > 0 {
		done := 0
		for _, r := range m.rows {
			if r.Task.Completed {
				done++
			}
		}
		meta = fmt.Sprintf("%d/%d", done, len(m.rows))
	}
	if m.loading {
		if meta != "" {
			meta += " · …"
		} else {
			meta = "…"
		}
	}
	nameW := w - lipgloss.Width(meta) - 3 // 2: gap before the counter, 1: right margin
	if nameW < 0 {
		nameW = 0
	}
	name := padRight(ansi.Truncate(padLeft+m.selectedListTitle(), nameW, "…"), nameW)
	return st.Render(name) + styleMeta.Render("  "+meta+" ")
}

func (m *Model) taskRowLine(i, w int) string {
	row := m.rows[i]

	indent := strings.Repeat("  ", row.Depth)
	if row.Depth > 0 {
		indent = strings.Repeat("  ", row.Depth-1) + " └ "
	}
	checkbox := "☐  "
	if row.Task.Completed {
		checkbox = "☑  "
	}
	raw := padRight(ansi.Truncate(padLeft+indent+checkbox+row.Task.Title, w, "…"), w)

	selected := i == m.taskCursor
	switch {
	case selected && m.focus == paneTasks && row.Task.Completed:
		return styleRowCursorCompleted.Render(raw)
	case selected && m.focus == paneTasks:
		return styleRowCursor.Render(raw)
	case selected && row.Task.Completed:
		return styleRowCursorBlurredCompleted.Render(raw)
	case selected:
		return styleRowCursorBlurred.Render(raw)
	case row.Task.Completed:
		return styleRowCompleted.Render(raw)
	default:
		return styleRow.Render(raw)
	}
}

func (m *Model) statusLine() string {
	var s string
	switch {
	case m.mode == modeConfirmDelete && len(m.rows) > 0:
		t := m.rows[m.taskCursor].Task
		s = fmt.Sprintf("Delete %q? Its subtasks are deleted too. [y/n]", t.Title)
	case m.status != "":
		s = m.status
	case m.loading:
		s = "Loading…"
	default:
		s = statusHint
	}
	s = padRight(" "+ansi.Truncate(s, m.width-2, "…"), m.width)
	if m.isError {
		return styleStatusBarErr.Render(s)
	}
	return styleStatusBar.Render(s)
}

func (m *Model) renderHelp() string {
	content := helpText
	if m.height < 24 || m.width < 60 {
		content = "guppy help — terminal too small for the full list.\n" + statusHint + "\n\nPress any key to close."
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		styleHelpBox.Render(content),
		lipgloss.WithWhitespaceStyle(styleApp))
}
