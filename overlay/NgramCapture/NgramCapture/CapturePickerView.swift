import SwiftUI

struct CapturePickerView: View {
    let onMixedMedia: () -> Void
    let onTextNote: () -> Void
    let onScreenshot: () -> Void
    let onDismiss: () -> Void

    var body: some View {
        VStack(spacing: 12) {
            Text("Capture")
                .font(.headline)
                .padding(.top, 8)

            Button(action: onMixedMedia) {
                HStack {
                    Image(systemName: "camera.on.rectangle")
                    VStack(alignment: .leading) {
                        Text("Mixed Media").font(.body.bold())
                        Text("Screenshots + text blocks").font(.caption).foregroundColor(.secondary)
                    }
                    Spacer()
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
            .buttonStyle(.plain)
            .background(Color.accentColor.opacity(0.1))
            .cornerRadius(8)

            Button(action: onTextNote) {
                HStack {
                    Image(systemName: "text.alignleft")
                    VStack(alignment: .leading) {
                        Text("Text Note").font(.body.bold())
                        Text("Quick brain dump").font(.caption).foregroundColor(.secondary)
                    }
                    Spacer()
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
            .buttonStyle(.plain)
            .background(Color.accentColor.opacity(0.1))
            .cornerRadius(8)

            Button(action: onScreenshot) {
                HStack {
                    Image(systemName: "camera.viewfinder")
                    VStack(alignment: .leading) {
                        Text("Screenshot").font(.body.bold())
                        Text("Region select, fire and forget").font(.caption).foregroundColor(.secondary)
                    }
                    Spacer()
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
            .buttonStyle(.plain)
            .background(Color.accentColor.opacity(0.1))
            .cornerRadius(8)

            Spacer()
        }
        .padding()
        .frame(width: 320, height: 200)
        .onExitCommand { onDismiss() }
    }
}
