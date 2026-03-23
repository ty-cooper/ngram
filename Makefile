BINARY = n
BUILD_DIR = bin
PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: build install test lint release clean overlay

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/n

install:
	go install ./cmd/n

overlay:
	cd overlay/NgramCapture && swift build -c release
	cp overlay/NgramCapture/.build/release/NgramCapture $(BUILD_DIR)/NgramCapture
	@echo "Built $(BUILD_DIR)/NgramCapture — run it or add to Login Items"

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

clean:
	rm -rf $(BUILD_DIR)
