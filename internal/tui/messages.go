package tui

import "guppy/internal/gtasks"

type listsLoadedMsg struct {
	lists []gtasks.TaskList
	err   error
}

type tasksLoadedMsg struct {
	listID string
	tasks  []gtasks.Task
	err    error
}

type mutationDoneMsg struct {
	err error
}
