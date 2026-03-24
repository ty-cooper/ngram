import SwiftUI

@main
struct NgramCaptureApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    init() {
        // Stay alive as a menu bar agent — no dock icon, no quit on last window close.
        NSApp.setActivationPolicy(.accessory)
    }

    var body: some Scene {
        Settings {
            EmptyView()
        }
    }
}
