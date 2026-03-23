import Foundation

struct VaultConfig {
    static func vaultPath() -> String {
        // Check env var first.
        if let envPath = ProcessInfo.processInfo.environment["NGRAM_VAULT_PATH"] {
            return (envPath as NSString).expandingTildeInPath
        }

        // Read ~/.ngram.yml
        let configPath = NSHomeDirectory() + "/.ngram.yml"
        guard let content = try? String(contentsOfFile: configPath, encoding: .utf8) else {
            return NSHomeDirectory() + "/.obsidian.ngram"
        }

        for line in content.components(separatedBy: "\n") {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("vault_path:") {
                var value = String(trimmed.dropFirst("vault_path:".count))
                    .trimmingCharacters(in: .whitespaces)
                    .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))
                value = (value as NSString).expandingTildeInPath
                return value
            }
        }

        return NSHomeDirectory() + "/.obsidian.ngram"
    }

    static func readBoxRC() -> (box: String, phase: String, ip: String) {
        let fm = FileManager.default
        var dir = fm.currentDirectoryPath

        while dir != "/" {
            let boxrcPath = dir + "/.boxrc"
            if let content = try? String(contentsOfFile: boxrcPath, encoding: .utf8) {
                var box = "", phase = "", ip = ""
                for line in content.components(separatedBy: "\n") {
                    let parts = line.components(separatedBy: "=")
                    guard parts.count == 2 else { continue }
                    let key = parts[0].trimmingCharacters(in: .whitespaces)
                    let val = parts[1].trimmingCharacters(in: .whitespaces)
                        .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))
                    switch key {
                    case "BOX": box = val
                    case "PHASE": phase = val
                    case "IP": ip = val
                    default: break
                    }
                }
                return (box, phase, ip)
            }
            dir = (dir as NSString).deletingLastPathComponent
        }
        return ("", "", "")
    }
}
