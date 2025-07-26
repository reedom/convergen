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

# =============================================================================
# Linting and Code Quality
# =============================================================================

.PHONY: lint
lint: ## Run comprehensive linting with golangci-lint
lint: fmt-check
	@echo "Running golangci-lint..."
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run linter and automatically fix issues where possible
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix ./...

.PHONY: lint-docker
lint-docker: ## Run linter using Docker (fallback option)
lint-docker:
	docker run --rm --platform=linux/amd64 \
		-v "${PWD}:/src" -w /src \
		golangci/golangci-lint:latest golangci-lint --go=1.23 run

.PHONY: lint-security
lint-security: ## Run security-focused linting
lint-security:
	@echo "Running security linting..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt sarif -out gosec-report.sarif -stdout -verbose=text ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

.PHONY: lint-complexity
lint-complexity: ## Check code complexity and maintainability
lint-complexity:
	@echo "Checking cyclomatic complexity..."
	@if command -v gocyclo >/dev/null 2>&1; then \
		gocyclo -over 15 .; \
	else \
		echo "gocyclo not found. Install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"; \
	fi
	@echo "Checking cognitive complexity..."
	@if command -v cognitive >/dev/null 2>&1; then \
		cognitive -over 20 .; \
	else \
		echo "cognitive not found. Install with: go install github.com/uudashr/gocognit/cmd/gocognit@latest"; \
	fi

.PHONY: lint-deps
lint-deps: ## Check for dependency issues
lint-deps:
	@echo "Checking for unused dependencies..."
	go mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "go.mod or go.sum has changes after 'go mod tidy'"; \
		git diff go.mod go.sum; \
		exit 1; \
	fi
	@echo "Checking for vulnerable dependencies..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

.PHONY: fmt
fmt: ## Format all Go code
fmt:
	@echo "Formatting code..."
	@if command -v goimports >/dev/null 2>&1; then \
		find . -name "*.go" -not -path "./.reference-projects/*" -not -path "./vendor/*" | xargs goimports -w -local github.com/reedom/convergen; \
	else \
		echo "goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
		echo "Falling back to gofmt..."; \
		find . -name "*.go" -not -path "./.reference-projects/*" -not -path "./vendor/*" | xargs gofmt -w -s; \
	fi

.PHONY: fmt-check
fmt-check: ## Check if code is properly formatted
fmt-check:
	@echo "Checking code formatting..."
	@if command -v goimports >/dev/null 2>&1; then \
		formatting_issues=$$(find . -name "*.go" -not -path "./.reference-projects/*" -not -path "./vendor/*" | xargs goimports -l -local github.com/reedom/convergen); \
		if [ -n "$$formatting_issues" ]; then \
			echo "The following files have formatting issues:"; \
			echo "$$formatting_issues"; \
			exit 1; \
		fi; \
	else \
		echo "goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
		echo "Falling back to gofmt..."; \
		unformatted=$$(find . -name "*.go" -not -path "./.reference-projects/*" -not -path "./vendor/*" | xargs gofmt -l); \
		if [ -n "$$unformatted" ]; then \
			echo "The following files are not properly formatted:"; \
			echo "$$unformatted"; \
			exit 1; \
		fi; \
	fi

.PHONY: lint-all
lint-all: ## Run all linting checks (comprehensive)
lint-all: fmt-check lint lint-security lint-complexity lint-deps
	@echo "All linting checks completed successfully!"

.PHONY: install-linters
install-linters: ## Install all recommended linting tools
install-linters:
	@echo "Installing linting tools..."
	@echo "Installing golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Installing goimports..."
	@if ! command -v goimports >/dev/null 2>&1; then \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@echo "Installing gosec..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@echo "Installing govulncheck..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@echo "Installing gocyclo..."
	@if ! command -v gocyclo >/dev/null 2>&1; then \
		go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	@echo "Installing gocognit..."
	@if ! command -v gocognit >/dev/null 2>&1; then \
		go install github.com/uudashr/gocognit/cmd/gocognit@latest; \
	fi
	@echo "All linting tools installed successfully!"

.PHONY: test
test: ## Run all tests
test:
	go test github.com/reedom/convergen/v8/tests && \
	go test github.com/reedom/convergen/v8/pkg/...

.PHONY: coverage
coverage:
	@go test -v -cover ./... -coverprofile coverage.out -coverpkg ./... 2>&1 >/dev/null && \
	go tool cover -func coverage.out -o coverage.out 2>&1 >/dev/null && \
	cat coverage.out
