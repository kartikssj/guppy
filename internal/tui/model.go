package tui

import (
	"context"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"guppy/internal/gtasks"
)

type pane int

const (
	paneLists pane = iota
	paneTasks
)

func (p pane) other() pane {
	if p == paneLists {
		return paneTasks
	}
	return paneLists
}

type mode int

const (
	modeBrowse mode = iota
	modeInput
	modeConfirmDelete
	modeHelp
)

type inputPurpose int

const (
	purposeAdd inputPurpose = iota
	purposeAddSubtask
	purposeRename
)

// Model is the root Bubble Tea model for guppy.
type Model struct {
	svc gtasks.Service

	lists      []gtasks.TaskList
	listCursor int
	listOffset int

	activeListID string
	cache        map[string][]gtasks.Task // raw tasks per list, for instant pane switches
	rows         []Row
	taskCursor   int
	taskOffset   int

	focus   pane
	mode    mode
	purpose inputPurpose

	input textinput.Model

	loading bool
	status  string
	isError bool

	width  int
	height int
}

// NewModel builds the initial model; lists load on Init.
func NewModel(svc gtasks.Service) *Model {
	return &Model{
		svc:     svc,
		cache:   map[string][]gtasks.Task{},
		focus:   paneLists,
		mode:    modeBrowse,
		loading: true,
		input:   newInput(),
	}
}

// Run starts the TUI.
func Run(svc gtasks.Service) error {
	p := tea.NewProgram(NewModel(svc))
	_, err := p.Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(fetchLists(m.svc), textinput.Blink)
}

// --- commands ---

func fetchLists(svc gtasks.Service) tea.Cmd {
	return func() tea.Msg {
		lists, err := svc.Lists(context.Background())
		return listsLoadedMsg{lists: lists, err: err}
	}
}

func fetchTasks(svc gtasks.Service, listID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := svc.Tasks(context.Background(), listID)
		return tasksLoadedMsg{listID: listID, tasks: tasks, err: err}
	}
}

// runMutation executes a write operation and reports completion. The Update
// handler for mutationDoneMsg refetches the affected list.
func runMutation(fn func() error) tea.Cmd {
	return func() tea.Msg {
		return mutationDoneMsg{err: fn()}
	}
}

// --- shared helpers ---

func (m *Model) selectedListID() string {
	if m.listCursor >= 0 && m.listCursor < len(m.lists) {
		return m.lists[m.listCursor].ID
	}
	return ""
}

func (m *Model) selectedListTitle() string {
	if m.listCursor >= 0 && m.listCursor < len(m.lists) {
		return m.lists[m.listCursor].Title
	}
	return "Tasks"
}

func (m *Model) setError(s string)  { m.status, m.isError = s, true }
func (m *Model) setStatus(s string) { m.status, m.isError = s, false }
func (m *Model) clearStatus()       { m.status, m.isError = "", false }

// paneHeight is the total height of each pane, border included.
func (m *Model) paneHeight() int {
	h := m.height - 1 - topMargin // status bar + top margin
	if m.mode == modeInput {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// itemHeight is the number of list/task rows visible below the pane header
// (top padding + title + spacing line).
func (m *Model) itemHeight() int {
	h := m.paneHeight() - paneHeaderLines
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) ensureListVisible() {
	visible := m.itemHeight()
	if m.listCursor < m.listOffset {
		m.listOffset = m.listCursor
	}
	if m.listCursor >= m.listOffset+visible {
		m.listOffset = m.listCursor - visible + 1
	}
}

func (m *Model) ensureTaskVisible() {
	visible := m.itemHeight()
	if m.taskCursor < m.taskOffset {
		m.taskOffset = m.taskCursor
	}
	if m.taskCursor >= m.taskOffset+visible {
		m.taskOffset = m.taskCursor - visible + 1
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
