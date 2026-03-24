import SwiftUI

@main
struct NgramCaptureApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    init() {
        // NSApp activation policy is set in AppDelegate.applicationDidFinishLaunching
    }

    var body: some Scene {
        // MenuBarExtra keeps the app alive as a persistent menu bar agent.
        // The actual menu bar icon and hotkey are managed by AppDelegate.
        MenuBarExtra {
            Button("Capture (⌘⌥N)") {
                appDelegate.showCapturePicker()
            }
            Divider()
            Button("Quit") {
                NSApplication.shared.terminate(nil)
            }
        } label: {
            Image(systemName: "brain.head.profile")
        }
    }
}
