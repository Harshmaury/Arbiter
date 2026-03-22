.PHONY: build test verify clean

build:
	go build -o arbiter ./cmd/arbiter/

test:
	go test ./...

verify:
	go vet ./...
	go build ./...

clean:
	rm -f arbiter
