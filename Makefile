.PHONY: all lint format clean

all:
	gb build

lint:
	golint src/...

format:
	find src/ -name "*.go" | xargs gofmt -l -w -s

clean:
	rm -rf bin/ pkg/
