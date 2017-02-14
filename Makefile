.PHONY: all lint

all:
	gb build

lint:
	golint src/...

clean:
	rm -f bin/ pkg/
