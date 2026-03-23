package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tylercooper/ngram/internal/vault"
)

// ServiceFunc is a long-running goroutine function.
type ServiceFunc func(ctx context.Context) error

// Service describes a named background goroutine.
type Service struct {
	Name string
	Run  ServiceFunc
}

// Heartbeat is the _meta/heartbeat.json schema.
type Heartbeat struct {
	PID           int               `json:"pid"`
	StartedAt     string            `json:"started_at"`
	LastHeartbeat string            `json:"last_heartbeat"`
	Goroutines    map[string]string `json:"goroutines"`
	Engaged       bool              `json:"engaged"`
	EngagementName string           `json:"engagement_name,omitempty"`
}

// Daemon manages the lifecycle of background services.
type Daemon struct {
	VaultPath  string
	Services   []Service
	mu         sync.Mutex
	statuses   map[string]string
	engaged    bool
	engName    string
}

// New creates a Daemon with the given vault path.
func New(vaultPath string) *Daemon {
	return &Daemon{
		VaultPath: vaultPath,
		statuses:  make(map[string]string),
	}
}

// IsEngaged returns whether engagement mode is active.
func (d *Daemon) IsEngaged() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.engaged
}

// SetEngaged enables or disables engagement mode.
func (d *Daemon) SetEngaged(engaged bool, name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.engaged = engaged
	d.engName = name
}

// Run starts all services and blocks until SIGTERM/SIGINT.
func (d *Daemon) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Signal handling.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	var wg sync.WaitGroup

	// Start services.
	for _, svc := range d.Services {
		wg.Add(1)
		svc := svc
		d.setStatus(svc.Name, "healthy")
		go func() {
			defer wg.Done()
			if err := svc.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("error: %s: %v", svc.Name, err)
				d.setStatus(svc.Name, "error")
			} else {
				d.setStatus(svc.Name, "stopped")
			}
		}()
	}

	// Heartbeat writer.
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.heartbeatLoop(ctx)
	}()

	log.Printf("ngram: daemon started with %d services (PID %d)", len(d.Services), os.Getpid())

	// Block on signal.
	select {
	case sig := <-sigCh:
		log.Printf("ngram: received %s, shutting down...", sig)
		cancel()
	case <-ctx.Done():
	}

	// Wait for goroutines with timeout.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("warn: shutdown timeout, some goroutines may not have finished")
	}

	// Remove heartbeat file.
	os.Remove(filepath.Join(d.VaultPath, "_meta", "heartbeat.json"))
	log.Printf("ngram: daemon stopped")
	return nil
}

func (d *Daemon) setStatus(name, status string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.statuses[name] = status
}

func (d *Daemon) heartbeatLoop(ctx context.Context) {
	d.writeHeartbeat()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.writeHeartbeat()
		}
	}
}

func (d *Daemon) writeHeartbeat() {
	d.mu.Lock()
	hb := Heartbeat{
		PID:            os.Getpid(),
		StartedAt:      time.Now().UTC().Format(time.RFC3339),
		LastHeartbeat:  time.Now().UTC().Format(time.RFC3339),
		Goroutines:     make(map[string]string),
		Engaged:        d.engaged,
		EngagementName: d.engName,
	}
	for k, v := range d.statuses {
		hb.Goroutines[k] = v
	}
	d.mu.Unlock()

	data, _ := json.MarshalIndent(hb, "", "  ")
	metaDir := filepath.Join(d.VaultPath, "_meta")
	vault.EnsureDir(metaDir)
	os.WriteFile(filepath.Join(metaDir, "heartbeat.json"), data, 0o644)
}

// ReadHeartbeat reads the heartbeat file from the vault.
func ReadHeartbeat(vaultPath string) (*Heartbeat, error) {
	path := filepath.Join(vaultPath, "_meta", "heartbeat.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return nil, err
	}
	return &hb, nil
}

// IsRunning checks if a daemon is currently running by reading the heartbeat.
func IsRunning(vaultPath string) (bool, *Heartbeat) {
	hb, err := ReadHeartbeat(vaultPath)
	if err != nil {
		return false, nil
	}
	last, err := time.Parse(time.RFC3339, hb.LastHeartbeat)
	if err != nil {
		return false, nil
	}
	// Stale if older than 120 seconds.
	if time.Since(last) > 120*time.Second {
		return false, hb
	}
	return true, hb
}

// StartMeilisearch runs docker compose up -d and waits for health.
func StartMeilisearch(vaultPath string) error {
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = vaultPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}

	// Wait for health (up to 30s).
	for i := 0; i < 30; i++ {
		resp, err := exec.Command("curl", "-sf", "http://127.0.0.1:7700/health").Output()
		if err == nil && len(resp) > 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("meilisearch did not become healthy within 30s")
}

// StopMeilisearch runs docker compose down.
func StopMeilisearch(vaultPath string) error {
	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = vaultPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
