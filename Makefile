.PHONY: help
help: ## Show this help message.
	@echo 'usage: make [target] ...'
	@echo
	@echo 'targets:'
	@egrep '^(.+)\:\ ##\ (.+)' ${MAKEFILE_LIST} | column -t -c 2 -s ':#'

SRC   := $(shell find . -name '*.go')
TARGET = build/convergen

.PHONY: build
build: ## Build cli command
build: $(TARGET)

$(TARGET): $(SRC)
	@mkdir -p build
	go build -o build/convergen main.go
