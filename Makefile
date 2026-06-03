.PHONY: build run test clean

build:
	@mkdir -p bin
	@go build -o bin/fs .

run: build
	@./bin/fs

test:
	@go test ./test

clean:
	@rm -rf bin