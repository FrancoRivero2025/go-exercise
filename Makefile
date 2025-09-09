BINARY=ltp-service

.PHONY: build run docker docker-build test fmt all-tests lint unit-tests integration-tests

build:
	go build -o bin/$(BINARY) ./cmd/ltp-service

run: build
	./bin/$(BINARY)

docker-build:
	docker build -t ltp-service:local .

docker-run: docker-build
	docker run --rm -p 8080:8080 ltp-service:local

test:
	go test ./...

fmt:
	gofmt -w .

lint:
	docker compose run --rm lint

unit-tests:
	docker compose run --rm unit-tests

integration-tests:
	docker compose run --rm integration-tests

all-tests: lint unit-tests integration-tests
