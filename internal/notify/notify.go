package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Send sends a desktop notification. macOS only (osascript).
// Falls back silently on other platforms.
func Send(title, message string) {
	if runtime.GOOS != "darwin" {
		return
	}

	script := fmt.Sprintf(`display notification %q with title %q`, message, title)
	exec.Command("osascript", "-e", script).Run()
}
