package imessage

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MessageBridge defines the interface for sending and receiving iMessages.
type MessageBridge interface {
	Send(phone string, text string) error
	Poll(since time.Time) ([]IncomingMessage, error)
}

// IncomingMessage is a received iMessage.
type IncomingMessage struct {
	From      string
	Body      string
	Timestamp time.Time
}

// AppleScriptBridge uses osascript for sending and chat.db for receiving.
type AppleScriptBridge struct {
	ChatDBPath string // defaults to ~/Library/Messages/chat.db
}

// NewAppleScriptBridge creates a bridge using the native macOS Messages stack.
func NewAppleScriptBridge() *AppleScriptBridge {
	home, _ := os.UserHomeDir()
	return &AppleScriptBridge{
		ChatDBPath: filepath.Join(home, "Library", "Messages", "chat.db"),
	}
}

// Send sends an iMessage via osascript.
func (b *AppleScriptBridge) Send(phone, text string) error {
	script := fmt.Sprintf(`tell application "Messages"
	set targetBuddy to buddy "%s" of service id "iMessage"
	send %q to targetBuddy
end tell`, phone, text)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript send: %w: %s", err, out)
	}
	return nil
}

// Poll queries chat.db for messages received after the given time.
// Requires Full Disk Access for the calling process.
func (b *AppleScriptBridge) Poll(since time.Time) ([]IncomingMessage, error) {
	db, err := sql.Open("sqlite3", b.ChatDBPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open chat.db: %w", err)
	}
	defer db.Close()

	// chat.db stores dates as nanoseconds since 2001-01-01 (Core Data epoch).
	coreDataEpoch := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	sinceCD := since.Sub(coreDataEpoch).Nanoseconds()

	query := `
		SELECT
			h.id AS phone,
			m.text,
			m.date
		FROM message m
		JOIN handle h ON m.handle_id = h.ROWID
		WHERE m.is_from_me = 0
			AND m.date > ?
			AND m.text IS NOT NULL
			AND m.text != ''
		ORDER BY m.date ASC
	`

	rows, err := db.Query(query, sinceCD)
	if err != nil {
		return nil, fmt.Errorf("query chat.db: %w", err)
	}
	defer rows.Close()

	var msgs []IncomingMessage
	for rows.Next() {
		var phone, text string
		var dateNS int64
		if err := rows.Scan(&phone, &text, &dateNS); err != nil {
			continue
		}
		ts := coreDataEpoch.Add(time.Duration(dateNS))
		msgs = append(msgs, IncomingMessage{
			From:      phone,
			Body:      text,
			Timestamp: ts,
		})
	}

	return msgs, nil
}
