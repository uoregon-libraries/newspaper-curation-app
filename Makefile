.PHONY: all docs deps fast validate build lint format test clean distclean

SOURCES := $(shell find ./src -name "*.go")
SOURCEDIRS := $(shell find ./src -type d)
BUILD := $(shell git describe --tags)

all: deps validate build

docs:
	rm -rf ./docs
	cd hugo && hugo
	mv hugo/public ./docs

docserver:
	rm -rf ./localdocs
	cp -r docs/ localdocs/
	find localdocs -type f | xargs sed -i 's|https://uoregon-libraries.github.io/newspaper-curation-app|http://localhost:8080|g'
	docker run --rm -it --name nca_docs -p 8080:80 -v "./localdocs":/usr/local/apache2/htdocs/ httpd:2.4
	rm -rf ./localdocs

deps:
	go mod download

fast: deps build

validate:
	./scripts/validate.sh

build: $(shell ./scripts/cmdslist.sh)

# For quick building of binaries, you can run something like "make bin/server"
bin/%: src/cmd/% $(SOURCES) $(SOURCEDIRS)
	go build -ldflags="-s -w -X github.com/uoregon-libraries/newspaper-curation-app/src/version.Version=$(BUILD)" -o $@ github.com/uoregon-libraries/newspaper-curation-app/$<

lint:
	revive --config=./revive.toml --formatter=unix src/...

format:
	find src/ -name "*.go" | xargs goimports -l -w

test:
	go test ./src/...

clean:
	rm bin/* -f

distclean: clean
	go clean -modcache -testcache -cache
