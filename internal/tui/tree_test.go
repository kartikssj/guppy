package tui

import (
	"testing"

	"guppy/internal/gtasks"
)

func TestFlattenOrdersByPosition(t *testing.T) {
	rows := Flatten([]gtasks.Task{
		{ID: "b", Title: "Second", Position: "2"},
		{ID: "a", Title: "First", Position: "1"},
	})
	if len(rows) != 2 || rows[0].Task.ID != "a" || rows[1].Task.ID != "b" {
		t.Fatalf("unexpected order: %+v", rows)
	}
	if rows[0].Depth != 0 || rows[1].Depth != 0 {
		t.Fatalf("unexpected depths: %+v", rows)
	}
}

func TestFlattenNestsChildren(t *testing.T) {
	rows := Flatten([]gtasks.Task{
		{ID: "a", Title: "Parent", Position: "1"},
		{ID: "c2", Title: "Child 2", Position: "3", Parent: "a"},
		{ID: "c1", Title: "Child 1", Position: "2", Parent: "a"},
		{ID: "b", Title: "Other root", Position: "4"},
	})

	want := []struct {
		id    string
		depth int
	}{
		{"a", 0}, {"c1", 1}, {"c2", 1}, {"b", 0},
	}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		if rows[i].Task.ID != w.id || rows[i].Depth != w.depth {
			t.Fatalf("row %d = %s@%d, want %s@%d", i, rows[i].Task.ID, rows[i].Depth, w.id, w.depth)
		}
	}
}

func TestFlattenOrphanPromotedToRoot(t *testing.T) {
	rows := Flatten([]gtasks.Task{
		{ID: "orphan", Title: "No parent here", Position: "1", Parent: "missing"},
	})
	if len(rows) != 1 || rows[0].Depth != 0 {
		t.Fatalf("orphan not promoted: %+v", rows)
	}
}

func TestFlattenEmpty(t *testing.T) {
	if rows := Flatten(nil); len(rows) != 0 {
		t.Fatalf("want no rows, got %+v", rows)
	}
}
