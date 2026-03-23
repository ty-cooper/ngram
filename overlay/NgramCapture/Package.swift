// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "NgramCapture",
    platforms: [.macOS(.v13)],
    targets: [
        .executableTarget(
            name: "NgramCapture",
            path: "NgramCapture"
        ),
    ]
)
