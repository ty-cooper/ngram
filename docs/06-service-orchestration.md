# Service Orchestration

The `n` binary is both the CLI and the daemon. One process, goroutines for background services, `context.Context` for lifecycle management.

## `n up`

1. Starts Meilisearch via `docker compose up -d`
2. Waits for Meilisearch healthcheck (`GET localhost:7700/health`)
3. Spawns goroutines with a shared root context:
   - **processor**: watches `_inbox/` via fsnotify, structures notes via Claude
   - **indexer**: watches `knowledge/`, `boxes/`, `tools/`, updates Meilisearch on file changes
   - **quiz-scheduler**: builds daily queue at wake_hour, dispatches questions at scheduled times
   - **message-watcher**: polls iMessage bridge every 30 seconds, matches replies to pending questions
   - **stats-builder**: periodically computes retention stats, writes `_meta/stats-cache.json`
   - **backup**: runs `git push` hourly
4. Writes heartbeat to `_meta/heartbeat.json` every 30 seconds
5. Blocks on signal (SIGTERM, SIGINT)

## `n down`

1. Sends SIGTERM to the `n` process (reads PID from heartbeat file)
2. Signal handler cancels the root context
3. Each goroutine receives cancellation and shuts down:
   - quiz-scheduler writes state to `_meta/quiz-delivery-state.json`
   - processor finishes any in-flight note (never abandons mid-processing)
   - backup runs one final `git push`
4. After goroutines exit, runs `docker compose down`
5. Removes heartbeat file

Total shutdown: under 5 seconds.

## `n status`

Reads heartbeat file and reports goroutine health:

```
$ n status

[ngram] Services:
  ✓ meilisearch        running  (docker, port 7700, uptime 3d 14h)
  ✓ processor          running  (goroutine, 847 notes processed)
  ✓ indexer            running  (goroutine, watching structured/)
  ✓ quiz-scheduler     running  (goroutine, 6 sent today, 2 pending)
  ✓ message-watcher    running  (goroutine, last poll 12s ago)
  ✓ stats-builder      running  (goroutine)
  ✓ backup             running  (goroutine, last push 42m ago)
```

Heartbeat older than 120 seconds or missing = process not running.

### Heartbeat File

```json
{
  "pid": 4821,
  "started_at": "2026-03-23T08:00:00Z",
  "last_heartbeat": "2026-03-23T14:22:30Z",
  "goroutines": {
    "processor": "healthy",
    "indexer": "healthy",
    "quiz-scheduler": "healthy",
    "message-watcher": "healthy",
    "backup": "healthy"
  }
}
```

## Boot Persistence

`n up --install` installs a system service so `n` starts on boot and restarts on crash.

**macOS (launchd):**

```bash
n up --install      # creates ~/Library/LaunchAgents/com.ngram.n.plist, loads it
n up --uninstall    # unloads and removes the plist
```

Plist runs `n up --foreground` with `KeepAlive: true` and `RunAtLoad: true`.

**Linux (systemd):**

```bash
n up --install      # creates ~/.config/systemd/user/ngram.service, enables it
n up --uninstall    # disables and removes the unit
```

## Engagement Mode

```bash
n engage htb-optimum    # pauses quiz-scheduler and message-watcher
n disengage             # resumes them
```

Sets a flag checked by quiz-scheduler and message-watcher on each iteration. They skip their work loop when set. Processing and indexing continue normally.

## fsnotify Debouncing

500ms debounce window. Events accumulate in a map. After 500ms of quiet, accumulated events process. Prevents double-processing when editors auto-save rapidly.

```go
func debouncedWatch(ctx context.Context, watcher *fsnotify.Watcher, handler func(string)) {
    pending := make(map[string]time.Time)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case event := <-watcher.Events:
            pending[event.Name] = time.Now()
        case <-ticker.C:
            now := time.Now()
            for path, last := range pending {
                if now.Sub(last) > 500*time.Millisecond {
                    handler(path)
                    delete(pending, path)
                }
            }
        case <-ctx.Done():
            return
        }
    }
}
```
