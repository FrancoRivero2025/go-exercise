BINARY=ltp-service

.PHONY: build run docker docker-build test fmt

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
