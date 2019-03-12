.PHONY: all deps fast validate build lint format clean distclean

all: deps validate build

deps:
	go mod download

fast: deps build

validate:
	./validate.sh

build:
	./makeall.sh

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs gofmt -l -w -s

clean:
	rm bin/* -f

distclean: clean
	go clean -modcache -testcache -cache
