.PHONY: build test lint cover clean live-test

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS  = -s -w -X github.com/VecTrade-io/vectrade-cli/cmd.version=$(VERSION) -X github.com/VecTrade-io/vectrade-cli/cmd.commit=$(COMMIT) -X github.com/VecTrade-io/vectrade-cli/cmd.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/vectrade .

test:
	go test -race ./...

lint:
	golangci-lint run ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	@coverage=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
	threshold=90; \
	if [ $$(echo "$$coverage < $$threshold" | bc -l) -eq 1 ]; then \
		echo "FAIL: coverage $$coverage% < $$threshold%"; exit 1; \
	else \
		echo "OK: coverage $$coverage% >= $$threshold%"; \
	fi

clean:
	rm -rf bin/ coverage.out dist/

live-test:
	@if [ -z "$$VECTRADE_TEST_API_KEY" ]; then echo "VECTRADE_TEST_API_KEY not set"; exit 1; fi
	go test -v -count=1 -timeout=120s ./tests/
