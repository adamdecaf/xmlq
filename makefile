ifndef VERSION
	ifndef RELEASE
	# If we're not publishing a release, set the dev commit hash
		ifndef DEV_TAG_SHA
			COMMIT_HASH :=$(shell git rev-parse --short=7 HEAD)
		else
			COMMIT_HASH :=$(shell echo ${DEV_TAG_SHA} | cut -c 1-7)
		endif
		VERSION := dev-${COMMIT_HASH}
	else
		VERSION := $(shell git describe --tags --abbrev=0)
	endif
endif

.PHONY: build
build:
	go build -o ./bin/xmlq -ldflags "-X github.com/adamdecaf/xmlq.Version=${VERSION}" github.com/adamdecaf/xmlq/cmd/xmlq

.PHONY: check
check:
ifeq ($(OS),Windows_NT)
	go test ./...
else
	@wget -O lint-project.sh https://raw.githubusercontent.com/moov-io/infra/master/go/lint-project.sh
	@chmod +x ./lint-project.sh
	COVER_THRESHOLD=50.0 ./lint-project.sh
endif

.PHONY: clean
clean:
	@rm -rf ./bin/ ./tmp/ coverage.txt misspell* staticcheck lint-project.sh


GOROOT_PATH=$(shell go env GOROOT)
WASM_124=$(GOROOT_PATH)/lib/wasm/wasm_exec.js
WASM_123=$(GOROOT_PATH)/misc/wasm/wasm_exec.js
TARGET_DIR=./docs/

.PHONY: wasm dist-webui
wasm:
	@if [ -f "$(WASM_124)" ]; then \
		cp "$(WASM_124)" "$(TARGET_DIR)/wasm_exec.js"; \
	else \
		cp "$(WASM_123)" "$(TARGET_DIR)/wasm_exec.js"; \
	fi
	GOOS=js GOARCH=wasm go build -o docs/xmlq.wasm ./docs/main.go

dist-webui: wasm
	git config user.name "adamdecaf-bot"
	git config user.email "bot@ashannon.us"
	git add ./docs/wasm_exec.js ./docs/xmlq.wasm
	git commit -m "chore: updating wasm webui [skip ci]" || echo "No changes to commit"
	git push origin master

.PHONY: cover-test cover-web
cover-test:
	go test -coverprofile=cover.out ./...
cover-web:
	go tool cover -html=cover.out
