GOPATH := $(shell go env GOPATH)
export GOPATH

TAG_DATE := $(shell date +%Y%m%d%H%M%S)
COMMIT_HASH := $(shell git rev-parse --short HEAD)
BINARY := sidecar

.DEFAULT_GOAL := build
build: deps
	go build -v -o $(BINARY)

clean:
	rm -f $(BINARY)

run: build
	@if [ ! -x ./$(BINARY) ]; then echo "Binary not found or not executable"; exit 1; fi
	./$(BINARY) watch -e 172.16.60.10:31478 -b test -d /Users/zhoulun/workspace/no8ge/sidecar/tmp

test:
	go test -coverprofile=coverage.out ./tests

coverage:
	@if [ ! -f coverage.out ]; then echo "Coverage profile file not found"; exit 1; fi
	go tool cover -html=coverage.out

deps:
	go mod tidy

docker:
	docker buildx build -f dockerfile --platform linux/amd64 -t no8ge/sidecar:$(TAG_DATE)-$(COMMIT_HASH) . --push

.PHONY: build clean run fmt test coverage deps docker chart