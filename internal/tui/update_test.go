package tui

import (
	"context"
	"errors"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"guppy/internal/gtasks"
)

// --- fake service ---

type addedTask struct{ listID, title, parentID string }

type fakeService struct {
	lists []gtasks.TaskList
	tasks map[string][]gtasks.Task
	err   error

	added   []addedTask
	renamed map[string]string
	toggled map[string]bool
	deleted []string
	nextID  int
}

func newFakeService() *fakeService {
	return &fakeService{
		lists: []gtasks.TaskList{
			{ID: "l1", Title: "One"},
			{ID: "l2", Title: "Two"},
		},
		tasks: map[string][]gtasks.Task{
			"l1": {
				{ID: "a", Title: "Alpha", Position: "1"},
				{ID: "b", Title: "Beta", Position: "2", Completed: true},
				{ID: "c", Title: "Gamma", Position: "3", Parent: "a"},
			},
			"l2": {},
		},
		renamed: map[string]string{},
		toggled: map[string]bool{},
	}
}

func (f *fakeService) Lists(context.Context) ([]gtasks.TaskList, error) {
	return f.lists, f.err
}

func (f *fakeService) Tasks(_ context.Context, listID string) ([]gtasks.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]gtasks.Task(nil), f.tasks[listID]...), nil
}

func (f *fakeService) Add(_ context.Context, listID, title, parentID string) (gtasks.Task, error) {
	if f.err != nil {
		return gtasks.Task{}, f.err
	}
	f.nextID++
	t := gtasks.Task{ID: fmt.Sprintf("new%d", f.nextID), Title: title, Parent: parentID, Position: "9"}
	f.tasks[listID] = append(f.tasks[listID], t)
	f.added = append(f.added, addedTask{listID, title, parentID})
	return t, nil
}

func (f *fakeService) Rename(_ context.Context, listID, taskID, title string) error {
	if f.err != nil {
		return f.err
	}
	for i, t := range f.tasks[listID] {
		if t.ID == taskID {
			f.tasks[listID][i].Title = title
		}
	}
	f.renamed[taskID] = title
	return nil
}

func (f *fakeService) SetCompleted(_ context.Context, listID, taskID string, completed bool) error {
	if f.err != nil {
		return f.err
	}
	for i, t := range f.tasks[listID] {
		if t.ID == taskID {
			f.tasks[listID][i].Completed = completed
		}
	}
	f.toggled[taskID] = completed
	return nil
}

func (f *fakeService) Delete(_ context.Context, listID, taskID string) error {
	if f.err != nil {
		return f.err
	}
	var kept []gtasks.Task
	for _, t := range f.tasks[listID] {
		if t.ID == taskID || t.Parent == taskID { // subtasks go too
			continue
		}
		kept = append(kept, t)
	}
	f.tasks[listID] = kept
	f.deleted = append(f.deleted, taskID)
	return nil
}

// --- helpers ---

func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "space":
		return tea.KeyPressMsg{Code: ' ', Text: " "}
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg{Code: r, Text: string(r)}
	}
}

// feed delivers a message to Update and then synchronously runs any returned
// command, feeding its message back until the model settles.
func feed(t *testing.T, m *Model, msg tea.Msg) *Model {
	t.Helper()
	for {
		um, cmd := m.Update(msg)
		m = um.(*Model)
		if cmd == nil {
			return m
		}
		next := cmd()
		if next == nil {
			return m
		}
		if _, isBatch := next.(tea.BatchMsg); isBatch {
			t.Fatal("unexpected batch command in test")
		}
		msg = next
	}
}

func press(t *testing.T, m *Model, keys ...string) *Model {
	t.Helper()
	for _, k := range keys {
		m = feed(t, m, keyMsg(k))
	}
	return m
}

func newTestModel() (*Model, *fakeService) {
	fake := newFakeService()
	m := NewModel(fake)
	m.width, m.height = 100, 30
	return m, fake
}

// load performs the initial lists+tasks load cycle.
func load(t *testing.T, m *Model) *Model {
	t.Helper()
	return feed(t, m, fetchLists(m.svc)())
}

// --- tests ---

func TestInitialLoad(t *testing.T) {
	m, _ := newTestModel()
	m = load(t, m)

	if len(m.lists) != 2 {
		t.Fatalf("lists = %+v", m.lists)
	}
	if m.activeListID != "l1" {
		t.Fatalf("activeListID = %q", m.activeListID)
	}
	// Alpha, its subtask Gamma, then Beta.
	want := []struct {
		title string
		depth int
	}{
		{"Alpha", 0}, {"Gamma", 1}, {"Beta", 0},
	}
	if len(m.rows) != len(want) {
		t.Fatalf("rows = %+v", m.rows)
	}
	for i, w := range want {
		if m.rows[i].Task.Title != w.title || m.rows[i].Depth != w.depth {
			t.Fatalf("row %d = %s@%d, want %s@%d",
				i, m.rows[i].Task.Title, m.rows[i].Depth, w.title, w.depth)
		}
	}
}

func TestListNavigationClampsAndSwitches(t *testing.T) {
	m, _ := newTestModel()
	m = load(t, m)

	m = press(t, m, "down")
	if m.listCursor != 1 || m.activeListID != "l2" {
		t.Fatalf("cursor=%d active=%q", m.listCursor, m.activeListID)
	}
	if len(m.rows) != 0 {
		t.Fatalf("l2 should be empty, got %+v", m.rows)
	}

	m = press(t, m, "down", "down") // clamped at the bottom
	if m.listCursor != 1 {
		t.Fatalf("cursor should clamp at 1, got %d", m.listCursor)
	}

	m = press(t, m, "up") // back to l1, served from cache
	if m.listCursor != 0 || m.activeListID != "l1" || len(m.rows) != 3 {
		t.Fatalf("cursor=%d active=%q rows=%d", m.listCursor, m.activeListID, len(m.rows))
	}
}

func TestToggleRequiresTasksFocus(t *testing.T) {
	m, fake := newTestModel()
	m = load(t, m)

	m = press(t, m, "space") // still on lists pane: no-op
	if len(fake.toggled) != 0 {
		t.Fatalf("toggle should not fire on lists pane: %v", fake.toggled)
	}
}

func TestToggleCompleteOptimistic(t *testing.T) {
	m, fake := newTestModel()
	m = load(t, m)
	m = press(t, m, "tab") // focus tasks

	m = press(t, m, "space") // toggle Alpha -> completed
	if !m.rows[0].Task.Completed {
		t.Fatal("optimistic update should mark the task completed immediately")
	}
	if done, ok := fake.toggled["a"]; !ok || !done {
		t.Fatalf("SetCompleted not recorded: %v", fake.toggled)
	}
	if !m.rows[0].Task.Completed {
		t.Fatal("task should still be completed after refetch")
	}
}

func TestAddTaskFlow(t *testing.T) {
	m, fake := newTestModel()
	m = load(t, m)

	m = press(t, m, "a")
	if m.mode != modeInput {
		t.Fatalf("mode = %v, want input", m.mode)
	}
	m = press(t, m, "B", "u", "y")
	if got := m.input.Value(); got != "Buy" {
		t.Fatalf("input value = %q", got)
	}
	m = press(t, m, "enter")

	if m.mode != modeBrowse {
		t.Fatalf("mode = %v, want browse", m.mode)
	}
	if len(fake.added) != 1 || fake.added[0] != (addedTask{"l1", "Buy", ""}) {
		t.Fatalf("added = %+v", fake.added)
	}
	found := false
	for _, r := range m.rows {
		if r.Task.Title == "Buy" {
			found = true
		}
	}
	if !found {
		t.Fatal("new task should appear after refetch")
	}
}

func TestAddSubtaskOnSubtaskBecomesSibling(t *testing.T) {
	m, fake := newTestModel()
	m = load(t, m)
	m = press(t, m, "tab", "down") // select Gamma (the subtask)

	m = press(t, m, "A")
	if m.mode != modeInput {
		t.Fatalf("mode = %v, want input", m.mode)
	}
	m = press(t, m, "X", "enter")

	if len(fake.added) != 1 || fake.added[0].parentID != "a" {
		t.Fatalf("subtask should attach to top-level parent a: %+v", fake.added)
	}
}

func TestRenameFlow(t *testing.T) {
	m, fake := newTestModel()
	m = load(t, m)
	m = press(t, m, "tab", "e")

	if m.mode != modeInput {
		t.Fatalf("mode = %v, want input", m.mode)
	}
	if m.input.Value() != "Alpha" {
		t.Fatalf("rename should prefill the current title, got %q", m.input.Value())
	}
	m.input.SetValue("Alpha two")
	m = press(t, m, "enter")

	if fake.renamed["a"] != "Alpha two" {
		t.Fatalf("renamed = %v", fake.renamed)
	}
}

func TestDeleteFlow(t *testing.T) {
	m, fake := newTestModel()
	m = load(t, m)
	m = press(t, m, "tab", "d")

	if m.mode != modeConfirmDelete {
		t.Fatalf("mode = %v, want confirm", m.mode)
	}
	m = press(t, m, "n") // cancelled
	if len(fake.deleted) != 0 {
		t.Fatalf("delete should be cancelled: %v", fake.deleted)
	}

	m = press(t, m, "d", "y") // confirmed
	if len(fake.deleted) != 1 || fake.deleted[0] != "a" {
		t.Fatalf("deleted = %v", fake.deleted)
	}
	for _, r := range m.rows {
		if r.Task.ID == "a" || r.Task.ID == "c" {
			t.Fatalf("deleted task/subtask should be gone after refetch: %+v", m.rows)
		}
	}
}

func TestStaleTasksResponseIgnored(t *testing.T) {
	m, _ := newTestModel()
	m = load(t, m)
	before := m.rows

	m = feed(t, m, tasksLoadedMsg{listID: "zz", tasks: nil})
	if len(m.rows) != len(before) {
		t.Fatalf("stale response should be ignored: %+v", m.rows)
	}
}

func TestLoadErrorSetsStatus(t *testing.T) {
	m, fake := newTestModel()
	fake.err = errors.New("boom")
	m = load(t, m)

	if m.status == "" || !m.isError {
		t.Fatalf("expected an error status, got %q", m.status)
	}
}

func TestHelpOpensAndCloses(t *testing.T) {
	m, _ := newTestModel()
	m = load(t, m)

	m = press(t, m, "?")
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	m = press(t, m, "j") // any key closes
	if m.mode != modeBrowse {
		t.Fatalf("mode = %v, want browse", m.mode)
	}
}
