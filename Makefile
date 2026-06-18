BINARY_NAME = census
BIN_DIR     = bin
SOURCES     = main.go

.PHONY: all clean platform

all: linux-amd64 linux-arm64 darwin-arm64 darwin-amd64

linux-amd64:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(SOURCES)

linux-arm64:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(SOURCES)

darwin-arm64:
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(SOURCES)

darwin-amd64:
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(SOURCES)

platform:
	@mkdir -p $(BIN_DIR)
	GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build -o $(BIN_DIR)/$(BINARY_NAME) $(SOURCES)

clean:
	rm -f $(BIN_DIR)/$(BINARY_NAME)-*
