# Makefile for hexarena
#
# Thin wrappers over the go commands. The one target worth having is `golden`:
# -update is declared only in the packages that own golden files, so
# `go test ./... -update` fails on every other package.

.PHONY: build install run auto play-tui forge forge-tui forge-tui-en test golden fmt vet check clean

build:
	@go build -o bin/ ./cmd/hexarena ./cmd/hexarena-tui ./cmd/hexforge ./cmd/hexforge-tui

install:
	@go install ./cmd/hexarena ./cmd/hexarena-tui ./cmd/hexforge ./cmd/hexforge-tui

run:
	@go run ./cmd/hexarena --seed 11 --side ally

auto:
	@go run ./cmd/hexarena --auto --seed 11

# The full-screen game client, over the same internal/screen the authoring tool
# draws. It takes over the screen, so it refuses to start when stdout is not a
# terminal — use `run`, `auto` or `--replay` from a script or a pipe.
#
# It opens in Vietnamese, like the authoring client: `make play-tui ARGS="--lang
# en"` starts it in English, HEXARENA_LANG=en does the same for a whole shell,
# and ctrl+l swaps the two from any screen.
play-tui:
	@go run ./cmd/hexarena-tui $(ARGS)

# The authoring tool. It reads and writes internal/seed/data, so run it from the
# module root; `make forge` with no arguments prints its subcommands.
forge:
	@go run ./cmd/hexforge $(ARGS)

# The full-screen authoring client, over the same internal/forge. It takes over
# the screen, so it refuses to start when stdout is not a terminal — use `forge`
# from a script or a pipe.
#
# It opens in Vietnamese. `make forge-tui ARGS="--lang en"` starts it in
# English, HEXARENA_LANG=en does the same for a whole shell, and ctrl+l swaps
# the two from any screen without losing what has been typed.
forge-tui:
	@go run ./cmd/hexforge-tui $(ARGS)

# The same in English, which is what `make forge` and `hexforge check` speak.
forge-tui-en:
	@go run ./cmd/hexforge-tui --lang en $(ARGS)

test:
	@go test ./...

# Accept new golden files, then read the diff — the goldens are the design
# record. Add a package here when it gains one.
golden:
	@go test ./cmd/hexarena-tui ./cmd/hexforge-tui ./internal/core/hex ./internal/i18n ./internal/screen ./internal/seed ./internal/tui -update

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
