import Foundation

struct CaptureItem: Identifiable {
    let id = UUID()
    let type: ItemType
    let timestamp: Date
    var content: String // filename for screenshot, text for text blocks

    enum ItemType {
        case screenshot
        case text
    }
}

class CaptureSessionManager: ObservableObject {
    @Published var items: [CaptureItem] = []
    @Published var isActive = false

    private var screenshotCount = 0
    private var screenshotFiles: [String] = [] // full paths to temp screenshots

    func startSession() {
        items = []
        screenshotCount = 0
        screenshotFiles = []
        isActive = true
    }

    func captureScreenshot(completion: @escaping () -> Void) {
        screenshotCount += 1
        let filename = String(format: "ss-%03d.png", screenshotCount)
        // Save to a temp location first.
        let tempPath = NSTemporaryDirectory() + filename

        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            let task = Process()
            task.executableURL = URL(fileURLWithPath: "/usr/sbin/screencapture")
            task.arguments = ["-i", "-x", tempPath]
            do {
                try task.run()
                task.waitUntilExit()
            } catch {
                DispatchQueue.main.async { completion() }
                return
            }

            DispatchQueue.main.async {
                if task.terminationStatus == 0 && FileManager.default.fileExists(atPath: tempPath) {
                    let attrs = try? FileManager.default.attributesOfItem(atPath: tempPath)
                    let size = attrs?[.size] as? Int ?? 0
                    if size > 0 {
                        self?.items.append(CaptureItem(type: .screenshot, timestamp: Date(), content: filename))
                        self?.screenshotFiles.append(tempPath)
                    } else {
                        try? FileManager.default.removeItem(atPath: tempPath)
                    }
                }
                completion()
            }
        }
    }

    func addText(_ text: String) {
        guard !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        items.append(CaptureItem(type: .text, timestamp: Date(), content: text))
    }

    func finish() {
        guard isActive, !items.isEmpty else {
            isActive = false
            return
        }

        let vaultPath = VaultConfig.vaultPath()
        let assetsDir = "\(vaultPath)/_assets"
        try? FileManager.default.createDirectory(atPath: assetsDir, withIntermediateDirectories: true)

        let ts = Int(Date().timeIntervalSince1970)
        let boxrc = VaultConfig.readBoxRC()

        // Move screenshots to _assets/ with timestamped names.
        for tempPath in screenshotFiles {
            let filename = (tempPath as NSString).lastPathComponent
            let destPath = "\(assetsDir)/\(ts)-\(filename)"
            try? FileManager.default.moveItem(atPath: tempPath, toPath: destPath)
        }

        // Build a single .md file with text and embedded screenshots.
        var body = ""
        for item in items {
            switch item.type {
            case .text:
                body += item.content + "\n\n"
            case .screenshot:
                body += "![[\(ts)-\(item.content)]]\n\n"
            }
        }

        // Frontmatter.
        var note = "---\n"
        note += "captured: \"\(ISO8601DateFormatter().string(from: Date()))\"\n"
        note += "source: \"capture-overlay\"\n"
        if !boxrc.box.isEmpty { note += "box: \"\(boxrc.box)\"\n" }
        if !boxrc.phase.isEmpty { note += "phase: \"\(boxrc.phase)\"\n" }
        note += "---\n\n"
        note += body.trimmingCharacters(in: .whitespacesAndNewlines) + "\n"

        // Write to _inbox/ as a single .md file.
        let inboxPath = "\(vaultPath)/_inbox/\(ts)-capture.md"
        try? note.write(toFile: inboxPath, atomically: true, encoding: .utf8)

        isActive = false
    }

    func abort() {
        // Clean up temp screenshots.
        for tempPath in screenshotFiles {
            try? FileManager.default.removeItem(atPath: tempPath)
        }
        items = []
        screenshotFiles = []
        isActive = false
    }
}
