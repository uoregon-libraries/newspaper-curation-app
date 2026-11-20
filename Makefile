SOURCES := $(shell find ./src ./internal -name "*.go")
SOURCEDIRS := $(shell find ./src ./internal -type d)
BUILD := $(shell git describe --tags 2>/dev/null)

.PHONY: all
all: deps validate build

.PHONY: docs
docs:
	rm -rf ./docs
	cd hugo && hugo
	mv hugo/public ./docs

.PHONY: deps
deps:
	go mod download

.PHONY: audit
audit:
	go tool govulncheck ./src/... ./internal/... ./test/harness/...

.PHONY: fast
fast: deps build

.PHONY: validate
validate:
	./scripts/validate.sh

.PHONY: build
build: $(shell ./scripts/cmdslist.sh)

# For quick building of binaries, you can run something like "make bin/server"
bin/%: src/cmd/% $(SOURCES) $(SOURCEDIRS)
	go build -ldflags="-s -w -X github.com/uoregon-libraries/newspaper-curation-app/src/version.Version=$(BUILD)" -o $@ github.com/uoregon-libraries/newspaper-curation-app/$<

# Dev and test stack tools, run on the host: see the Developer's Guide. They're
# their own module, with no dependencies, so they build fast without NCA's.
TOOL_SOURCES := $(shell find ./tools/ncadev -name "*.go") tools/ncadev/go.mod

.PHONY: tools
tools: bin/ncadev bin/ncatest

bin/ncadev: $(TOOL_SOURCES)
	go build -C tools/ncadev -o $(abspath $@) .

bin/ncatest: $(TOOL_SOURCES)
	go build -C tools/ncadev -o $(abspath $@) ./cmd/ncatest

.PHONY: lint
lint:
	go tool revive --config=./revive.toml --formatter=unix src/... test/harness/... tools/ncadev/...

.PHONY: format
format:
	find src/ internal/ test/harness/ tools/ -name "*.go" | xargs go tool goimports -l -w

.PHONY: test
test:
	go test ./src/... ./internal/...
	go -C tools/ncadev test ./...

.PHONY: clean
clean:
	rm bin/* -f

.PHONY: distclean
distclean: clean
	go clean -modcache -testcache -cache
