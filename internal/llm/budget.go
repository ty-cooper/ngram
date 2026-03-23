package llm

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrBudgetExceeded = errors.New("daily API call budget exceeded")

// Budget tracks API call counts against a daily limit.
type Budget struct {
	MaxCallsPerDay int
	WarnAtPercent  int
	VaultPath      string

	mu       sync.Mutex
	today    string
	count    int
	loaded   bool
}

// Check returns an error if the daily budget is exceeded.
// Logs a warning if approaching the limit.
func (b *Budget) Check() error {
	if b.MaxCallsPerDay <= 0 {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.ensureLoaded()

	if b.count >= b.MaxCallsPerDay {
		return fmt.Errorf("%w: %d/%d calls today", ErrBudgetExceeded, b.count, b.MaxCallsPerDay)
	}
	return nil
}

// Increment records one API call.
func (b *Budget) Increment() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ensureLoaded()
	b.count++
}

// Count returns today's call count.
func (b *Budget) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ensureLoaded()
	return b.count
}

func (b *Budget) ensureLoaded() {
	today := time.Now().Format("2006-01-02")
	if b.loaded && b.today == today {
		return
	}

	b.today = today
	b.count = countCallsForDate(b.VaultPath, today)
	b.loaded = true
}

// countCallsForDate counts API usage log entries for a given date.
func countCallsForDate(vaultPath, date string) int {
	path := filepath.Join(vaultPath, "_meta", "api-usage.jsonl")
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, date) {
			var entry struct {
				Timestamp string `json:"timestamp"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil {
				if strings.HasPrefix(entry.Timestamp, date) {
					count++
				}
			}
		}
	}
	return count
}
