.PHONY: build

CMD_DIR=cmd/logistics/main.go
BIN_DIR_LINUX=bin/logistics/linux/logistics
BIN_DIR_WIN=bin/logistics/win/logistics.exe
BIN_DIR_DARWIN=bin/logistics/darwin/logistics
GO_BUILD=go build

build:
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR_LINUX)  -v $(CMD_DIR)
	GO111MODULE=on CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR_WIN)    -v $(CMD_DIR)
	GO111MODULE=on CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR_DARWIN) -v $(CMD_DIR)
