# Makefile for hexarena
#
# Thin wrappers over the go commands. The one target worth having is `golden`:
# -update is declared only in the packages that own golden files, so
# `go test ./... -update` fails on every other package.

.PHONY: build install run auto test golden fmt vet check clean

build:
	@go build -o bin/ ./cmd/hexarena

install:
	@go install ./cmd/hexarena

run:
	@go run ./cmd/hexarena --seed 11 --side ally

auto:
	@go run ./cmd/hexarena --auto --seed 11

test:
	@go test ./...

# Accept new golden files, then read the diff — the goldens are the design
# record. Add a package here when it gains one.
golden:
	@go test ./internal/core/hex ./internal/seed ./internal/tui -update

fmt:
	@gofmt -w .

vet:
	@go vet ./...

# The gate: what has to be clean before a change is done.
check:
	@gofmt -l .
	@go vet ./...
	@go test ./... -count=1

clean:
	@rm -rf bin/
