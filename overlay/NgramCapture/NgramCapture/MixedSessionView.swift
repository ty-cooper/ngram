import SwiftUI

struct MixedSessionView: View {
    @ObservedObject var manager: CaptureSessionManager
    let onDismiss: () -> Void
    @State private var textInput = ""

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Text("Capture Session")
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
                    .onSubmit {
                        manager.addText(textInput)
                        textInput = ""
                    }

                Button("Add") {
                    manager.addText(textInput)
                    textInput = ""
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

                Button("Abort") {
                    manager.abort()
                    onDismiss()
                }
                .keyboardShortcut(.escape, modifiers: [])

                Button("Finish") {
                    manager.finish()
                    onDismiss()
                    showNotification(count: manager.items.count)
                }
                .keyboardShortcut(.return, modifiers: .command)
                .buttonStyle(.borderedProminent)
            }
            .padding()
        }
        .frame(width: 480, height: 430)
    }

    private func showNotification(count: Int) {
        let notification = NSUserNotification()
        notification.title = "Ngram"
        notification.informativeText = "\(count) items captured"
        NSUserNotificationCenter.default.deliver(notification)
    }
}
