package storage

import (
	"testing"
)

func TestDownloadTaskTableName(t *testing.T) {
	dt := DownloadTask{}
	if dt.TableName() != "download_tasks" {
		t.Errorf("TableName() = %q, want %q", dt.TableName(), "download_tasks")
	}
}

func TestDownloadLocationTableName(t *testing.T) {
	dl := DownloadLocation{}
	if dl.TableName() != "download_locations" {
		t.Errorf("TableName() = %q, want %q", dl.TableName(), "download_locations")
	}
}

func TestDailyStatTableName(t *testing.T) {
	ds := DailyStat{}
	if ds.TableName() != "daily_stats" {
		t.Errorf("TableName() = %q, want %q", ds.TableName(), "daily_stats")
	}
}

func TestAppSettingTableName(t *testing.T) {
	as := AppSetting{}
	if as.TableName() != "app_settings" {
		t.Errorf("TableName() = %q, want %q", as.TableName(), "app_settings")
	}
}

func TestSpeedTestHistoryTableName(t *testing.T) {
	sth := SpeedTestHistory{}
	if sth.TableName() != "speed_test_history" {
		t.Errorf("TableName() = %q, want %q", sth.TableName(), "speed_test_history")
	}
}

func TestTaskTypeAlias(t *testing.T) {
	// Task is an alias for DownloadTask — verify assignment works
	var task Task
	task.ID = "test-123"
	task.Filename = "file.zip"
	task.Status = "pending"

	if task.ID != "test-123" {
		t.Errorf("Task.ID = %q, want %q", task.ID, "test-123")
	}
	if task.Filename != "file.zip" {
		t.Errorf("Task.Filename = %q, want %q", task.Filename, "file.zip")
	}
	if task.Status != "pending" {
		t.Errorf("Task.Status = %q, want %q", task.Status, "pending")
	}
}

func TestPartState(t *testing.T) {
	ps := PartState{
		Start:    0,
		End:      1024,
		Complete: true,
		Offset:   512,
	}
	if ps.Start != 0 {
		t.Errorf("Start = %d, want 0", ps.Start)
	}
	if ps.End != 1024 {
		t.Errorf("End = %d, want 1024", ps.End)
	}
	if !ps.Complete {
		t.Error("Complete should be true")
	}
	if ps.Offset != 512 {
		t.Errorf("Offset = %d, want 512", ps.Offset)
	}
}

func TestResumeState(t *testing.T) {
	rs := ResumeState{
		Version:      1,
		ETag:         "abc123",
		LastModified: "Wed, 01 Jan 2025 00:00:00 GMT",
		TotalSize:    1024 * 1024,
		Parts: map[int]PartState{
			0: {Start: 0, End: 512, Complete: true},
			1: {Start: 512, End: 1024, Complete: false, Offset: 100},
		},
	}
	if rs.Version != 1 {
		t.Errorf("Version = %d, want 1", rs.Version)
	}
	if rs.ETag != "abc123" {
		t.Errorf("ETag = %q, want %q", rs.ETag, "abc123")
	}
	if rs.LastModified != "Wed, 01 Jan 2025 00:00:00 GMT" {
		t.Errorf("LastModified = %q, want %q", rs.LastModified, "Wed, 01 Jan 2025 00:00:00 GMT")
	}
	if rs.TotalSize != 1024*1024 {
		t.Errorf("TotalSize = %d, want %d", rs.TotalSize, 1024*1024)
	}
	if len(rs.Parts) != 2 {
		t.Errorf("Parts len = %d, want 2", len(rs.Parts))
	}
	if !rs.Parts[0].Complete {
		t.Error("Part 0 should be complete")
	}
	if rs.Parts[1].Complete {
		t.Error("Part 1 should not be complete")
	}
}
