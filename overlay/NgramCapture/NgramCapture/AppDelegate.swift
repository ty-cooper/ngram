import Cocoa
import SwiftUI
import HotKey

class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var hotKey: HotKey?
    private var capturePanel: NSPanel?
    private var sessionManager = CaptureSessionManager()

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Global hotkey: Cmd+Option+N via soffes/HotKey.
        hotKey = HotKey(key: .n, modifiers: [.command, .option])
        hotKey?.keyDownHandler = { [weak self] in
            self?.showCapturePicker()
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return false
    }

    @objc func showCapturePicker() {
        if capturePanel != nil {
            capturePanel?.close()
            capturePanel = nil
        }
        startSession()
    }

    func startSession() {
        let sessionView = CaptureSessionView(manager: sessionManager) { [weak self] in
            self?.capturePanel?.close()
            self?.capturePanel = nil
        }

        sessionManager.startSession()

        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 500, height: 450),
            styleMask: [.titled, .closable, .nonactivatingPanel, .utilityWindow],
            backing: .buffered,
            defer: false
        )
        panel.title = "Ngram Capture Session"
        panel.level = .floating
        panel.isFloatingPanel = true
        panel.contentView = NSHostingView(rootView: sessionView)

        // Position top-right.
        if let screen = NSScreen.main {
            let x = screen.visibleFrame.maxX - 520
            let y = screen.visibleFrame.maxY - 470
            panel.setFrameOrigin(NSPoint(x: x, y: y))
        }

        panel.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        capturePanel = panel
    }
}
