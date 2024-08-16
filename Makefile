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

.PHONY: lint
lint: ## Run linter
lint:
	docker run --rm --platform=linux/amd64 \
		-v "${PWD}:/src" -w /src \
		--rm \
		golangci/golangci-lint:latest golangci-lint --go=1.19 run

.PHONY: test
test: ## Run all tests
test:
	go test github.com/reedom/convergen/tests && \
	go test github.com/reedom/convergen/pkg/...

.PHONY: coverage
coverage:
	@go test -v -cover ./... -coverprofile coverage.out -coverpkg ./... 2>&1 >/dev/null && \
	go tool cover -func coverage.out -o coverage.out 2>&1 >/dev/null && \
	cat coverage.out

.PHONY: build-linux
build-linux:
	mkdir -p build/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/convergen .

.PHONY: gen
gen:
	./build/convergen -suffix transfer tests/fixtures/usecase/lixinio/data/copygen.go
	./build/convergen -suffix transfer tests/fixtures/usecase/lixinio/data/copygen_arg.go