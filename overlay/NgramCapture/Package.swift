// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "NgramCapture",
    platforms: [.macOS(.v13)],
    dependencies: [
        .package(url: "https://github.com/soffes/HotKey", from: "0.2.1"),
    ],
    targets: [
        .executableTarget(
            name: "NgramCapture",
            dependencies: ["HotKey"],
            path: "NgramCapture",
            exclude: ["Info.plist"]
        ),
    ]
)
