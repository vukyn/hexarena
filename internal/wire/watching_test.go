package wire

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// TestAClientAsksToWatchOnItsHello is the ask crossing, by value, in both
// directions and at both of its values.
//
// The half worth stating out loud is the `false` one. Watch carries no
// omitempty, so a client that means to play writes the flag rather than omitting
// it — which is the same decision Welcome.Drafts made and for the same reason,
// that a mode flag read whole in a log and in the golden is a mode flag a reader
// can tell from a field that was never sent. The golden holds that half as bytes
// (the hello fixture is a player, so `"watch": false` is in the record and
// vanishing from it is a red golden); this holds it as a claim, so the reason is
// somewhere a reader meets it rather than only in a diff.
func TestAClientAsksToWatchOnItsHello(t *testing.T) {
	for _, one := range []struct {
		what  string
		watch bool
		wants string
	}{
		{"a watcher", true, `"watch":true`},
		{"a player", false, `"watch":false`},
	} {
		hello := &Hello{
			Version: Version{Protocol: Protocol, Build: "fixture-build", Data: fixtureDigest()},
			Squad:   fixtureSquad(),
			Watch:   one.watch,
		}
		raw, err := Encode(hello)
		if err != nil {
			t.Fatalf("encode %s: %v", one.what, err)
		}
		if !strings.Contains(string(raw), one.wants) {
			t.Errorf("%s encodes without %s, so the ask does not travel whole: %s",
				one.what, one.wants, raw)
		}
		decoded, err := Decode(raw)
		if err != nil {
			t.Fatalf("decode %s: %v", one.what, err)
		}
		back, isHello := decoded.(*Hello)
		if !isHello {
			t.Fatalf("%s decoded as %T", one.what, decoded)
		}
		if back.Watch != one.watch {
			t.Errorf("%s asked Watch=%v and came back Watch=%v", one.what, one.watch, back.Watch)
		}
	}
}

// TestAWatchingWelcomeNamesNoSeat is the room's half of the same exchange: an
// empty Welcome.Seat **is** "you are watching", and there is no second field
// saying so.
//
// ⚠️ **It is checked on the encoded bytes and it cannot live in the golden.**
// TestEveryMessageIsTheBytesTheGoldenHolds records exactly one fixture per Kind
// and the one welcome fixture has a real seat, so there is no slot there for a
// watching one — which leaves the empty seat, the field a whole class of client
// is identified by, recorded nowhere. Hence `"seat":""` by value: a reader of
// this test can see the wire form, and adding omitempty to Seat (the tidy-up
// that would delete the answer entirely, because an omitted field and a watcher
// would then look identical) reddens it here.
func TestAWatchingWelcomeNamesNoSeat(t *testing.T) {
	watching := &Welcome{Format: Format3v3, Battles: 1, Allowance: 90, TurnCap: 200}
	if !watching.Watching() {
		t.Error("a welcome naming no seat does not report itself as watching, so nothing tells a " +
			"client it came to watch")
	}
	raw, err := Encode(watching)
	if err != nil {
		t.Fatalf("encode a watching welcome: %v", err)
	}
	if !strings.Contains(string(raw), `"seat":""`) {
		t.Errorf("a watching welcome does not carry an empty seat, so the absence a watcher is "+
			"recognised by is not on the wire: %s", raw)
	}
	decoded, err := Decode(raw)
	if err != nil {
		t.Fatalf("decode a watching welcome: %v", err)
	}
	back, isWelcome := decoded.(*Welcome)
	if !isWelcome {
		t.Fatalf("a welcome decoded as %T", decoded)
	}
	if !back.Watching() {
		t.Errorf("a watching welcome came back seated at %q", back.Seat)
	}
	// The other side of the same claim, without which every check above passes on
	// a Watching that returns true always: a seated welcome is a player's.
	for _, seat := range []Seat{SeatHost, SeatGuest} {
		seated := &Welcome{Format: Format3v3, Battles: 1, Allowance: 90, TurnCap: 200, Seat: seat}
		if seated.Watching() {
			t.Errorf("a welcome seating the %s reports itself as watching", seat)
		}
		raw, err := Encode(seated)
		if err != nil {
			t.Fatalf("encode a welcome seating the %s: %v", seat, err)
		}
		if !strings.Contains(string(raw), `"seat":"`+string(seat)+`"`) {
			t.Errorf("a welcome seating the %s does not name it: %s", seat, raw)
		}
	}
}

// TestTheProtocolMintsExactlyTwoSeats is the binding half of the spectator
// decision, held as a count.
//
// ⚠️ **A watcher must not be a third seat, and the reason is that it would
// change who wins.** internal/room/series.go keeps the seats in an array rather
// than a map because the order the two are visited in reaches the roster, and
// the roster's order decides which side wins a speed tie — worth up to sixty
// points in a mirror. So a watcher threaded through the same structure as the
// players is fighting a different battle from the one it came to watch, and
// **nothing else in the suite would say so**: the roster stays legal, and two
// peers that both hold the watcher agree on every digest. Only a comparison
// against the same match played without one shows it, and nobody runs that.
//
// It is therefore held here, at the vocabulary, which is the cheapest place a
// third seat can be stopped. The count is read off the **source** rather than
// off a list written in this file, because a list written here is a list a new
// constant is simply not added to; and it is a count rather than two assertions
// naming host and guest, because two assertions still pass beside a third
// constant.
func TestTheProtocolMintsExactlyTwoSeats(t *testing.T) {
	declared, scanned := declaredSeats(t)
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	if len(declared) == 0 {
		t.Fatal("the scan found no seat constants at all, so it is looking for the wrong shape " +
			"and would pass beside any number of them")
	}
	valid := 0
	for name, value := range declared {
		if Seat(value).Valid() {
			valid++
			continue
		}
		t.Errorf("the protocol declares %s = %q and Seat.Valid does not accept it, so a room "+
			"would refuse a seat it hands out", name, value)
	}
	if valid != 2 {
		t.Errorf("the protocol declares %d valid seats (%v), want exactly the two a room hands "+
			"out: a watcher takes NO seat, because the order the seats are visited in reaches "+
			"the roster and the roster's order decides which side wins a speed tie",
			valid, declared)
	}
	// And Valid did not widen without a constant to widen it, which is the other
	// way a third seat arrives: the two names a spectator would most likely be
	// given, plus the zero value, which means "no seat" and must keep meaning it.
	for _, near := range []Seat{"", "spectator", "watcher", "viewer", "Host", "host "} {
		if near.Valid() {
			t.Errorf("Seat(%q) reports itself valid, so something other than the room's two "+
				"places can sit down", string(near))
		}
	}
	t.Logf("scanned %d source files; %d seats declared, %d valid", scanned, len(declared), valid)
}

// TestTheOnlyCodeAboutAWatcherIsTheCap is the "no new code" half of step one,
// written as the decision rather than as a pinned count — and ⚠️ **the decision
// it holds is narrower than the one it used to claim**.
//
// It was TestAWatcherIsRefusedByNoCodeOfItsOwn and it banned *any* code named
// after watching, which reads as one rule and is two. The rule step one decided
// is about **admitting** a watcher: a watcher's squad is **ignored**, not
// refused, because CodeSquadUnwanted exists for a squad quietly dropped — a
// player watching the side they spent an evening building fail to appear — and a
// watcher expects no side of its own, so nothing fails to appear and answering
// it would misdirect. That rule is untouched. What the flat ban also forbade,
// with no argument behind it, was a refusal for a watcher that **cannot be let
// in at all**: the transport carries a bounded number of connections per table,
// and CodeRoomFull cannot say so — both books word it about the two seats and
// then advise opening a room of your own, which is nonsense advice for somebody
// who came to watch this match. CodeRoomFull's own comment predicted this code
// before it existed. → CodeTooManyWatchers, and TODO.md § *Spectators* step 4.
//
// ⚠️ **That paragraph said "its admission" and step six made it false.** The
// flat ban was narrowed once to let the cap in, and the sentence describing what
// was left still read that a code named for a watcher's *admission* was
// forbidden — which would forbid the second entry below as well, on the same
// no-argument-behind-it grounds the first narrowing threw out. Whether a room
// takes watchers at all is a host's choice (room.Config.Watchable), and a room
// that never opened to spectators cannot be worded by the cap: the cap's own
// sentence tells the reader to wait for one of the current watchers to leave,
// which is false rather than merely unhelpful where nobody is watching and
// nobody can. → CodeWatchingClosed, and TODO.md § *Spectators* step 6.
//
// So the allowlist is exactly two, both of them the same kind of refusal — a
// watcher that **cannot be let in**, for a reason no code about a seat can
// state — and what the walk still fails on is the code step one was about: a
// refusal named for a watcher's **squad or its side**, which is ignored rather
// than refused because a watcher expects no side of its own.
//
// ⚠️ **This deliberately does not pin CodeCount, and the reason is worth
// knowing.** Two tests outside this package already make a code impossible to
// add quietly: internal/i18n/protocol_test.go walks CodeCount against both word
// books, and cmd/hexarena-tui/shown_test.go holds `gate + inMatch + owed ==
// CodeCount-1` against codes produced out of a real room and a real server. So
// an added code is already loud. A literal count on top of those would be a
// figure pinned twice, where the second copy sits somewhere that reads like a
// safety check and gets bumped rather than read — which is a mistake this
// repository has already made once and written down.
func TestTheOnlyCodeAboutAWatcherIsTheCap(t *testing.T) {
	// The words a code for a watcher would be named with. A name is what a peer
	// reads, so a code about watching cannot avoid saying so in its name.
	about := []string{"watch", "spectat", "viewer", "observ"}
	// The one that is allowed to, with the reason it is not the mistake above.
	allowed := map[Code]string{
		CodeTooManyWatchers: "the transport's cap on how many connections one " +
			"table carries, which no code about a seat can say",
		CodeWatchingClosed: "the room never having been opened to spectators, " +
			"which the cap cannot say either — its wording tells the reader to " +
			"wait for a watcher to leave, and there are none",
	}
	found := 0
	for value := range CodeCount {
		code, name := Code(value), codeNames[value]
		for _, word := range about {
			if !strings.Contains(name, word) {
				continue
			}
			if because, spared := allowed[code]; spared {
				found++
				t.Logf("the refusal %q is named for a watcher and is allowed to be: %s", name, because)
				continue
			}
			t.Errorf("the protocol declares the refusal %q: a watcher's squad is ignored "+
				"rather than refused, because a watcher expects no side of its own, so "+
				"nothing fails to appear — → Hello.Watch", name)
		}
	}
	// Non-vacuity, three ways: the walk above passes over an empty table, the
	// words it looks for have to be findable in the first place, and an
	// allowlist whose entry the walk never reaches is an entry that has stopped
	// meaning anything.
	if CodeCount == 0 {
		t.Fatal("there are no codes to walk")
	}
	if !strings.Contains("a watcher", about[0]) {
		t.Fatalf("the walk looks for %q, which matches nothing it would need to match", about[0])
	}
	if found != len(allowed) {
		t.Errorf("the walk spared %d of the %d codes the allowlist names, so an entry in it "+
			"names a code that is gone or has been renamed out of the words above", found, len(allowed))
	}
}

// declaredSeats is every constant of type Seat this package declares, read off
// the source: the constant's name against its wire value, plus how many files
// were scanned to find them.
//
// It parses rather than greps, in the shape TestTheProtocolCannotReadAClock uses
// for imports, so a mention of `SeatSpectator` in a comment is not a constant and
// a constant spelled across two lines still is.
func declaredSeats(t *testing.T) (map[string]string, int) {
	t.Helper()
	declared := make(map[string]string, 2)
	scanned := 0
	for _, entry := range mustReadPackageDir(t) {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			block, isGeneric := decl.(*ast.GenDecl)
			if !isGeneric || block.Tok != token.CONST {
				continue
			}
			// A const block carries its type forward over specs that state no
			// type and no value of their own, so the walk carries it forward too
			// — a seat declared that way is still a seat.
			carried := false
			for _, spec := range block.Specs {
				value, isValue := spec.(*ast.ValueSpec)
				if !isValue {
					continue
				}
				switch typed := value.Type.(type) {
				case *ast.Ident:
					carried = typed.Name == "Seat"
				case nil:
					if len(value.Values) > 0 {
						carried = false
					}
				default:
					carried = false
				}
				if !carried {
					continue
				}
				for at, ident := range value.Names {
					literal := ""
					if at < len(value.Values) {
						if basic, isBasic := value.Values[at].(*ast.BasicLit); isBasic && basic.Kind == token.STRING {
							unquoted, err := strconv.Unquote(basic.Value)
							if err != nil {
								t.Fatalf("%s: unquote %s: %v", name, basic.Value, err)
							}
							literal = unquoted
						}
					}
					declared[ident.Name] = literal
				}
			}
		}
	}
	return declared, scanned
}
