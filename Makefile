# ---------------------------------------------------------------------------
# Variable block
# ---------------------------------------------------------------------------

# CUR_DIR captures the working directory where make is invoked.
CUR_DIR=$(shell pwd)

# CMD_DIR is the path to the selected command's main package (e.g. cmd/logistics).
CMD_DIR=cmd/$(cmd)

# BIN_DIR_* are the per-OS output directories under bin/<cmd>/.
BIN_DIR_LINUX=bin/$(cmd)/linux
BIN_DIR_WIN=bin/$(cmd)/win
BIN_DIR_DARWIN=bin/$(cmd)/darwin

# Standard Go tool aliases.
GO_BUILD=go build
GO_MOD_TIDY=go mod tidy

# Extract the latest Git tag and strip the leading "v" to produce a semver
# string used as the container image tag (e.g. v1.2.3 -> 1.2.3).
GIT_TAG=$(shell git describe --abbrev=0 --tags)
VERSION=$(GIT_TAG:v%=%)

# ---------------------------------------------------------------------------
# Guard: require the `cmd` variable for every invocation.
# Example: make build cmd=logistics
# ---------------------------------------------------------------------------
ifeq ($(cmd),)
$(error cmd is not set. Usage: make docker cmd=insights)
endif

.PHONY: build docker clean

# ---------------------------------------------------------------------------
# build: cross-compile the selected command for linux/amd64, darwin/amd64,
#        and windows/amd64. Binaries are written to bin/<cmd>/<os>/.
# ---------------------------------------------------------------------------
build:
	GO111MODULE=on $(GO_MOD_TIDY)
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR_LINUX)/$(cmd) -v $(CMD_DIR)/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR_DARWIN)/$(cmd) -v $(CMD_DIR)/main.go
	GO111MODULE=on CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR_WIN)/$(cmd).exe -v $(CMD_DIR)/main.go

# ---------------------------------------------------------------------------
# docker: build a linux/amd64 container image for the selected command using
#         build/Dockerfile. The image is tagged with the current Git version.
# ---------------------------------------------------------------------------
docker:
	docker build --platform linux/amd64 -f build/Dockerfile --build-arg "CMD=$(cmd)" -t cr.yandex/crpoinjsjge915cq8ufl/go-x5-$(cmd):$(VERSION) .

# ---------------------------------------------------------------------------
# clean: remove all compiled binaries under the bin/ directory.
# ---------------------------------------------------------------------------
clean:
	rm -r bin/
