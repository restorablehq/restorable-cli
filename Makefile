BINARY=restorable

build:
	go build -o bin/$(BINARY) ./cmd/restorable

run:
	go run ./cmd/restorable

lint:
	golangci-lint run

clean:
	rm -f bin/$(BINARY)

