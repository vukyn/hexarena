# Makefile for hexarena
#
# Thin wrappers over the go commands. The one target worth having is `golden`:
# -update is declared only in the packages that own golden files, so
# `go test ./... -update` fails on every other package.

.PHONY: build install run auto play-tui forge forge-tui forge-tui-en test golden fmt vet check clean

build:
	@go build -o bin/ ./cmd/hexarena ./cmd/hexarena-tui ./cmd/hexforge ./cmd/hexforge-tui ./cmd/hexarena-host

install:
	@go install ./cmd/hexarena ./cmd/hexarena-tui ./cmd/hexforge ./cmd/hexforge-tui ./cmd/hexarena-host

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

# The PvP host: opens one room, prints the code a player pastes, serves the match
# and exits. It plays nothing — both players are clients — so this is the process
# that holds the board, and ctrl-c stops it.
#
# It listens on 13579 by default and puts the address it works out into the code.
# `make host ARGS="-advertise 10.0.0.7"` says which address the code carries on a
# machine where that is ambiguous (a container bridge looks exactly like a LAN),
# and `make host ARGS="-battles 3 -password nhaminh"` is a bo3 behind a gate.
host:
	@go run ./cmd/hexarena-host $(ARGS)

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
# internal/room, internal/socket, cmd/hexarena-host and cmd/hexarena-tui are each
# run a second time under the race detector, and those four are the whole of the
# concurrency in the repository: the registry runs one goroutine per room, the
# transport runs a reader and a keepalive per connection plus a timer per prompt,
# the host binary runs an http server, a signal handler and a bounded shutdown
# beside them, and the game client runs a reader, a keepalive and a Play loop
# beside a bubbletea event loop. The detector is the primary net over all four —
# a data race in the room is a battle that stops reproducing from its seed, which
# takes the log format, --verify and undo down with it; a race in the transport is
# a message delivered to the wrong seat; and a race in the host is two goroutines
# writing one line of somebody's terminal.
#
# ⚠️ cmd/hexarena-tui is the newest of the four and it earned its place the way
# cmd/hexarena-host did. It is the first thing in the repository that **draws** a
# battle another goroutine is stepping: Play steps the mirror while bubbletea's
# loop reads it to draw, which is what socket.Mirror grew an RWMutex for, and the
# client answers its chooser from the goroutine handling a keystroke. A race here
# is a board drawn half a turn into the next one.
#
# ⚠️ **It is by far the most expensive line here and the figure is not what it
# looks like.** Measured: **3.6s plain, 33.1s under the detector**, so the second
# run costs about thirty seconds — roughly doubling the whole gate. Almost none
# of that is the concurrency: the four tests that actually run a match total
# about 4s, and the rest is this package's five *sweeps*, each of which renders
# every screen in both languages at two sizes and plays several battles out to do
# it. The detector is roughly ten times slower at that arithmetic.
#
# It is the whole package all the same, and narrowing it to `-run` the four
# concurrency tests was considered and refused: a filter is a list somebody has to
# remember to add to, which is the "a race test nobody runs" failure one step
# removed. If the gate has to get shorter, the sweeps are what to make cheaper.
#
# ⚠️ cmd/hexarena-host earned its place rather than being added on principle. The
# transport calls Options.Joined and Options.Report on **a connection's own
# goroutine**, while main is printing the banner and the result, so this binary
# prints from three goroutines at once — and the detector caught exactly that on
# the join test's first run, before the lock (main.screen) existed. Measured: 1.3s
# plain, 2.0s under the detector.
#
# Measured against a gate of about a minute: internal/room ~4s, internal/socket
# 4.7s plain and 6.1s under the detector, so the second run costs about 1.4s.
# ⚠️ Some of internal/socket's own time is a deliberate sleep — the timeout test
# runs a one-second allowance against a client that thinks for three — so the
# detector's share of it is smaller than the totals suggest. A race test nobody
# runs is not a net, so all four are in the gate rather than in a comment.
check:
	@gofmt -l .
	@go vet ./...
	@go test ./... -count=1
	@go test -race -count=1 ./internal/room/
	@go test -race -count=1 ./internal/socket/
	@go test -race -count=1 ./cmd/hexarena-host/
	@go test -race -count=1 ./cmd/hexarena-tui/

clean:
	@rm -rf bin/
