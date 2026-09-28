package gtasks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"google.golang.org/api/option"
	tasks "google.golang.org/api/tasks/v1"
)

func testService(t *testing.T, h http.Handler) Service {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	api, err := tasks.NewService(context.Background(),
		option.WithEndpoint(srv.URL+"/"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return &googleService{api: api}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestListsPaginates(t *testing.T) {
	var gotTokens []string
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/v1/users/@me/lists" {
			http.NotFound(w, r)
			return
		}
		gotTokens = append(gotTokens, r.URL.Query().Get("pageToken"))
		if r.URL.Query().Get("pageToken") == "" {
			writeJSON(t, w, map[string]any{
				"items":         []map[string]string{{"id": "1", "title": "One"}},
				"nextPageToken": "p2",
			})
			return
		}
		writeJSON(t, w, map[string]any{
			"items": []map[string]string{{"id": "2", "title": "Two"}},
		})
	}))

	lists, err := svc.Lists(context.Background())
	if err != nil {
		t.Fatalf("Lists: %v", err)
	}
	if len(lists) != 2 || lists[0].Title != "One" || lists[1].ID != "2" {
		t.Fatalf("unexpected lists: %+v", lists)
	}
	if len(gotTokens) != 2 || gotTokens[1] != "p2" {
		t.Fatalf("pagination not followed: %v", gotTokens)
	}
}

func TestTasksMapping(t *testing.T) {
	var gotQuery url.Values
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/v1/lists/L1/tasks" {
			http.NotFound(w, r)
			return
		}
		gotQuery = r.URL.Query()
		writeJSON(t, w, map[string]any{
			"items": []map[string]string{
				{"id": "a", "title": "Parent", "status": "needsAction", "position": "1"},
				{"id": "b", "title": "Done", "status": "completed", "position": "2"},
				{"id": "c", "title": "Child", "status": "needsAction", "position": "3", "parent": "a"},
			},
		})
	}))

	tasksOut, err := svc.Tasks(context.Background(), "L1")
	if err != nil {
		t.Fatalf("Tasks: %v", err)
	}
	if len(tasksOut) != 3 {
		t.Fatalf("want 3 tasks, got %d", len(tasksOut))
	}
	if tasksOut[1].Completed != true || tasksOut[0].Completed != false {
		t.Fatalf("completed mapping wrong: %+v", tasksOut)
	}
	if tasksOut[2].Parent != "a" {
		t.Fatalf("parent mapping wrong: %+v", tasksOut[2])
	}
	if gotQuery.Get("showCompleted") != "true" || gotQuery.Get("showHidden") != "false" {
		t.Fatalf("unexpected query: %v", gotQuery)
	}
}

func TestAddSubtaskSendsParentParam(t *testing.T) {
	var gotParent string
	var gotBody map[string]any
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/v1/lists/L1/tasks" {
			http.NotFound(w, r)
			return
		}
		gotParent = r.URL.Query().Get("parent")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		writeJSON(t, w, map[string]any{
			"id": "new", "title": gotBody["title"], "status": "needsAction",
			"position": "1", "parent": gotParent,
		})
	}))

	created, err := svc.Add(context.Background(), "L1", "Child task", "p9")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if gotParent != "p9" {
		t.Fatalf("parent param = %q, want p9", gotParent)
	}
	if gotBody["title"] != "Child task" {
		t.Fatalf("body title = %v", gotBody["title"])
	}
	if created.ID != "new" || created.Parent != "p9" {
		t.Fatalf("unexpected created task: %+v", created)
	}
}

func TestAddTopLevelOmitsParent(t *testing.T) {
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.URL.Query()["parent"]; present {
			t.Errorf("parent param should be omitted for top-level tasks")
		}
		writeJSON(t, w, map[string]any{"id": "x", "title": "T", "status": "needsAction", "position": "1"})
	}))
	if _, err := svc.Add(context.Background(), "L1", "T", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
}

func TestRenameSendsPatch(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		writeJSON(t, w, map[string]any{"id": "t1", "title": gotBody["title"], "status": "needsAction", "position": "1"})
	}))

	if err := svc.Rename(context.Background(), "L1", "t1", "Renamed"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("method = %s, want PATCH", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/tasks/v1/lists/L1/tasks/t1") {
		t.Fatalf("path = %s", gotPath)
	}
	if gotBody["title"] != "Renamed" {
		t.Fatalf("body = %v", gotBody)
	}
}

func TestSetCompletedSendsStatus(t *testing.T) {
	var gotBody map[string]any
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		writeJSON(t, w, map[string]any{"id": "t1", "title": "T", "status": gotBody["status"], "position": "1"})
	}))

	if err := svc.SetCompleted(context.Background(), "L1", "t1", true); err != nil {
		t.Fatalf("SetCompleted: %v", err)
	}
	if gotBody["status"] != "completed" {
		t.Fatalf("status = %v", gotBody["status"])
	}
	if err := svc.SetCompleted(context.Background(), "L1", "t1", false); err != nil {
		t.Fatalf("SetCompleted: %v", err)
	}
	if gotBody["status"] != "needsAction" {
		t.Fatalf("status = %v", gotBody["status"])
	}
}

func TestDeleteCallsDelete(t *testing.T) {
	var gotMethod, gotPath string
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	if err := svc.Delete(context.Background(), "L1", "t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if gotMethod != http.MethodDelete || !strings.HasSuffix(gotPath, "/tasks/v1/lists/L1/tasks/t1") {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
}

func TestAPIErrorPropagates(t *testing.T) {
	svc := testService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(t, w, map[string]any{"error": map[string]any{"code": 500, "message": "boom"}})
	}))
	if _, err := svc.Lists(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}
