SOURCES := $(shell find ./src ./internal -name "*.go")
SOURCEDIRS := $(shell find ./src ./internal -type d)
BUILD := $(shell git describe --tags)

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
	go tool govulncheck ./src/... ./internal/...

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

.PHONY: lint
lint:
	go tool revive --config=./revive.toml --formatter=unix src/...

.PHONY: format
format:
	find src/ internal/ -name "*.go" | xargs go tool goimports -l -w

.PHONY: test
test:
	go test ./src/... ./internal/...

.PHONY: clean
clean:
	rm bin/* -f

.PHONY: distclean
distclean: clean
	go clean -modcache -testcache -cache
