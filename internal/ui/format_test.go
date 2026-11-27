package ui

import (
	"testing"
	"time"

	"sheep-get/internal/task"
)

func TestFilterTasksAndSort(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: "1", Status: task.StatusCompleted, CreatedAt: now.Add(-2 * time.Minute)},
		{ID: "2", Status: task.StatusDownloading, CreatedAt: now},
		{ID: "3", Status: task.StatusPaused, CreatedAt: now.Add(-1 * time.Minute)},
		{ID: "4", Status: task.StatusError, CreatedAt: now.Add(-3 * time.Minute)},
	}

	all := filterTasks(tasks, FilterAll)
	if len(all) != 4 || all[0].ID != "2" || all[1].ID != "3" {
		t.Fatalf("unexpected FilterAll order: %+v", all)
	}
	if got := filterTasks(tasks, FilterDownloading); len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("unexpected FilterDownloading result: %+v", got)
	}
	if got := filterTasks(tasks, FilterCompleted); len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("unexpected FilterCompleted result: %+v", got)
	}
	if got := filterTasks(tasks, FilterPaused); len(got) != 1 || got[0].ID != "3" {
		t.Fatalf("unexpected FilterPaused result: %+v", got)
	}
	if got := filterTasks(tasks, FilterFailed); len(got) != 1 || got[0].ID != "4" {
		t.Fatalf("unexpected FilterFailed result: %+v", got)
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := formatBytes(1536); got != "1.5 KiB" {
		t.Fatalf("formatBytes(1536) = %q, want 1.5 KiB", got)
	}
	if got := taskProgressPercent(&task.Task{TotalBytes: 200, Downloaded: 50}); got != 25 {
		t.Fatalf("taskProgressPercent = %v, want 25", got)
	}
	if got := taskProgressPercent(&task.Task{Status: task.StatusCompleted}); got != 100 {
		t.Fatalf("completed taskProgressPercent = %v, want 100", got)
	}
	if label, _ := statusMeta(task.StatusDownloading); label != "下载中" {
		t.Fatalf("statusMeta(downloading) = %q", label)
	}
	if got := formatChunkSummary(&task.Task{SegmentsDone: 3, SegmentsTotal: 10}); got != "分片 3/10" {
		t.Fatalf("formatChunkSummary = %q", got)
	}
}

func TestProgressWindowTasks(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: "a", Status: task.StatusDownloading, TotalBytes: 100, Downloaded: 20},
		{ID: "b", Status: task.StatusDownloading, TotalBytes: 100, Downloaded: 80},
		{ID: "c", Status: task.StatusCompleted, UpdatedAt: now},
		{ID: "d", Status: task.StatusCompleted, UpdatedAt: now.Add(-time.Minute)},
	}
	activeIDs := map[string]struct{}{"c": {}, "d": {}}
	dismissedIDs := map[string]struct{}{"d": {}}

	comp, act := progressWindowTasks(tasks, activeIDs, dismissedIDs)
	if len(comp) != 1 || comp[0].ID != "c" {
		t.Fatalf("unexpected completed tasks: %+v", comp)
	}
	if len(act) != 2 || act[0].ID != "b" || act[1].ID != "a" {
		t.Fatalf("expected active tasks sorted descending by progress, got %+v", act)
	}
}
