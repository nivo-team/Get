// Package ui builds the native NSGo desktop UI: the main task list, the shared
// progress window, the file-info queue window and the settings view.
//
// 这一层只做展示与把用户操作交回 Backend：任务状态、队列与持久化都以 Go 后端为准。
package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/ui"
	"sheep-get/internal/task"
)

// TaskFilter represents the sidebar filter selection in the main window.
type TaskFilter string

const (
	FilterAll         TaskFilter = "all"
	FilterDownloading TaskFilter = "downloading"
	FilterCompleted   TaskFilter = "completed"
	FilterPaused      TaskFilter = "paused"
	FilterFailed      TaskFilter = "failed"
	FilterSettings    TaskFilter = "settings"
)

// filterTasks returns a sorted copy of tasks matching the active filter.
func filterTasks(tasks []*task.Task, filter TaskFilter) []*task.Task {
	out := make([]*task.Task, 0, len(tasks))
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if matchesFilter(t.Status, filter) {
			out = append(out, t)
		}
	}
	slices.SortStableFunc(out, func(a, b *task.Task) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		return 1
	})
	return out
}

func matchesFilter(status task.Status, filter TaskFilter) bool {
	switch filter {
	case FilterDownloading:
		return status == task.StatusDownloading || status == task.StatusQueued ||
			status == task.StatusProcessing || status == task.StatusCollecting
	case FilterCompleted:
		return status == task.StatusCompleted
	case FilterPaused:
		return status == task.StatusPaused
	case FilterFailed:
		return status == task.StatusError
	default:
		return true
	}
}

func statusMeta(status task.Status) (string, ui.BadgeVariant) {
	switch status {
	case task.StatusDownloading:
		return "下载中", ui.BadgeAccent
	case task.StatusQueued:
		return "排队中", ui.BadgeNeutral
	case task.StatusProcessing:
		return "处理中", ui.BadgeAccent
	case task.StatusCollecting:
		return "收集中", ui.BadgeNeutral
	case task.StatusPaused:
		return "已暂停", ui.BadgeWarn
	case task.StatusCompleted:
		return "已完成", ui.BadgeSuccess
	case task.StatusError:
		return "失败", ui.BadgeDanger
	default:
		return string(status), ui.BadgeNeutral
	}
}

func taskProgressPercent(t *task.Task) float32 {
	if t == nil {
		return 0
	}
	if t.Status == task.StatusCompleted {
		return 100
	}
	if t.TotalBytes <= 0 {
		return 0
	}
	pct := float64(t.Downloaded) * 100 / float64(t.TotalBytes)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return float32(pct)
}

func formatBytes(n int64) string {
	if n < 0 {
		return "未知"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 4; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGT"[exp])
}

func formatSpeed(bytesPerSec int64) string {
	if bytesPerSec <= 0 {
		return ""
	}
	return formatBytes(bytesPerSec) + "/s"
}

func formatETA(t *task.Task) string {
	if t == nil || t.Speed <= 0 || t.TotalBytes <= 0 || t.Downloaded >= t.TotalBytes {
		return ""
	}
	secs := (t.TotalBytes - t.Downloaded) / t.Speed
	if secs <= 0 {
		return ""
	}
	d := time.Duration(secs) * time.Second
	if d >= time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if d >= time.Minute {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func formatTaskSubtitle(t *task.Task) string {
	if t == nil {
		return ""
	}
	if t.Status == task.StatusError && t.ErrorMsg != "" {
		return t.ErrorMsg
	}
	sizePart := formatBytes(t.Downloaded)
	if t.TotalBytes > 0 {
		sizePart = fmt.Sprintf("%s / %s (%.1f%%)", formatBytes(t.Downloaded), formatBytes(t.TotalBytes), taskProgressPercent(t))
	} else if t.Status != task.StatusCompleted {
		sizePart = fmt.Sprintf("%s / 未知大小", formatBytes(t.Downloaded))
	}
	parts := []string{sizePart}
	if t.Status == task.StatusDownloading {
		if spd := formatSpeed(t.Speed); spd != "" {
			parts = append(parts, spd)
		}
		if eta := formatETA(t); eta != "" {
			parts = append(parts, "剩余 "+eta)
		}
	}
	return strings.Join(parts, " · ")
}

func formatChunkSummary(t *task.Task) string {
	if t == nil {
		return ""
	}
	if t.SegmentsTotal > 0 {
		return fmt.Sprintf("分片 %d/%d", t.SegmentsDone, t.SegmentsTotal)
	}
	if len(t.Chunks) > 1 {
		done := 0
		for _, c := range t.Chunks {
			if c.Completed {
				done++
			}
		}
		return fmt.Sprintf("并发分块 %d/%d", done, len(t.Chunks))
	}
	if t.MaxConcurrency > 0 && t.Status == task.StatusDownloading {
		return fmt.Sprintf("最大并发 %d", t.MaxConcurrency)
	}
	return ""
}

// progressWindowTasks partitions tasks for the shared progress window:
// completed items (recently finished and not dismissed) stay at the top,
// followed by active/queued/processing items sorted by progress descending.
func progressWindowTasks(tasks []*task.Task, activeIDs, dismissedIDs map[string]struct{}) (completed, active []*task.Task) {
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if _, dismissed := dismissedIDs[t.ID]; dismissed {
			continue
		}
		switch t.Status {
		case task.StatusDownloading, task.StatusProcessing, task.StatusQueued, task.StatusCollecting:
			active = append(active, t)
		case task.StatusCompleted, task.StatusPaused, task.StatusError:
			if _, tracked := activeIDs[t.ID]; tracked {
				if t.Status == task.StatusCompleted {
					completed = append(completed, t)
				} else {
					active = append(active, t)
				}
			}
		}
	}
	slices.SortStableFunc(completed, func(a, b *task.Task) int {
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
	slices.SortStableFunc(active, func(a, b *task.Task) int {
		pa, pb := taskProgressPercent(a), taskProgressPercent(b)
		if pa > pb {
			return -1
		}
		if pa < pb {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
	return completed, active
}
