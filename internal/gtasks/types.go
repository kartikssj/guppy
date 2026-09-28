// Package gtasks defines guppy's view of the Google Tasks API.
package gtasks

// TaskList is a single Google Tasks list.
type TaskList struct {
	ID    string
	Title string
}

// Task is a single task. A subtask has Parent set to the ID of its
// top-level task; Google Tasks supports exactly one level of nesting.
type Task struct {
	ID        string
	Title     string
	Parent    string
	Position  string
	Completed bool
}
