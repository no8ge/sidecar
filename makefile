GOPATH := $(shell go env GOPATH)
export GOPATH

TAG_DATE := $(shell date +%Y%m%d%H%M%S)
COMMIT_HASH := $(shell git rev-parse --short HEAD)

.DEFAULT_GOAL := build
build: deps
	go build -v -o $(CHART_NAME)

clean:
	rm -f $(CHART_NAME)

run: build
	@if [ ! -x ./$(CHART_NAME) ]; then echo "Binary not found or not executable"; exit 1; fi
	./$(CHART_NAME)

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