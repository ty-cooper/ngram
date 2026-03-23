package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDaemon_StartStop(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "_meta"), 0o755)

	d := New(dir)
	d.Services = []Service{
		{
			Name: "test-service",
			Run: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := d.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Heartbeat file should be removed after shutdown.
	hbPath := filepath.Join(dir, "_meta", "heartbeat.json")
	if _, err := os.Stat(hbPath); !os.IsNotExist(err) {
		t.Error("heartbeat.json should be removed after shutdown")
	}
}

func TestReadHeartbeat_Missing(t *testing.T) {
	_, err := ReadHeartbeat(t.TempDir())
	if err == nil {
		t.Error("expected error for missing heartbeat")
	}
}

func TestIsRunning_StaleHeartbeat(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "_meta"), 0o755)

	// Write a stale heartbeat (5 minutes ago).
	stale := Heartbeat{
		PID:           99999,
		LastHeartbeat: time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
		Goroutines:    map[string]string{"test": "healthy"},
	}
	data, _ := json.MarshalIndent(stale, "", "  ")
	os.WriteFile(filepath.Join(dir, "_meta", "heartbeat.json"), data, 0o644)

	running, _ := IsRunning(dir)
	if running {
		t.Error("stale heartbeat should report not running")
	}
}
