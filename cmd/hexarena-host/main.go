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
//	hexarena-host -logs ./logs          write each finished battle out, replayable
//	hexarena-host -version              say what this binary is, and host nothing
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
//   - **It writes a battle log only when asked.** -logs names a directory and
//     each finished battle lands in it as a `battle.Log`, replayable with
//     `hexarena --replay FILE --verify`. Without the flag it writes nothing:
//     the room composes the log either way, because it costs a cursor, but a
//     binary that filled a directory on every match would be doing something
//     nobody typed.
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
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/discovery"
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
	// budget is the chess clock: seconds each player has for the whole match, and
	// nought is no clock at all. → room.Config.Budget.
	budget   int
	turns    int
	password wire.Password
	seed     uint64
	// draft says the two sides ban and pick before they fight, out of one shared
	// pool, rather than each bringing a squad built at home.
	//
	// ⚠️ It is a room's whole shape rather than a preference, and it is why the
	// banner has to say so: a drafting room **refuses a squad**, so a player who
	// joins with one selected is turned away and has to join again with none.
	// Until the handshake is two-phase that refusal is the only other thing that
	// says it — a client cannot know the room drafts until it has been welcomed,
	// and the hello carrying its squad goes first. → TODO.md.
	draft bool
	// watch says a spectator may watch this match: a client that asked to watch
	// is welcomed with no seat and handed the match off the room's record.
	//
	// ⚠️ **Off by default, and that is a decision rather than a bounds check.**
	// A room is for the two people in it unless the host said otherwise, so the
	// flag is what says otherwise — and a room that was not opened to spectators
	// refuses one by name (wire.CodeWatchingClosed) rather than letting them in
	// quietly. → room.Config.Watchable.
	watch bool
	// browse says this room announces itself on the local network over mDNS, so
	// a client can list it and never be read a code at all.
	//
	// ⚠️ **Off by default, for the reason -watch is.** A room is for the people
	// the host told about it; announcing it puts the code in front of every
	// machine on the segment, which is a different room from the one somebody
	// opened by default. The flag is what says otherwise.
	//
	// ⚠️ The announcement carries the room CODE, and the code is the whole of
	// what a client needs to join. A password still gates the join
	// (wire.CodeBadPassword), so -browse and -password are not in tension — but a
	// room with no password, announced, is a room anybody on the LAN may walk
	// into. That is the point of a LAN game and it is worth typing on purpose.
	browse bool
	// logs is the directory each finished battle is written into as a
	// `battle.Log`, and empty writes nothing.
	//
	// ⚠️ **Off by default, because writing files is not what a host was asked
	// to do.** A room hands the log out whether anybody wants it or not — it
	// costs a cursor — and this flag is the only thing that turns it into a file.
	// A binary that quietly filled a directory on every match would be a
	// side effect nobody typed.
	//
	// What lands there re-runs: `hexarena --replay <file> --verify` rebuilds the
	// battle from the log's own seed and roster and checks every event. That is
	// the whole point of the flag, and it is why the log carries the placement
	// rather than a reference to one — a PvP squad is built on a player's own
	// machine and is in no book this binary could look it up in.
	logs string
	// version is the ask that is answered instead of hosting anything: print
	// what this binary is and exit. Everything else in this struct configures a
	// room, and this one says no room is wanted.
	version bool
}

// String is the settings as a line, with the password redacted through the type
// that owns the redaction. → the note on the struct, for why this exists at all.
func (s settings) String() string {
	return fmt.Sprintf("port %d, advertise %q, format %d, battles %d, allowance %d, budget %d, turns %d, password %s, seed %d, draft %t, watch %t, browse %t, logs %q, version %t",
		s.port, s.advertise, s.format, s.battles, s.allowance, s.budget, s.turns, s.password, s.seed, s.draft, s.watch, s.browse, s.logs, s.version)
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

// programName is what this binary is called, and it is what -version puts on the
// first line of its report. It is a constant rather than a literal per call site
// because cmd/hexarena-tui keeps one too and the two reports have to name their
// own binaries with the same certainty they name the same three numbers.
const programName = "hexarena-host"

func main() {
	err := run(os.Args[1:], os.Stdout, os.Stderr)
	switch {
	case err == nil:
		return
	case errors.Is(err, errSaid):
	default:
		fmt.Fprintf(os.Stderr, "%s: %v\n", programName, err)
	}
	os.Exit(1)
}

// flags is the command line, as a set rather than as globals, so the usage text
// is one value a test can render.
func flags(chosen *settings) *flag.FlagSet {
	set := flag.NewFlagSet(programName, flag.ContinueOnError)
	set.IntVar(&chosen.port, "port", DefaultPort, "listen on this port; 0 takes any free one")
	set.StringVar(&chosen.advertise, "advertise", "", "the IPv4 address to put in the room code; empty works it out")
	set.IntVar(&chosen.format, "format", int(wire.Format3v3), "units a side; only 3 is offered today")
	set.IntVar(&chosen.battles, "battles", 1, "battles in the series: 1 or 3")
	set.IntVar(&chosen.allowance, "allowance", room.DefaultAllowance, "seconds a player has to answer one prompt")
	set.IntVar(&chosen.turns, "turns", room.DefaultTurnCap, "turns one battle may open before the room stops asking")
	set.IntVar(&chosen.budget, "budget", 0, "seconds each player has for the WHOLE match; 0 is no clock, and the per-turn allowance still applies")
	set.Func("password", "a gate against strangers on the network; NOT security, and visible in ps", func(given string) error {
		chosen.password = wire.Password(given)
		return nil
	})
	set.Uint64Var(&chosen.seed, "seed", 0, "the match's seed; 0 draws one and prints it")
	set.BoolVar(&chosen.draft, "draft", false, "ban and pick from one shared pool instead of bringing squads; join with NO squad")
	set.BoolVar(&chosen.watch, "watch", false, "let spectators watch; they paste the SAME code the players do")
	set.BoolVar(&chosen.browse, "browse", false, "announce this room on the local network so a client can list it without a code")
	set.StringVar(&chosen.logs, "logs", "", "write each finished battle here; replay one with `hexarena --replay FILE --verify`")
	// The same sentence cmd/hexarena-tui's flag shows in English, and the same
	// three numbers. That client takes its descriptions from internal/i18n
	// because it has two languages to keep honest; this binary has one and reads
	// its wording off the page, so the two are worded alike by hand and the
	// *output* is the thing neither of them spells twice. → wire.Version.Report.
	set.BoolVar(&chosen.version, "version", false, "print the build, protocol and data, then exit")
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
	// ⚠️ **Answered here, ahead of everything, and the position is the feature.**
	// Somebody asking what this binary is has asked for none of what follows: a
	// seed is drawn out of crypto/rand, an address is worked out by walking this
	// machine's interfaces, the embedded books are loaded and a listener is bound
	// on a port somebody else may want. So -version prints and returns before the
	// first of those, and it is deliberately ahead of the *configuration checks*
	// too — `-version -battles 2` says what this binary is rather than refusing a
	// series of two, because a question about the binary is not a request to host
	// a match. TestVersionSaysWhatThisBinaryIsAndHostsNothing drives exactly that.
	if chosen.version {
		version, err := wire.Local(buildString())
		if err != nil {
			return err
		}
		fmt.Fprint(out, version.Report(programName))
		return nil
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
	// The room draws no randomness of its own — that is what keeps it a state
	// machine a test can drive a message at a time — so the source is handed in
	// here, where a binary that is already reading the clock and the network is
	// the honest place for it. → room.Deps.Tokens.
	return room.Deps{
		Books: books, Characters: characters, Version: version,
		Tokens: wire.NewSeatToken,
	}, nil
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
	// logs is the directory finished battles are written into, and empty writes
	// none. → the settings field of the same name.
	logs string
	// announced is this room's mDNS advertisement, and is nil when -browse was
	// not given. Closing it is safe either way, which is why stop can do it
	// unconditionally.
	announced *discovery.Advertisement
	// browseAsked is whether -browse was given, and it is a second field rather
	// than a nil check because the two answer different questions: announced says
	// whether this room IS on the network, and this says whether it was meant to
	// be. A host who asked and did not get it needs the banner to say so — an
	// empty list on the other machine looks exactly like nobody hosting.
	browseAsked bool
	// announceErr is why the announcement did not happen, and is nil when it did
	// or when nobody asked.
	//
	// ⚠️ **`announced == nil` and `announceErr == nil` together mean nobody
	// asked**, and that is what makes the three states tell each other apart. A
	// host who typed -browse must end up with exactly one of the two set — a room
	// that asked and got neither an advertisement nor a reason is a code path that
	// silently did nothing, which is what a test can catch and a person cannot.
	announceErr error
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
		Budget:    chosen.budget,
		Seed:      chosen.seed,
		TurnCap:   chosen.turns,
		Password:  chosen.password,
		Drafts:    chosen.draft,
		Watchable: chosen.watch,
	}
	// ⚠️ **A drafting bo3 used to be refused here, in words, and is not any
	// more.** The refusal named two flags rather than two struct fields, because
	// what a host types is flags — and it stood while "what a draft means across
	// a series" was undecided. It is decided: a draft a battle, out of a fresh
	// pool each time, so `-draft -battles 3` is three ban-and-picks and up to
	// three different squads. → room.Room.redraft, and the banner below, which
	// says so where a host reads it.
	// ⚠️ **Five a side used to be held back at this flag and nowhere else, and it
	// is open now.** The whole of the history is worth keeping, because the guard
	// was removed on a measurement rather than on a hunch and the next reader
	// deserves to know which reasons died of what.
	//
	// There were two. The draft half closed while the draft was being built —
	// `draft.Fits` seats a 5v5 with room to spare. The balance half said "the
	// shipped balance was read at five a side", and that was never true: every
	// figure in this repository was read at three or fewer, `roster.json`
	// included. So the board nobody had read was five, and reading it found that
	// it resolves and stays fair — a mirrored squad comes to exactly 500‰ over two
	// hundred battles with nothing endless, and screening is worth *more* there
	// than at three.
	//
	// What it also found was the one real blocker, and it was a **design
	// collision rather than a number**: a summon lives in the gap between what a
	// side FIELDS and what the board ADMITS, and one constant was doing both jobs at
	// five, so a full side computed `room = 0`, `summonWorth` priced every
	// summoning skill at nought and the rating never cast one. Three shipped
	// skills and the `diglett.three` build were dead slots here.
	//
	// `ENG-013` split the constant — `hex.BoardSlots` (every formation slot)
	// against `hex.MaxSquadSize` (what a side is fielded with) — so the gap is
	// four slots wide at this format. Re-measured on the same mirrored instrument:
	// a 5v5 goes from **0** casts and 0 arrivals to hundreds of each, the mirror
	// stays exactly 500‰ with nothing endless over two hundred battles, battles
	// run about 2% longer, and **3v3 does not move at all** — the readings are in
	// `docs/balance.md` § *Five a side, read at last*.
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
		logs:     chosen.logs,

		browseAsked: chosen.browse,
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
	if chosen.browse {
		// ⚠️ **After the room is open and before the first player can arrive.**
		// Announcing a code the registry has not issued yet would put a row on
		// somebody's screen that refuses the join behind it, and there is no
		// moment earlier than this at which the code exists.
		//
		// The advertised address is the one the code carries, so the SRV record
		// and the code cannot disagree about where this room is. → the note on
		// discovery.Advertise about registering as a proxy.
		//
		// A failure here does not stop the room. Browsing is a way to find a
		// room the code already reaches, so a machine that cannot multicast — a
		// container, a locked-down laptop — should host a perfectly good match
		// and say that one convenience is missing, rather than refuse to open.
		announced, err := discovery.Advertise(discovery.Room{
			Code:     held.code,
			Format:   configuration.Format,
			Battles:  configuration.Battles,
			Draft:    configuration.Drafts,
			Watch:    configuration.Watchable,
			Data:     dependencies.Version.Data.Short(),
			Protocol: dependencies.Version.Protocol,
		}, at)
		if err != nil {
			held.announceErr = err
			fmt.Fprintf(errs, "%s: %v\n", programName, err)
		} else {
			held.announced = announced
		}
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
	// ⚠️ **A drafting room says so, and says what to do about it.** It refuses a
	// squad, so a player who joins with one selected is turned away — and until
	// the handshake is two-phase that refusal is the only other thing that tells
	// them, because a client cannot know the room drafts until it has been
	// welcomed and the hello carrying its squad goes first. The line is drawn
	// only when it is true: a banner that always mentioned drafting would be a
	// line every ordinary host reads past, which is how the one that matters
	// stops being read.
	if held.config.Drafts {
		fmt.Fprintf(out, "  draft       yes — both sides ban and pick here; JOIN WITH NO SQUAD\n")
		// ⚠️ **Said only in a series**, for the draft line's own reason: a line
		// every ordinary drafting host reads past is how the one that matters
		// stops being read. What it has to say is that the squads are not kept —
		// a player who drafted a side they liked will otherwise expect it back
		// in battle two, and it is a fresh pool and a fresh pick every time.
		if held.config.Battles > 1 {
			fmt.Fprintf(out, "  draft       ...once per battle — %d ban-and-picks, out of a fresh pool each time\n",
				held.config.Battles)
		}
	}
	// ⚠️ **What a host has to be told is that there is no second code.** The
	// room code decision the whole feature rests on is that a spectator pastes
	// the same twelve characters a player does — one flag on the hello rather
	// than a second code space, a second thing to print and a second thing to
	// mistype — and the host is the one person who reads that code out, so the
	// host is where it has to be said. Drawn only when it is true, for the draft
	// line's reason: a line every ordinary host reads past is how the one that
	// matters stops being read.
	if held.config.Watchable {
		fmt.Fprintf(out, "  watch       yes — spectators paste the SAME %d characters the players do\n",
			wire.RoomCodeLength)
	}
	// ⚠️ **Drawn on the ASKING rather than on the succeeding**, and the two are
	// different lines on purpose. A host who typed -browse and is not announced
	// has to see that, because the symptom on the other machine — an empty list —
	// is indistinguishable from "nobody is hosting". The failure itself already
	// went to stderr when open tried; this is the same fact where the host is
	// looking. Drawn only when asked for, for the draft and watch lines' reason.
	if held.announced != nil {
		fmt.Fprintf(out, "  browse      yes — this room is announced on the local network\n")
	} else if held.browseAsked {
		fmt.Fprintf(out, "  browse      ASKED FOR AND FAILED: %v — the code still works\n",
			held.announceErr)
	}
	fmt.Fprintf(out, "  allowance   %ds a turn, %d turns a battle at most\n", held.config.Allowance, held.config.TurnCap)
	// ⚠️ **Drawn only when there is one**, for the draft and watch lines' reason:
	// a line every ordinary host reads past is how the one that matters stops
	// being read. What it has to say is that the two clocks BOTH apply — a host
	// who set a budget and expects it to replace the allowance would otherwise
	// find turns still cut off at ninety seconds.
	if held.config.Budget > 0 {
		fmt.Fprintf(out, "  budget      %ds each for the whole match, on top of the allowance above\n",
			held.config.Budget)
	}
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
		// After the result and not before it: the match is what the host is
		// waiting for, and a directory that will not take a file is a thing to
		// be told about rather than a reason to withhold the result.
		if err := held.write(reading, out); err != nil {
			fmt.Fprintf(errs, "%s: %v\n", programName, err)
		}
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
	// First, and before anything that can take time. The withdrawal is a
	// goodbye packet, and a browser that never gets one keeps offering this room
	// until the record's TTL runs out — a row that pastes a code at a listener
	// which is already closing. → discovery.Advertisement.Close.
	held.announced.Close()
	err := held.server.Shutdown(ctx)
	// The http server is shut down whatever the transport said, because the
	// listener is this process's and holding it open helps nobody.
	if closed := held.web.Shutdown(ctx); closed != nil && err == nil {
		err = fmt.Errorf("close the listener: %w", closed)
	}
	return err
}

// write puts each finished battle into the -logs directory as a `battle.Log`,
// and does nothing at all when no directory was named.
//
// ⚠️ **The name is the room's code, the battle number and the seed**, and it is
// deliberately not the shape internal/forge writes under `data/battles/`. That
// one is `<home>-vs-<away>-seed<N>.json`, and it can be: a spar is between two
// squads the library holds by id. A PvP match is between two people whose squads
// were built on their own machines and are in no book here — the seats are
// "host" and "guest" every time — so naming a file after them would put the same
// two words on every file this binary ever wrote. The code and the battle number
// are what tell one match's files from another's.
//
// ⚠️ **A battle nobody finished is in no reading**, so nothing here has to decide
// whether to write one: an abandoned match records no BattleResult at all, which
// is the room's rule and not a second one here. → room.BattleResult.Log.
//
// The directory is created if it is missing, because -logs names where the files
// go rather than an existing place, and a host who typed a path should not have
// to make it first.
func (held *hosted) write(reading room.Reading, out io.Writer) error {
	if held.logs == "" {
		return nil
	}
	if len(reading.Played) == 0 {
		return nil
	}
	if err := os.MkdirAll(held.logs, 0o755); err != nil {
		return fmt.Errorf("make %s to write the logs into: %w", held.logs, err)
	}
	for _, fought := range reading.Played {
		raw, err := battle.MarshalLog(fought.Log)
		if err != nil {
			return fmt.Errorf("encode battle %d: %w", fought.Battle, err)
		}
		name := filepath.Join(held.logs,
			fmt.Sprintf("%s-battle%d-seed%d.json", held.code, fought.Battle, fought.Seed))
		if err := os.WriteFile(name, raw, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
		fmt.Fprintf(out, "wrote %s (%d events, %d choices)\n",
			name, len(fought.Log.Events), len(fought.Log.Choices))
	}
	fmt.Fprintf(out, "replay one with: hexarena --replay FILE --verify\n")
	return nil
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
