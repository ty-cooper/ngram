package pipeline

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ty-cooper/ngram/internal/vault"
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

	// Process existing files in _inbox/ that were added before the daemon started.
	w.drainExisting(ctx, inboxDir)

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

func (w *Watcher) drainExisting(ctx context.Context, inboxDir string) {
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			continue
		}
		p := filepath.Join(inboxDir, e.Name())
		// Accept .md files and capture bundle directories (contain manifest.yml).
		if !strings.HasSuffix(e.Name(), ".md") && !IsBundle(p) {
			continue
		}
		log.Printf("ngram: processing existing %s", e.Name())
		if err := w.Processor.Process(ctx, p); err != nil {
			log.Printf("error: process %s: %v", e.Name(), err)
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

					// Verify file/dir still exists (may have been moved already).
					info, err := os.Stat(path)
					if os.IsNotExist(err) {
						continue
					}

					// For directories, only process if they're capture bundles.
					if info != nil && info.IsDir() && !IsBundle(path) {
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
	// Accept .md files and capture bundle directories.
	if strings.HasSuffix(name, ".md") {
		return true
	}
	// Check if it's a directory (potential capture bundle). For fsnotify Create
	// events on directories, we delay processing until the manifest.yml is written.
	// The debounce handles this — by the time 500ms passes, the bundle is complete.
	if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
		return true
	}
	return false
}
