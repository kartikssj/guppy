package tui

import (
	"sort"

	"guppy/internal/gtasks"
)

// Row is one rendered line in the tasks pane: a task plus its nesting depth.
type Row struct {
	Task  gtasks.Task
	Depth int
}

// Flatten orders tasks for display: top-level tasks sorted by position, each
// followed by their subtasks (also sorted by position). Subtasks whose parent
// is missing from the input are promoted to the top level rather than lost.
func Flatten(tasks []gtasks.Task) []Row {
	ids := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		ids[t.ID] = true
	}

	var roots []gtasks.Task
	byParent := map[string][]gtasks.Task{}
	for _, t := range tasks {
		if t.Parent != "" && ids[t.Parent] {
			byParent[t.Parent] = append(byParent[t.Parent], t)
		} else {
			roots = append(roots, t)
		}
	}

	byPosition := func(ts []gtasks.Task) {
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].Position < ts[j].Position })
	}
	byPosition(roots)
	for _, kids := range byParent {
		byPosition(kids)
	}

	var rows []Row
	var walk func(ts []gtasks.Task, depth int)
	walk = func(ts []gtasks.Task, depth int) {
		for _, t := range ts {
			rows = append(rows, Row{Task: t, Depth: depth})
			if kids, ok := byParent[t.ID]; ok {
				walk(kids, depth+1)
			}
		}
	}
	walk(roots, 0)
	return rows
}
