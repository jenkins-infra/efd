.PHONY: build

build:
	go build -o bin/efd

run:
	./bin/efd
	
lint:
	golangci-lint run
