package services

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"tavily-proxy/server/internal/db"
)

func TestKeyBatchCreateJobService_StartAndComplete(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	database, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	keys := NewKeyService(database, logger)
	jobs := NewKeyBatchCreateJobService(keys, logger)

	started, alreadyRunning, err := jobs.Start([]string{
		"tvly-test-1",
		"tvly-test-2",
		"tvly-test-1", // duplicate should be ignored
		"",
		"  ",
	}, "", 1000)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if alreadyRunning {
		t.Fatalf("unexpected alreadyRunning=true")
	}
	if started.Status != "running" {
		t.Fatalf("unexpected status: %q", started.Status)
	}
	if started.Total != 2 {
		t.Fatalf("unexpected total: got %d want %d", started.Total, 2)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		got := jobs.Get()
		if got.Status != "running" {
			if got.Status != "completed" {
				t.Fatalf("unexpected final status: %q", got.Status)
			}
			if got.Completed != 2 || got.Succeeded != 2 || got.Failed != 0 {
				t.Fatalf("unexpected counters: %+v", got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for completion")
		}
		time.Sleep(10 * time.Millisecond)
	}

	items, err := keys.List(context.Background())
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected key count: got %d want %d", len(items), 2)
	}
}

func TestKeyBatchCreateJobService_Failures(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	database, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	keys := NewKeyService(database, logger)
	jobs := NewKeyBatchCreateJobService(keys, logger)

	if _, err := keys.Create(context.Background(), "tvly-exists", "Default", 1000); err != nil {
		t.Fatalf("seed key: %v", err)
	}

	started, _, err := jobs.Start([]string{"tvly-exists", "tvly-new"}, "", 1000)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Total != 2 {
		t.Fatalf("unexpected total: got %d want %d", started.Total, 2)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		got := jobs.Get()
		if got.Status != "running" {
			if got.Status != "completed" {
				t.Fatalf("unexpected final status: %q", got.Status)
			}
			if got.Completed != 2 || got.Succeeded != 1 || got.Failed != 1 {
				t.Fatalf("unexpected counters: %+v", got)
			}
			if len(got.Failures) != 1 || got.Failures[0].Key != "tvly-exists" {
				t.Fatalf("unexpected failures: %+v", got.Failures)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for completion")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestKeyBatchCreateJobService_ValidateInput(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	database, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	keys := NewKeyService(database, logger)
	jobs := NewKeyBatchCreateJobService(keys, logger)

	if _, _, err := jobs.Start(nil, "", 1000); err != ErrBatchCreateNoKeys {
		t.Fatalf("unexpected error for empty keys: %v", err)
	}

	raw := make([]string, maxBatchCreateKeys+1)
	for i := range raw {
		raw[i] = "tvly-bulk-" + strconv.Itoa(i)
	}
	if _, _, err := jobs.Start(raw, "", 1000); err != ErrBatchCreateTooLarge {
		t.Fatalf("unexpected error for oversized batch: %v", err)
	}
}
