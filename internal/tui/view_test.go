package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// checkScreen verifies the render exactly fills the terminal: every line has
// the full width (so backgrounds cover the whole screen) and the total number
// of lines matches the height (so nothing overflows and scrolls).
func checkScreen(t *testing.T, out string, width, height int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != height {
		t.Fatalf("rendered %d lines, want %d", len(lines), height)
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != width {
			t.Fatalf("line %d has width %d, want %d: %q", i, w, width, ln)
		}
	}
}

func TestRenderFillsScreen(t *testing.T) {
	cases := map[string]struct {
		prepare func(t *testing.T, m *Model) *Model
	}{
		"browse":  {prepare: func(t *testing.T, m *Model) *Model { return m }},
		"input":   {prepare: func(t *testing.T, m *Model) *Model { return press(t, m, "a") }},
		"help":    {prepare: func(t *testing.T, m *Model) *Model { return press(t, m, "?") }},
		"confirm": {prepare: func(t *testing.T, m *Model) *Model { return press(t, m, "tab", "d") }},
		"narrow": {prepare: func(t *testing.T, m *Model) *Model {
			m.width, m.height = 48, 12
			return m
		}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m, _ := newTestModel()
			m = load(t, m)
			m = tc.prepare(t, m)
			checkScreen(t, m.render(), m.width, m.height)
		})
	}
}

func TestRenderEmptyModel(t *testing.T) {
	m, _ := newTestModel() // no load: zero lists, loading state
	checkScreen(t, m.render(), m.width, m.height)
}
