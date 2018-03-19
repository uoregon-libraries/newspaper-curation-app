.PHONY: all fast validate build lint format clean

all: vendor/src validate build

fast: vendor/src build

validate:
	./validate.sh

build:
	gb build

vendor/src:
	gb vendor restore

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs gofmt -l -w -s

clean:
	rm -rf bin/* pkg/*

cleanall: clean
	rm -rf vendor/src
	rm -rf ${HOME}/.gb/cache
