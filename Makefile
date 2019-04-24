.PHONY: all deps fast validate build lint format test clean distclean

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

test:
	go test ./src/...

clean:
	rm bin/* -f

distclean: clean
	go clean -modcache -testcache -cache
