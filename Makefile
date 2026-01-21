BIN_NAME=restorable
VERSION  ?= $(shell git describe --tags --always)
DIST_DIR := bin

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64


build:
	go build -o $(DIST_DIR)/$(BIN_NAME) ./cmd/restorable

build-all:
	mkdir -p $(DIST_DIR)
	@for p in $(PLATFORMS); do \
		OS=$${p%/*}; ARCH=$${p#*/}; \
		OUT=$(DIST_DIR)/$(BIN_NAME)-$$OS-$$ARCH; \
		echo "Building $$OUT"; \
		GOOS=$$OS GOARCH=$$ARCH CGO_ENABLED=0 \
			go build -ldflags "-s -w" -o $$OUT ./cmd/$(BIN_NAME); \
	done

checksum:
	@for f in $(DIST_DIR)/$(BIN_NAME)-*; do \
		shasum -a 256 $$f | awk '{print $$1}' > $$f.sha256; \
	done

run:
	go run ./cmd/restorable

lint:
	golangci-lint run

clean:
	rm -rf $(DIST_DIR)

release: clean build-all checksum
	@echo "Release artifacts are in $(DIST_DIR)/"
