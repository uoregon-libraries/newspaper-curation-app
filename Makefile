.PHONY: all deps fast validate build lint format test clean distclean

SOURCES := $(shell find ./src -name "*.go")
SOURCEDIRS := $(shell find ./src -type d)

all: deps validate build

deps:
	go mod download

fast: deps build

validate:
	./scripts/validate.sh

build: $(shell ./scripts/cmdslist.sh)

# For quick building of binaries, you can run something like "make bin/server"
# and still have a little bit of the vetting without running the entire
# validation script
bin/%: src/cmd/% $(SOURCES) $(SOURCEDIRS)
	golint -set_exit_status $</...
	go vet ./$<
	go build -ldflags="-s -w" -o $@ github.com/uoregon-libraries/newspaper-curation-app/$<

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs gofmt -l -w -s

test:
	go test ./src/...

clean:
	rm bin/* -f

distclean: clean
	go clean -modcache -testcache -cache
