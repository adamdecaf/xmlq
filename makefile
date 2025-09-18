.PHONY: check
check:
ifeq ($(OS),Windows_NT)
	go test ./...
else
	@wget -O lint-project.sh https://raw.githubusercontent.com/moov-io/infra/master/go/lint-project.sh
	@chmod +x ./lint-project.sh
	COVER_THRESHOLD=75.0 ./lint-project.sh
endif

.PHONY: clean
clean:
	@rm -rf ./bin/ ./tmp/ coverage.txt misspell* staticcheck lint-project.sh


GOROOT_PATH=$(shell go env GOROOT)
WASM_124=$(GOROOT_PATH)/lib/wasm/wasm_exec.js
WASM_123=$(GOROOT_PATH)/misc/wasm/wasm_exec.js
TARGET_DIR=./docs/

.PHONY: wasm
wasm:
	@if [ -f "$(WASM_124)" ]; then \
		cp "$(WASM_124)" "$(TARGET_DIR)/wasm_exec.js"; \
	else \
		cp "$(WASM_123)" "$(TARGET_DIR)/wasm_exec.js"; \
	fi
	GOOS=js GOARCH=wasm go build -o docs/xmlq.wasm github.com/adamdecaf/xmlq/docs

.PHONY: cover-test cover-web
cover-test:
	go test -coverprofile=cover.out ./...
cover-web:
	go tool cover -html=cover.out
