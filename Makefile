.PHONY: all build run test clean tidy lint help

APP_NAME ?= api
BUILD_DIR ?= bin
MAIN_PATH ?= ./cmd/api

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

test:
	go test -v -race ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)

help:
	@echo "Available commands:"
	@echo "  make build - Compile binary to $(BUILD_DIR)/$(APP_NAME)"
	@echo "  make run   - Run application"
	@echo "  make test  - Run tests"
	@echo "  make tidy  - Tidy go modules"
	@echo "  make clean - Remove build artifacts"

