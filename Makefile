.PHONY: all lint format clean

all: vendor/src
	@./validate.sh && gb build

vendor/src:
	gb vendor restore

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs gofmt -l -w -s

clean:
	rm -rf bin/ pkg/
