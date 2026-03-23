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

    private var sessionDir: String = ""
    private var screenshotCount = 0

    func startSession() {
        let ts = Int(Date().timeIntervalSince1970)
        let vaultPath = VaultConfig.vaultPath()
        sessionDir = "\(vaultPath)/_inbox/\(ts)-capture-session"
        try? FileManager.default.createDirectory(atPath: sessionDir, withIntermediateDirectories: true)
        items = []
        screenshotCount = 0
        isActive = true
    }

    func captureScreenshot(completion: @escaping () -> Void) {
        screenshotCount += 1
        let filename = String(format: "ss-%03d.png", screenshotCount)
        let filepath = "\(sessionDir)/\(filename)"

        let task = Process()
        task.launchPath = "/usr/sbin/screencapture"
        task.arguments = ["-i", filepath]
        task.terminationHandler = { [weak self] process in
            DispatchQueue.main.async {
                if process.terminationStatus == 0 && FileManager.default.fileExists(atPath: filepath) {
                    self?.items.append(CaptureItem(type: .screenshot, timestamp: Date(), content: filename))
                }
                completion()
            }
        }
        task.launch()
    }

    func addText(_ text: String) {
        guard !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        items.append(CaptureItem(type: .text, timestamp: Date(), content: text))
    }

    func finish() {
        guard isActive else { return }

        let boxrc = VaultConfig.readBoxRC()
        var manifest = "session_id: \"\(ISO8601DateFormatter().string(from: Date()))\"\n"
        manifest += "capture_mode: \"mixed\"\n"
        if !boxrc.box.isEmpty { manifest += "box: \"\(boxrc.box)\"\n" }
        if !boxrc.phase.isEmpty { manifest += "phase: \"\(boxrc.phase)\"\n" }
        manifest += "items:\n"

        for item in items {
            let ts = ISO8601DateFormatter().string(from: item.timestamp)
            switch item.type {
            case .screenshot:
                manifest += "  - type: screenshot\n"
                manifest += "    file: \(item.content)\n"
                manifest += "    timestamp: \"\(ts)\"\n"
            case .text:
                let escaped = item.content.replacingOccurrences(of: "\"", with: "\\\"")
                manifest += "  - type: text\n"
                manifest += "    content: \"\(escaped)\"\n"
                manifest += "    timestamp: \"\(ts)\"\n"
            }
        }

        try? manifest.write(toFile: "\(sessionDir)/manifest.yml", atomically: true, encoding: .utf8)
        isActive = false
    }

    func abort() {
        if !sessionDir.isEmpty {
            try? FileManager.default.removeItem(atPath: sessionDir)
        }
        items = []
        isActive = false
    }
}
