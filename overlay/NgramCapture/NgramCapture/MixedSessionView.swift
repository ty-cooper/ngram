import SwiftUI

struct CaptureSessionView: View {
    @ObservedObject var manager: CaptureSessionManager
    let onDismiss: () -> Void
    @State private var textInput = ""
    @FocusState private var textFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Ngram Capture")
                    .font(.headline)
                Spacer()
                Text("\(manager.items.count) items")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding()

            Divider()

            // Items list
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 8) {
                        ForEach(manager.items) { item in
                            HStack {
                                switch item.type {
                                case .screenshot:
                                    Image(systemName: "photo")
                                        .foregroundColor(.blue)
                                    Text(item.content)
                                        .font(.system(.body, design: .monospaced))
                                case .text:
                                    Image(systemName: "text.quote")
                                        .foregroundColor(.green)
                                    Text(item.content)
                                        .font(.body)
                                        .lineLimit(2)
                                }
                            }
                            .padding(.horizontal)
                            .padding(.vertical, 4)
                            .id(item.id)
                        }
                    }
                    .padding(.vertical, 8)
                }
                .onChange(of: manager.items.count) { _ in
                    if let last = manager.items.last {
                        proxy.scrollTo(last.id, anchor: .bottom)
                    }
                }
            }
            .frame(maxHeight: .infinity)

            Divider()

            // Text input
            HStack {
                TextField("Add text...", text: $textInput)
                    .textFieldStyle(.plain)
                    .focused($textFocused)
                    .onSubmit {
                        if !textInput.trimmingCharacters(in: .whitespaces).isEmpty {
                            manager.addText(textInput)
                            textInput = ""
                        }
                    }

                Button("Add") {
                    manager.addText(textInput)
                    textInput = ""
                    textFocused = true
                }
                .disabled(textInput.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding()

            Divider()

            // Actions
            HStack {
                Button("Screenshot") {
                    NSApp.hide(nil)
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                        manager.captureScreenshot {
                            NSApp.activate(ignoringOtherApps: true)
                        }
                    }
                }
                .keyboardShortcut("s", modifiers: .command)

                Spacer()

                Button("Cancel") {
                    manager.abort()
                    onDismiss()
                }
                .keyboardShortcut(.escape, modifiers: [])

                Button("Save") {
                    let pending = textInput.trimmingCharacters(in: .whitespacesAndNewlines)
                    if !pending.isEmpty {
                        manager.addText(pending)
                        textInput = ""
                    }
                    manager.finish()
                    // Hide the panel but keep the app alive in menu bar.
                    onDismiss()
                }
                .keyboardShortcut(.return, modifiers: .command)
                .buttonStyle(.borderedProminent)
                .disabled(manager.items.isEmpty && textInput.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding()
        }
        .frame(width: 480, height: 430)
        .onAppear { textFocused = true }
    }
}
