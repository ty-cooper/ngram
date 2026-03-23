BINARY = n
BUILD_DIR = bin
PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: build install test lint release clean overlay obsidian

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/n

install:
	go install ./cmd/n

overlay:
	cd overlay/NgramCapture && swift build -c release
	@mkdir -p $(BUILD_DIR)/NgramCapture.app/Contents/MacOS
	cp overlay/NgramCapture/.build/release/NgramCapture $(BUILD_DIR)/NgramCapture.app/Contents/MacOS/NgramCapture
	cp overlay/NgramCapture/NgramCapture/Info.plist $(BUILD_DIR)/NgramCapture.app/Contents/Info.plist
	codesign --force --deep --sign - $(BUILD_DIR)/NgramCapture.app
	xattr -r -d com.apple.quarantine $(BUILD_DIR)/NgramCapture.app 2>/dev/null || true
	@echo "Built $(BUILD_DIR)/NgramCapture.app"
	@echo "NOTE: If this is a fresh build, re-grant Accessibility + Screen Recording in System Settings"

test:
	go test ./...

lint:
	go vet ./...

release:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build -o $(BUILD_DIR)/$(BINARY)-$${platform%/*}-$${platform#*/} ./cmd/n; \
		echo "built $(BUILD_DIR)/$(BINARY)-$${platform%/*}-$${platform#*/}"; \
	done

obsidian:
	cd obsidian-plugin && npm run build
	@VAULT=$$(grep vault_path ~/.ngram.yml | awk '{print $$2}' | sed "s|~|$$HOME|"); \
	PLUGIN_DIR="$$VAULT/.obsidian/plugins/ngram-search"; \
	mkdir -p "$$PLUGIN_DIR"; \
	cp obsidian-plugin/main.js "$$PLUGIN_DIR/main.js"; \
	cp obsidian-plugin/manifest.json "$$PLUGIN_DIR/manifest.json"; \
	echo "Installed to $$PLUGIN_DIR — reload Obsidian plugins"

clean:
	rm -rf $(BUILD_DIR)
