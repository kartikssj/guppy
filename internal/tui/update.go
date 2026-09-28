package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(max(m.width-6, 10))
		m.ensureListVisible()
		m.ensureTaskVisible()
		return m, nil

	case listsLoadedMsg:
		return m.onListsLoaded(msg)

	case tasksLoadedMsg:
		return m.onTasksLoaded(msg)

	case mutationDoneMsg:
		if msg.err != nil {
			m.setError("Couldn't reach Google Tasks — press r to retry")
		}
		if m.activeListID != "" {
			return m, fetchTasks(m.svc, m.activeListID)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.onKey(msg)
	}
	return m, nil
}

func (m *Model) onListsLoaded(msg listsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.setError("Couldn't load your lists — press r to retry")
		return m, nil
	}
	m.lists = msg.lists
	m.listCursor = clamp(m.listCursor, 0, len(m.lists)-1)
	return m.loadSelectedList()
}

func (m *Model) onTasksLoaded(msg tasksLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil {
		m.cache[msg.listID] = msg.tasks
	}
	if msg.listID != m.activeListID {
		return m, nil // stale response for a list we've since left
	}
	m.loading = false
	if msg.err != nil {
		m.setError("Couldn't load tasks — press r to retry")
		return m, nil
	}
	m.rows = Flatten(msg.tasks)
	m.taskCursor = clamp(m.taskCursor, 0, len(m.rows)-1)
	m.ensureTaskVisible()
	return m, nil
}

// loadSelectedList switches the tasks pane to the currently selected list,
// rendering cached data instantly when available and refreshing in the
// background.
func (m *Model) loadSelectedList() (tea.Model, tea.Cmd) {
	id := m.selectedListID()
	m.activeListID = id
	m.taskCursor, m.taskOffset = 0, 0
	if cached, ok := m.cache[id]; ok {
		m.rows = Flatten(cached)
		m.loading = false
	} else {
		m.rows = nil
		m.loading = id != ""
	}
	if id == "" {
		return m, nil
	}
	return m, fetchTasks(m.svc, id)
}

func (m *Model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInput:
		return m.onInputKey(msg)
	case modeConfirmDelete:
		return m.onConfirmKey(msg)
	case modeHelp:
		m.mode = modeBrowse // any key closes help
		return m, nil
	default:
		return m.onBrowseKey(msg)
	}
}

func (m *Model) onBrowseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.clearStatus()
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "tab", "shift+tab":
		m.focus = m.focus.other()
	case "right", "l":
		m.focus = paneTasks
	case "left", "h":
		m.focus = paneLists
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "enter":
		if m.focus == paneLists {
			m.focus = paneTasks
		}
	case "r":
		m.loading = true
		return m, fetchLists(m.svc)
	case "space":
		return m.toggleSelected()
	case "a":
		return m.startInput(purposeAdd)
	case "A":
		return m.startInput(purposeAddSubtask)
	case "e":
		return m.startInput(purposeRename)
	case "d":
		return m.confirmDelete()
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	if m.focus == paneLists {
		next := clamp(m.listCursor+delta, 0, len(m.lists)-1)
		if next == m.listCursor {
			return m, nil
		}
		m.listCursor = next
		m.ensureListVisible()
		return m.loadSelectedList()
	}
	m.taskCursor = clamp(m.taskCursor+delta, 0, len(m.rows)-1)
	m.ensureTaskVisible()
	return m, nil
}

func (m *Model) toggleSelected() (tea.Model, tea.Cmd) {
	if m.focus != paneTasks || len(m.rows) == 0 || m.activeListID == "" {
		return m, nil
	}
	// Optimistic update: flip locally, persist, refetch on completion.
	row := &m.rows[m.taskCursor]
	row.Task.Completed = !row.Task.Completed
	listID, taskID, done := m.activeListID, row.Task.ID, row.Task.Completed
	return m, runMutation(func() error {
		return m.svc.SetCompleted(context.Background(), listID, taskID, done)
	})
}

func (m *Model) startInput(p inputPurpose) (tea.Model, tea.Cmd) {
	if m.activeListID == "" {
		m.setError("No list selected")
		return m, nil
	}
	m.purpose = p
	switch p {
	case purposeAdd:
		m.input.Placeholder = "New task"
		m.input.SetValue("")
	case purposeAddSubtask:
		if len(m.rows) == 0 {
			m.setError("Select a task first")
			return m, nil
		}
		m.input.Placeholder = "New subtask"
		m.input.SetValue("")
	case purposeRename:
		if len(m.rows) == 0 {
			m.setError("Select a task first")
			return m, nil
		}
		m.input.Placeholder = "Rename task"
		m.input.SetValue(m.rows[m.taskCursor].Task.Title)
	}
	m.mode = modeInput
	return m, m.input.Focus()
}

func (m *Model) onInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		title := strings.TrimSpace(m.input.Value())
		purpose := m.purpose
		m.cancelInput()
		if title == "" {
			return m, nil
		}
		return m.commitInput(purpose, title)
	case "esc":
		m.cancelInput()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) cancelInput() {
	m.mode = modeBrowse
	m.input.Blur()
	m.input.SetValue("")
}

func (m *Model) commitInput(p inputPurpose, title string) (tea.Model, tea.Cmd) {
	listID := m.activeListID
	switch p {
	case purposeAdd:
		return m, runMutation(func() error {
			_, err := m.svc.Add(context.Background(), listID, title, "")
			return err
		})
	case purposeAddSubtask:
		parent := m.rows[m.taskCursor].Task
		parentID := parent.ID
		if parent.Parent != "" {
			// Google Tasks supports one nesting level: a "subtask of a
			// subtask" becomes a sibling under the same parent.
			parentID = parent.Parent
		}
		return m, runMutation(func() error {
			_, err := m.svc.Add(context.Background(), listID, title, parentID)
			return err
		})
	case purposeRename:
		taskID := m.rows[m.taskCursor].Task.ID
		return m, runMutation(func() error {
			return m.svc.Rename(context.Background(), listID, taskID, title)
		})
	}
	return m, nil
}

func (m *Model) confirmDelete() (tea.Model, tea.Cmd) {
	if m.focus != paneTasks || len(m.rows) == 0 {
		return m, nil
	}
	m.mode = modeConfirmDelete
	return m, nil
}

func (m *Model) onConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.mode = modeBrowse
	if msg.String() != "y" && msg.String() != "enter" {
		return m, nil
	}
	listID := m.activeListID
	taskID := m.rows[m.taskCursor].Task.ID
	return m, runMutation(func() error {
		return m.svc.Delete(context.Background(), listID, taskID)
	})
}
