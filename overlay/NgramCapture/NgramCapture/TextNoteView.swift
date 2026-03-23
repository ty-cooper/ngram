import SwiftUI

struct TextNoteView: View {
    let onDismiss: () -> Void
    @State private var text = ""

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text("Note")
                    .font(.headline)
                Spacer()
            }
            .padding()

            Divider()

            TextEditor(text: $text)
                .font(.system(.body, design: .monospaced))
                .padding(8)
                .frame(maxHeight: .infinity)

            Divider()

            HStack {
                Button("Cancel") {
                    onDismiss()
                }
                .keyboardShortcut(.escape, modifiers: [])

                Spacer()

                Button("Save") {
                    saveNote()
                    onDismiss()
                }
                .keyboardShortcut(.return, modifiers: .command)
                .buttonStyle(.borderedProminent)
                .disabled(text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
            .padding()
        }
        .frame(width: 480, height: 340)
    }

    private func saveNote() {
        let body = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !body.isEmpty else { return }

        let vaultPath = VaultConfig.vaultPath()
        let ts = Int(Date().timeIntervalSince1970)
        let slug = String(body.prefix(50))
            .lowercased()
            .replacingOccurrences(of: "[^a-z0-9]+", with: "-", options: .regularExpression)
            .trimmingCharacters(in: CharacterSet(charactersIn: "-"))

        let filename = "\(ts)-\(slug).md"
        let inboxDir = "\(vaultPath)/_inbox"
        try? FileManager.default.createDirectory(atPath: inboxDir, withIntermediateDirectories: true)

        let iso = ISO8601DateFormatter().string(from: Date())
        let boxrc = VaultConfig.readBoxRC()

        var frontmatter = "---\n"
        frontmatter += "captured: \"\(iso)\"\n"
        frontmatter += "source: \"overlay\"\n"
        frontmatter += "capture_mode: \"text\"\n"
        if !boxrc.box.isEmpty { frontmatter += "box: \"\(boxrc.box)\"\n" }
        if !boxrc.phase.isEmpty { frontmatter += "phase: \"\(boxrc.phase)\"\n" }
        frontmatter += "---\n\n"

        let content = frontmatter + body + "\n"
        try? content.write(toFile: "\(inboxDir)/\(filename)", atomically: true, encoding: .utf8)

        sendNotification(title: "Ngram", body: "Note captured")
    }
}
