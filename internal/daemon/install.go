package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Install creates a system service for the daemon.
func Install(binaryPath, vaultPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(binaryPath, vaultPath)
	case "linux":
		return installSystemd(binaryPath, vaultPath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Uninstall removes the system service.
func Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd()
	case "linux":
		return uninstallSystemd()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

const launchdLabel = "com.ngram.n"

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

func installLaunchd(binaryPath, vaultPath string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>up</string>
        <string>--foreground</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/_meta/ngram.log</string>
    <key>StandardErrorPath</key>
    <string>%s/_meta/ngram.log</string>
</dict>
</plist>`, launchdLabel, binaryPath, vaultPath, vaultPath)

	path := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}

	cmd := exec.Command("launchctl", "load", path)
	return cmd.Run()
}

func uninstallLaunchd() error {
	path := launchdPlistPath()
	exec.Command("launchctl", "unload", path).Run()
	return os.Remove(path)
}

const systemdServiceName = "ngram"

func systemdUnitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", systemdServiceName+".service")
}

func installSystemd(binaryPath, vaultPath string) error {
	unit := strings.Join([]string{
		"[Unit]",
		"Description=Ngram Knowledge System",
		"After=docker.service",
		"",
		"[Service]",
		fmt.Sprintf("ExecStart=%s up --foreground", binaryPath),
		fmt.Sprintf("WorkingDirectory=%s", vaultPath),
		"Restart=always",
		"RestartSec=5",
		"",
		"[Install]",
		"WantedBy=default.target",
	}, "\n")

	path := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return exec.Command("systemctl", "--user", "enable", "--now", systemdServiceName).Run()
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "disable", "--now", systemdServiceName).Run()
	return os.Remove(systemdUnitPath())
}
