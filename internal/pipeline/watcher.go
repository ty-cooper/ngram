package pipeline

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tylercooper/ngram/internal/vault"
)

// Watcher monitors _inbox/ for new files and dispatches them to the processor.
type Watcher struct {
	VaultPath string
	Processor *Processor
}

// Start begins watching _inbox/ for new files. Blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	inboxDir, err := vault.InboxDir(w.VaultPath)
	if err != nil {
		return err
	}

	// Crash recovery: move orphaned _processing/ files back to _inbox/.
	w.recoverOrphans()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(inboxDir); err != nil {
		return err
	}

	log.Printf("ngram: watching %s", inboxDir)
	return w.debouncedWatch(ctx, watcher)
}

func (w *Watcher) recoverOrphans() {
	procDir := filepath.Join(w.VaultPath, "_processing")
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return
	}

	inboxDir := filepath.Join(w.VaultPath, "_inbox")
	vault.EnsureDir(inboxDir)

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			os.Remove(filepath.Join(procDir, e.Name()))
			continue
		}
		src := filepath.Join(procDir, e.Name())
		dst := filepath.Join(inboxDir, e.Name())
		if err := os.Rename(src, dst); err != nil {
			log.Printf("warn: recover orphan %s: %v", e.Name(), err)
		} else {
			log.Printf("ngram: recovered orphan %s", e.Name())
		}
	}
}

func (w *Watcher) debouncedWatch(ctx context.Context, watcher *fsnotify.Watcher) error {
	pending := make(map[string]time.Time)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !isProcessable(event) {
				continue
			}
			pending[event.Name] = time.Now()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("warn: fsnotify error: %v", err)

		case <-ticker.C:
			now := time.Now()
			for path, last := range pending {
				if now.Sub(last) > 500*time.Millisecond {
					delete(pending, path)

					// Verify file still exists (may have been moved already).
					if _, err := os.Stat(path); os.IsNotExist(err) {
						continue
					}

					go func(p string) {
						if err := w.Processor.Process(ctx, p); err != nil {
							log.Printf("error: process %s: %v", filepath.Base(p), err)
						}
					}(path)
				}
			}
		}
	}
}

func isProcessable(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return false
	}
	name := filepath.Base(event.Name)
	if strings.HasPrefix(name, ".tmp-") {
		return false
	}
	if !strings.HasSuffix(name, ".md") {
		return false
	}
	return true
}
