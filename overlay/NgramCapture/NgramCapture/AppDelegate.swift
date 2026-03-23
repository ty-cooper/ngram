import Cocoa
import SwiftUI
import Carbon.HIToolbox

class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var hotKeyRef: EventHotKeyRef?
    private var capturePanel: NSPanel?
    private var sessionManager = CaptureSessionManager()

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Menu bar icon.
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        if let button = statusItem.button {
            button.image = NSImage(systemSymbolName: "brain.head.profile", accessibilityDescription: "Ngram")
        }

        let menu = NSMenu()
        menu.addItem(NSMenuItem(title: "Capture (⌘⌥N)", action: #selector(showCapturePicker), keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(NSMenuItem(title: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        statusItem.menu = menu

        // Register global hotkey: Cmd+Option+N via Carbon Events.
        registerHotKey()
    }

    private func registerHotKey() {
        var hotKeyID = EventHotKeyID()
        hotKeyID.signature = OSType(0x4E475243) // "NGRC"
        hotKeyID.id = 1

        var eventType = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))

        let handler: EventHandlerUPP = { _, event, _ -> OSStatus in
            DispatchQueue.main.async {
                guard let delegate = NSApp.delegate as? AppDelegate else { return }
                delegate.showCapturePicker()
            }
            return noErr
        }

        InstallEventHandler(GetApplicationEventTarget(), handler, 1, &eventType, nil, nil)

        // kVK_ANSI_N = 0x2D = 45, cmdKey = 256, optionKey = 2048
        let modifiers: UInt32 = UInt32(cmdKey | optionKey)
        RegisterEventHotKey(UInt32(kVK_ANSI_N), modifiers, hotKeyID, GetApplicationEventTarget(), 0, &hotKeyRef)
    }

    @objc func showCapturePicker() {
        if capturePanel != nil {
            capturePanel?.close()
            capturePanel = nil
        }

        let pickerView = CapturePickerView(
            onMixedMedia: { [weak self] in
                self?.capturePanel?.close()
                self?.capturePanel = nil
                self?.startMixedSession()
            },
            onTextNote: { [weak self] in
                self?.capturePanel?.close()
                self?.capturePanel = nil
                self?.showTextNote()
            },
            onScreenshot: { [weak self] in
                self?.capturePanel?.close()
                self?.capturePanel = nil
                self?.quickScreenshot()
            },
            onDismiss: { [weak self] in
                self?.capturePanel?.close()
                self?.capturePanel = nil
            }
        )

        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 340, height: 200),
            styleMask: [.titled, .closable, .nonactivatingPanel, .utilityWindow],
            backing: .buffered,
            defer: false
        )
        panel.title = "Ngram Capture"
        panel.level = .floating
        panel.isFloatingPanel = true
        panel.contentView = NSHostingView(rootView: pickerView)
        panel.center()
        panel.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)

        capturePanel = panel
    }

    func startMixedSession() {
        let sessionView = MixedSessionView(manager: sessionManager) { [weak self] in
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

    func showTextNote() {
        let textView = TextNoteView { [weak self] in
            self?.capturePanel?.close()
            self?.capturePanel = nil
        }

        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 500, height: 350),
            styleMask: [.titled, .closable, .nonactivatingPanel, .utilityWindow],
            backing: .buffered,
            defer: false
        )
        panel.title = "Ngram Note"
        panel.level = .floating
        panel.isFloatingPanel = true
        panel.contentView = NSHostingView(rootView: textView)
        panel.center()
        panel.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        capturePanel = panel
    }

    func quickScreenshot() {
        // Hide briefly, take screenshot, write to _inbox/.
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
            let ts = Int(Date().timeIntervalSince1970)
            let vaultPath = VaultConfig.vaultPath()
            let bundleDir = "\(vaultPath)/_inbox/\(ts)-screenshot"

            try? FileManager.default.createDirectory(atPath: bundleDir, withIntermediateDirectories: true)

            let imgPath = "\(bundleDir)/capture.png"
            let task = Process()
            task.launchPath = "/usr/sbin/screencapture"
            task.arguments = ["-i", imgPath]
            task.launch()
            task.waitUntilExit()

            if FileManager.default.fileExists(atPath: imgPath) {
                let manifest = """
                items:
                  - type: image
                    path: capture.png
                source: screenshot
                """
                try? manifest.write(toFile: "\(bundleDir)/manifest.yml", atomically: true, encoding: .utf8)
                sendNotification(title: "Ngram", body: "Screenshot captured")
            } else {
                // User cancelled.
                try? FileManager.default.removeItem(atPath: bundleDir)
            }
        }
    }

}
