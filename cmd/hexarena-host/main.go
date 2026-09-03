// Command hexarena-host serves one PvP match over a LAN.
//
// It opens a room, prints the twelve-character code a player pastes, serves the
// match, prints the result and exits. It plays nothing itself: both players are
// clients, and this is the process that holds the board.
//
//	hexarena-host                       a 3v3, best of one, on the default port
//	hexarena-host -battles 3            a best of three
//	hexarena-host -password nhaminh     a gate against strangers on the network
//	hexarena-host -advertise 10.0.0.7   say exactly which address the code carries
//
// Everything it decides is here, because internal/socket decides none of it: a
// socket.Server is an http.Handler that opens nothing, so the listener, the
// signal handling and every printed word are this binary's.
//
// # ⚠️ A password on the command line is visible to anybody on this machine
//
// `ps` shows the arguments of every process, so -password is readable by any
// other user of the machine while the match runs. That is acceptable **only**
// because of what the password is: a gate that keeps the housemate who guessed
// the room code off the board, explicitly not security — there is no TLS on this
// wire either, and a self-signed certificate implying otherwise was refused on
// the same grounds. → README.md § Not in the first version.
//
// For a shell where that matters, HEXARENA_ROOM_PASSWORD is read when the flag
// is empty. It is not a fix — an environment is only a little less visible than
// an argument list — it is one fewer place the string is written down.
//
// The password is never printed by this binary. wire.Password redacts itself
// under every fmt verb and TestARoomPasswordIsNeverPrintedByTheHost drives a real
// one through everything here that writes.
//
// # What it deliberately does not do
//
//   - **It does not copy the code to the clipboard.** TODO.md left that open and
//     this is the answer: there is no clipboard in Go's standard library, so it
//     would mean shelling out to pbcopy on macOS, xclip or xsel on X11 and
//     wl-copy on Wayland — three external binaries, a per-platform branch, and a
//     silent failure mode on any machine that has none of them. The code is
//     twelve characters from an alphabet that already excludes 0, 1, 8 and 9
//     because people mishear those; it is meant to be read out loud. → wire.RoomCode.
//   - **It runs one room.** The registry holds 256 and the transport serves them
//     all behind one listener, but a host binary that opened several would need a
//     way to say which one finished and which code to print for each, and this is
//     the tool for two friends playing one match.
//   - **It writes no battle log.** A finished match is already a battle.Log the
//     day the room writes one out, and it does not yet. → TODO.md.
package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// DefaultPort is the port this binary listens on unless it is told otherwise,
// and it is **fixed rather than ephemeral** because somebody opening a room
// should get the same port every time: a code that changes shape every run buys
// nothing, and a firewall rule, a router forward or a `pf`/`ufw` line has to name
// a number that stays still.
//
// 13579, and the four things it was picked against:
//
//   - **Nothing is registered on it.** It is free in IANA's list and in
//     /etc/services. The registered 13xxx neighbours are Veritas NetBackup
//     (13720–13785), powwow (13223/13224) and i-zipqd (13160), and none of them
//     is near it.
//   - **Below both operating systems' ephemeral floors**, so the OS will never
//     hand it out from under this process to somebody else's outbound socket.
//     Measured: `net.inet.ip.portrange.first` is **49152** on the darwin machine
//     this was written on, and Linux's default `ip_local_port_range` starts at
//     **32768**. A default inside the ephemeral range would collide at random,
//     which is the exact failure a fixed port exists to remove.
//   - **Not a scan magnet**, and this is the interesting half. **31337 was a
//     candidate and is rejected**: it is Back Orifice's port, so IDS and firewall
//     rule sets flag traffic on it, and a game between friends should not look
//     like a 1999 remote-access trojan to somebody's router.
//   - **Sayable.** "One three five seven nine", which matters for the same reason
//     the room code is twelve characters somebody reads out loud.
//
// ⚠️ **The cost of a fixed port is that "address already in use" becomes an
// ordinary failure** — two hosts on one machine, or the last match's process
// still running — where an ephemeral port could never hit it. That cost is paid
// in listen, which names the port and the flag rather than passing the syscall's
// own words through.
const DefaultPort = 13579

// shutdownGrace is how long a graceful shutdown is given before the process
// stops waiting and says what it was still holding.
//
// It is bounded at all because a wedged socket must not trap the user in a
// process that will not exit, and it is **five seconds** because everything it
// waits on is local: four socket closes and a room goroutine returning, measured
// at hundredths of a second in internal/socket's own shutdown test. Anything
// still outstanding after five seconds is not slow, it is stuck, and the second
// ctrl-c below is the answer to that.
const shutdownGrace = 5 * time.Second

// passwordEnv is where a password is read from when the flag is empty. → the
// package comment, for why the flag exists at all and what it is not.
const passwordEnv = "HEXARENA_ROOM_PASSWORD"

// settings is everything this binary was told, parsed but not yet checked.
//
// # ⚠️ It carries a password, and the type's own redaction does NOT cover it
//
// wire.Password redacts itself under every fmt verb, and room.Config — whose
// fields are exported — is safe by that alone: `%+v` of one prints `[set]`. This
// struct is **not**, and the reason is a rule of fmt rather than anything about
// the password: printing a struct reaches a field's String method through
// reflect.Value.Interface, and an **unexported** field cannot be interfaced, so
// fmt falls back to printing the underlying string. Measured — `%v` of this
// struct printed the password in full before String below existed, and every
// other test in the repository stayed green while it did.
//
// So the redaction is restated here, on this type, rather than inherited. Both
// String and GoString, because %#v takes the second and would otherwise be the
// one verb that still leaked.
type settings struct {
	port      int
	advertise string
	format    int
	battles   int
	allowance int
	turns     int
	password  wire.Password
	seed      uint64
}

// String is the settings as a line, with the password redacted through the type
// that owns the redaction. → the note on the struct, for why this exists at all.
func (s settings) String() string {
	return fmt.Sprintf("port %d, advertise %q, format %d, battles %d, allowance %d, turns %d, password %s, seed %d",
		s.port, s.advertise, s.format, s.battles, s.allowance, s.turns, s.password, s.seed)
}

// GoString is the same for %#v, which does not go through String.
func (s settings) GoString() string { return "main.settings{" + s.String() + "}" }

// screen is one of this binary's two outputs, with a lock around it.
//
// # ⚠️ It exists because this binary prints from more than one goroutine
//
// socket.Options.Joined and socket.Options.Report are called by the transport,
// on **a connection's own goroutine** — one per peer — while main is printing the
// banner and, later, the result. So three or more goroutines write to stdout, and
// nothing about io.Writer promises that is safe. Measured: the race detector
// caught it on the very first run of the join test, which was the whole argument
// for putting ./cmd/hexarena-host on the -race line of `make check`.
//
// os.Stdout would *mostly* survive it — a small write is one write(2) — but
// "mostly" is not a property, an fmt.Fprintf of a long format is several writes,
// and a caller may hand in any writer at all. The lock is four lines and removes
// the question.
type screen struct {
	mu sync.Mutex
	to io.Writer
}

func newScreen(to io.Writer) *screen { return &screen{to: to} }

func (s *screen) Write(raw []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.to.Write(raw)
}

// withLock runs something with this screen's writer held still, for a caller that
// has to read back what was written to it. It is what this binary's own tests use
// to read a buffer the transport is still writing into.
func (s *screen) withLock(read func(io.Writer)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	read(s.to)
}

// errSaid is a failure the caller has already been told about, and it exists to
// stop the one place this binary would otherwise say something twice.
//
// ⚠️ `flag` with ContinueOnError prints the error **and** the usage itself before
// it returns, so main printing the same error again puts it once above a screen
// of usage and once below it. Measured — `hexarena-host -nonsense` said "flag
// provided but not defined" at both ends. The exit code is still 1; only the
// second wording is dropped.
var errSaid = errors.New("already reported")

func main() {
	err := run(os.Args[1:], os.Stdout, os.Stderr)
	switch {
	case err == nil:
		return
	case errors.Is(err, errSaid):
	default:
		fmt.Fprintf(os.Stderr, "hexarena-host: %v\n", err)
	}
	os.Exit(1)
}

// flags is the command line, as a set rather than as globals, so the usage text
// is one value a test can render.
func flags(chosen *settings) *flag.FlagSet {
	set := flag.NewFlagSet("hexarena-host", flag.ContinueOnError)
	set.IntVar(&chosen.port, "port", DefaultPort, "listen on this port; 0 takes any free one")
	set.StringVar(&chosen.advertise, "advertise", "", "the IPv4 address to put in the room code; empty works it out")
	set.IntVar(&chosen.format, "format", int(wire.Format3v3), "units a side; only 3 is offered today")
	set.IntVar(&chosen.battles, "battles", 1, "battles in the series: 1 or 3")
	set.IntVar(&chosen.allowance, "allowance", room.DefaultAllowance, "seconds a player has to answer one prompt")
	set.IntVar(&chosen.turns, "turns", room.DefaultTurnCap, "turns one battle may open before the room stops asking")
	set.Func("password", "a gate against strangers on the network; NOT security, and visible in ps", func(given string) error {
		chosen.password = wire.Password(given)
		return nil
	})
	set.Uint64Var(&chosen.seed, "seed", 0, "the match's seed; 0 draws one and prints it")
	set.Usage = func() {
		out := set.Output()
		fmt.Fprintf(out, "hexarena-host serves one PvP match over a LAN.\n\n")
		fmt.Fprintf(out, "It opens a room, prints the code a player pastes, serves the match and exits.\n")
		fmt.Fprintf(out, "Both players are clients; this process plays nothing.\n\n")
		fmt.Fprintf(out, "Usage:\n  hexarena-host [flags]\n\nFlags:\n")
		set.PrintDefaults()
		fmt.Fprintf(out, "\nA note on -password. It is a gate that keeps strangers on the network off the\n")
		fmt.Fprintf(out, "board and it is NOT security: this is a plain WebSocket on a LAN, with no TLS.\n")
		fmt.Fprintf(out, "Arguments are visible to every other process on this machine through ps, so a\n")
		fmt.Fprintf(out, "password given as a flag is readable by anybody with an account here. Setting\n")
		fmt.Fprintf(out, "%s instead is one fewer place it is written down, and it is\n", passwordEnv)
		fmt.Fprintf(out, "read whenever -password is empty. The password is never printed.\n")
	}
	return set
}

// run is main with its writers handed in, which is what makes the output
// testable — every line this binary prints goes through one of these two.
func run(arguments []string, out, errs io.Writer) error {
	var chosen settings
	set := flags(&chosen)
	set.SetOutput(errs)
	if err := set.Parse(arguments); err != nil {
		// flag has already said what was wrong and printed the usage, so -h is a
		// success and anything else is a failure nobody needs told twice.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %w", errSaid, err)
	}
	if !chosen.password.Set() {
		chosen.password = wire.Password(os.Getenv(passwordEnv))
	}
	if chosen.seed == 0 {
		drawn, err := drawSeed()
		if err != nil {
			return err
		}
		chosen.seed = drawn
	}

	advertised, how, err := advertising(chosen.advertise)
	if err != nil {
		return err
	}
	dependencies, err := dependenciesOf(buildString())
	if err != nil {
		return err
	}

	// Wrapped before anything is printed, because the transport starts writing
	// through them the moment a peer connects. → screen.
	printing, refusals := newScreen(out), newScreen(errs)
	held, err := open(chosen, advertised, dependencies, printing, refusals)
	if err != nil {
		return err
	}
	banner(held, how, printing)
	return held.serve(printing, refusals)
}

// advertising is the address a room code will carry, and how it was decided.
//
// # ⚠️ The flag is deliberately more permissive than the picker
//
// pick refuses loopback and link-local, because it is **choosing on the host's
// behalf** and a code it hands out has to be one a player on another machine can
// use. This is the flag that exists to overrule it, so refusing the same
// addresses here would be the escape hatch declining to be one — and
// `-advertise 127.0.0.1` is exactly how somebody tries the thing out with two
// clients on one machine, which is a real thing to want and which works. So it is
// allowed, and the banner says what it means rather than the flag refusing it.
//
// What is still refused is an address **nothing** can dial: not IPv4, the
// unspecified address, a multicast group, the broadcast address. Those are typos
// with no reading under which they work, and they are refused here rather than at
// wire.EncodeRoom so that the message names the flag they were typed into.
func advertising(given string) (netip.Addr, string, error) {
	if given == "" {
		return autodetect()
	}
	address, err := netip.ParseAddr(given)
	if err != nil {
		return netip.Addr{}, "", fmt.Errorf("-advertise %q is not an address: %w", given, err)
	}
	address = address.Unmap()
	if !address.Is4() {
		return netip.Addr{}, "", fmt.Errorf(
			"-advertise %s is not IPv4, and a room code carries four address bytes; "+
				"a LAN hands out v4, so pass the v4 address of this machine", given)
	}
	if address.IsUnspecified() || address.IsMulticast() || address == netip.AddrFrom4([4]byte{255, 255, 255, 255}) {
		return netip.Addr{}, "", fmt.Errorf(
			"-advertise %s is not an address anything can open a connection to, "+
				"so the code would name somewhere nobody can join", given)
	}
	const told = "was told by -advertise"
	switch {
	case address.IsLoopback():
		return address, told + "; loopback, so only this machine can join", nil
	case address.IsLinkLocalUnicast():
		return address, told + "; link-local, so only this segment can join", nil
	}
	return address, told, nil
}

// drawSeed is the seed a match runs on when none was given.
//
// ⚠️ crypto/rand rather than internal/core/rng, and the distinction matters. The
// engine's rng is the *battle's* randomness and is seeded rather than seeding:
// everything under internal/core is a pure function of its inputs, and drawing a
// number out of the air is exactly what it may not do. This is main, one layer
// out, where a number has to come from somewhere — and the number is printed, so
// the match stays reproducible from it.
func drawSeed() (uint64, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 0, fmt.Errorf("draw a seed: %w", err)
	}
	return binary.BigEndian.Uint64(raw[:]), nil
}

// dependenciesOf is the data this binary embeds and the version it announces.
func dependenciesOf(stamp string) (room.Deps, error) {
	books, err := seed.Books()
	if err != nil {
		return room.Deps{}, fmt.Errorf("load the game data: %w", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		return room.Deps{}, fmt.Errorf("load the cast: %w", err)
	}
	version, err := wire.Local(stamp)
	if err != nil {
		return room.Deps{}, err
	}
	return room.Deps{Books: books, Characters: characters, Version: version}, nil
}

// hosted is a bound listener with one room open behind it, serving.
type hosted struct {
	code     wire.RoomCode
	at       netip.AddrPort
	config   room.Config
	version  wire.Version
	rooms    *room.Registry
	server   *socket.Server
	web      *http.Server
	listener net.Listener
	// finished carries the room's own last reading, once. Buffered so the
	// transport's goroutine never blocks on a main that has not got there yet.
	finished chan room.Reading
}

// open binds the listener, opens the room behind it and starts serving.
//
// # ⚠️ Listen first, THEN open — this order is the requirement
//
// A room code carries the port, and Registry.Open takes the address the code
// will name, so the port has to be **known** before the room exists. With -port 0
// the port is not known until the listener is bound: opening first would put a
// literal 0 in the code, which decodes to a perfectly well-formed room code that
// dials port 0 and can never connect.
//
// It is not a hypothetical made safe by the default. TestTheCodeCarriesThePortThatWasActuallyBound
// drives -port 0 and fails if the order is reversed — and it drives -port 0 on
// purpose, because at a fixed port the wrong order still produces a code carrying
// 13579, which still works, so a test at the default would pass either way and
// measure nothing.
func open(chosen settings, advertised netip.Addr, dependencies room.Deps, out, errs io.Writer) (*hosted, error) {
	configuration := room.Config{
		Format:    wire.Format(chosen.format),
		Battles:   chosen.battles,
		Allowance: chosen.allowance,
		Seed:      chosen.seed,
		TurnCap:   chosen.turns,
		Password:  chosen.password,
	}
	// ⚠️ **Five a side is held back at this flag and nowhere else.**
	// wire.Format5v5 stays valid on the wire on purpose: taking it out of
	// Format.Valid would be a protocol change, and a peer one version either way
	// has to keep agreeing about what the format field can hold. What is held
	// back is the ability to *open* such a room, which is the only place a
	// format is chosen in this repository.
	//
	// Two reasons, both written down elsewhere and both still open. The shipped
	// balance was read at five a side and the room's default has been three
	// since the host binary landed, so five is the format whose numbers are the
	// less wrong of the two but whose board nobody has re-measured; and the
	// ban-and-pick draft cannot seat a 5v5 at all — ten picks and three bans a
	// side want sixteen in the pool and there are eleven. → TODO.md, "read the
	// balance again at 3v3" and "ban and pick".
	if wire.Format(chosen.format) == wire.Format5v5 {
		return nil, fmt.Errorf("five a side is not offered yet: its balance has not been read on this board and a draft cannot seat it; open a 3v3")
	}
	// The room's own refusals are surfaced word for word rather than reworded.
	// "a series of 2 battles is even, and an even series has to invent a rule for
	// a 1–1" is a design decision explaining itself, and a second wording of it
	// here would be a second place to keep it true.
	if err := configuration.Validate(); err != nil {
		return nil, err
	}

	listener, err := listen(chosen.port)
	if err != nil {
		return nil, err
	}
	bound, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, fmt.Errorf("a tcp listener reported a %T as its address", listener.Addr())
	}
	// #nosec G115 -- a TCP port is a uint16 and the field is an int holding one.
	at := netip.AddrPortFrom(advertised, uint16(bound.Port))

	held := &hosted{
		at:       at,
		config:   configuration,
		version:  dependencies.Version,
		rooms:    room.NewRegistry(),
		listener: listener,
		finished: make(chan room.Reading, 1),
	}
	held.server = socket.NewServer(held.rooms, socket.Options{
		Report: func(err error) { fmt.Fprintf(errs, "hexarena-host: %v\n", err) },
		Joined: func(_ wire.RoomCode, seat wire.Seat, name string) {
			fmt.Fprintf(out, "%s joined as %s.\n", playerName(name), seat)
		},
		Finished: func(_ wire.RoomCode, reading room.Reading) {
			select {
			case held.finished <- reading:
			default:
			}
		},
	})
	held.code, err = held.rooms.Open(at, configuration, dependencies)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	held.web = &http.Server{Handler: held.server, ReadHeaderTimeout: shutdownGrace}
	go func() {
		if err := held.web.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(errs, "hexarena-host: serve: %v\n", err)
		}
	}()
	return held, nil
}

// listen binds the port, and turns the one failure a fixed default makes
// ordinary into something a host can act on.
//
// ⚠️ **"address already in use" is the cost of DefaultPort.** With an ephemeral
// port it could not happen; with a fixed one it happens whenever a second host
// is started or the last match's process is still running. `listen tcp
// :13579: bind: address already in use` names neither what to do nor which of
// those it is, so it is caught and rewritten.
func listen(port int) (net.Listener, error) {
	// ":port" and not the advertised address: what this process listens on and
	// what a room code carries are two different decisions, and binding every
	// interface is what lets a host advertise one address while a player on
	// another segment still reaches it.
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err == nil {
		return listener, nil
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return nil, fmt.Errorf(
			"port %d is already in use — another hexarena-host, or the last one still running; "+
				"stop it, or pass -port with another number", port)
	}
	return nil, fmt.Errorf("listen on port %d: %w", port, err)
}

// playerName is a name a peer chose, as it goes on this screen.
//
// ⚠️ It is **somebody else's bytes**: wire.Hello carries whatever a stranger on
// the network typed, and nothing in the transport checks it. A name is bounded
// and an empty one becomes a word, so a peer cannot push the rest of this
// screen off it or produce a line that says nothing.
func playerName(given string) string {
	const longest = 32
	if given == "" {
		return "somebody"
	}
	runes := []rune(given)
	if len(runes) > longest {
		return string(runes[:longest]) + "…"
	}
	return given
}

// banner is everything a host needs on screen before the first player arrives,
// and everything a *joiner* would ask them to read out.
//
// ⚠️ The password is not on it and never will be. Whether one is set is, because
// that is what a player who cannot get in needs to know.
func banner(held *hosted, how string, out io.Writer) {
	fmt.Fprintf(out, "\n  %s\n\n", held.code)
	fmt.Fprintf(out, "  that code means %s (%s)\n", held.at, how)
	fmt.Fprintf(out, "  format      %s, best of %d\n", held.config.Format, held.config.Battles)
	fmt.Fprintf(out, "  allowance   %ds a turn, %d turns a battle at most\n", held.config.Allowance, held.config.TurnCap)
	fmt.Fprintf(out, "  seed        %d\n", held.config.Seed)
	fmt.Fprintf(out, "  password    %s\n", passwordLine(held.config.Password))
	// The two numbers a refused joiner has to compare against their own. A peer
	// whose data digest differs is refused at the gate and cannot be told why in
	// any more detail than an id, so these are what the two people read to each
	// other. → wire.Version.Check.
	fmt.Fprintf(out, "  data        %s\n", held.version.Data.Short())
	fmt.Fprintf(out, "  build       %s\n\n", held.version.Build)
	fmt.Fprintf(out, "waiting for two players. ctrl-c stops.\n")
}

// passwordLine says whether a room has a password without saying what it is.
//
// It is a function rather than the type's own String because those are two
// different jobs: wire.Password.String is a redaction that has to be safe under
// every fmt verb anywhere, and this is one line on one screen that has to be
// plain English.
func passwordLine(password wire.Password) string {
	if password.Set() {
		return "set (players will need it)"
	}
	return "none — anybody with the code can join"
}

// serve waits for the one thing that ends this process and then stops cleanly.
//
// # Two ways out, and both run the same shutdown
//
//   - **The match ends.** The result is printed, everything is shut down, and the
//     process exits 0. A host that had to ctrl-c after the last turn would be a
//     host wondering whether the result was saved.
//   - **SIGINT or SIGTERM.** Shut down, exit 0 — stopping a server on purpose is
//     not a failure, so it is not an exit code.
//
// ⚠️ **A second signal exits immediately.** A graceful shutdown is bounded
// (shutdownGrace) and a bound can still be hit by something genuinely wedged, so
// the user must always have a way out that does not depend on this process
// behaving. The second ctrl-c is that way out, and it is why the shutdown runs on
// its own goroutine rather than inline.
func (held *hosted) serve(out, errs io.Writer) error {
	// Buffered for two, because the second signal is the one that matters and a
	// signal delivered to a full channel is dropped.
	notified := make(chan os.Signal, 2)
	signal.Notify(notified, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(notified)

	select {
	case reading := <-held.finished:
		report(reading, out)
	case <-notified:
		fmt.Fprintf(out, "\nstopping. ctrl-c again to stop without waiting.\n")
	}

	stopped := make(chan error, 1)
	go func() { stopped <- held.stop() }()
	select {
	case err := <-stopped:
		if err != nil {
			// Not returned as the process's error. The match is over either way
			// and the host is not being asked to do anything about a socket that
			// would not close; what it is owed is being told, and being let go.
			fmt.Fprintf(errs, "hexarena-host: %v\n", err)
		}
	case <-notified:
		fmt.Fprintf(errs, "hexarena-host: stopped without waiting for the shutdown to finish\n")
	}
	return nil
}

// stop is the shutdown, bounded: the transport's own — which is what waits for
// the hijacked WebSockets http.Server.Shutdown cannot see — and then the
// listener.
//
// ⚠️ The order is the transport **first**. Closing the listener first would only
// stop new connections; the sockets already upgraded are hijacked, so net/http
// has stopped counting them and would report a clean shutdown over a match still
// being played.
func (held *hosted) stop() error {
	ctx, done := context.WithTimeout(context.Background(), shutdownGrace)
	defer done()
	err := held.server.Shutdown(ctx)
	// The http server is shut down whatever the transport said, because the
	// listener is this process's and holding it open helps nobody.
	if closed := held.web.Shutdown(ctx); closed != nil && err == nil {
		err = fmt.Errorf("close the listener: %w", closed)
	}
	return err
}

// report is the match, as the room's own last reading had it.
//
// ⚠️ It reads room.Reading and computes nothing. A verdict, a winner and the
// per-seat wins are the room's, and a second derivation of any of them here would
// be this binary having an opinion about who won — the exact mistake the missing
// series-standing message exists to avoid, one layer further out.
func report(reading room.Reading, out io.Writer) {
	result := reading.Result
	fmt.Fprintf(out, "\nthe match is %s after %d battle(s).\n", result.Verdict, result.Battles)
	switch {
	case result.Winner.Valid():
		fmt.Fprintf(out, "%s took it, %d–%d.\n", result.Winner, result.Wins[0], result.Wins[1])
	case result.Departed.Valid():
		// Not a loss and not a forfeit: nobody wins a match nobody played out,
		// and on a LAN between friends the enforcement of walking away is social.
		// → wire.ClosureLeft.
		fmt.Fprintf(out, "%s went away, so nobody took it.\n", result.Departed)
	default:
		fmt.Fprintf(out, "nobody took it, %d–%d.\n", result.Wins[0], result.Wins[1])
	}
	for _, fought := range reading.Played {
		fmt.Fprintf(out, "  battle %d: %s, %s took it, %s was home, seed %d%s\n",
			fought.Battle, fought.Outcome, wonBy(fought.Winner), fought.Home, fought.Seed, cappedNote(fought.Capped))
	}
}

// wonBy is the seat that took one battle, in words, because the zero Seat prints
// as nothing at all and a line reading "battle 1: annihilation,  took it" would
// be a bug on screen rather than a draw.
func wonBy(winner wire.Seat) string {
	if !winner.Valid() {
		return "nobody"
	}
	return string(winner)
}

// cappedNote says a battle stopped at the turn cap rather than at an ending,
// because those two read identically in a result line and are not the same thing.
func cappedNote(capped bool) string {
	if capped {
		return " (stopped at the turn cap)"
	}
	return ""
}
