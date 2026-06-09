.PHONY: build test clean docker-up docker-down docker-logs

build:
	@mkdir -p bin
	@go build -o bin/node ./cmd/node
	@go build -o bin/client ./cmd/client

test:
	@go test ./...

clean:
	@rm -rf bin node1 node2 node3 node4 node5 downloaded-*.txt

docker-up:
	@docker compose up --build

docker-down:
	@docker compose down -v

docker-logs:
	@docker compose logs -f