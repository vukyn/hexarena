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
	@go test ./cmd/hexarena-tui ./cmd/hexforge-tui ./internal/core/hex ./internal/i18n ./internal/screen ./internal/seed ./internal/tui ./internal/wire -update

fmt:
	@gofmt -w .

vet:
	@go vet ./...

# The gate: what has to be clean before a change is done.
#
# internal/room and internal/socket are each run a second time under the race
# detector, and those two are the whole of the concurrency in the repository: the
# registry runs one goroutine per room, and the transport runs a reader and a
# keepalive per connection plus a timer per prompt. The detector is the primary
# net over both — a data race in the room is a battle that stops reproducing from
# its seed, which takes the log format, --verify and undo down with it, and a race
# in the transport is a message delivered to the wrong seat.
#
# Measured against a gate of about a minute: internal/room ~4s, internal/socket
# 4.7s plain and 6.1s under the detector, so the second run costs about 1.4s.
# ⚠️ Some of internal/socket's own time is a deliberate sleep — the timeout test
# runs a one-second allowance against a client that thinks for three — so the
# detector's share of it is smaller than the totals suggest. A race test nobody
# runs is not a net, so both are in the gate rather than in a comment.
check:
	@gofmt -l .
	@go vet ./...
	@go test ./... -count=1
	@go test -race -count=1 ./internal/room/
	@go test -race -count=1 ./internal/socket/

clean:
	@rm -rf bin/
