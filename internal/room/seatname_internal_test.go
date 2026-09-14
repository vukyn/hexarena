// ⚠️ **This is the only test in this package written from INSIDE it, and the
// reason is the finding rather than a preference.** peer.name is stored by
// Room.Join and, as of the change this file arrived with, **read by nothing** —
// the host's screen takes the name off the hello in internal/socket, not off the
// seat. So there is no exported surface to observe it through, and a rule with
// nothing observing it is a rule a mutation deletes for free. Every other test
// here stays in room_test, where the gate's behaviour is visible from outside;
// this one asserts a field, so it lives where the field does. When a reader is
// wired up (→ wire.Hello.Name's own note), this may move out and assert through
// it instead.
package room

import (
	"testing"

	"github.com/vukyn/hexarena/internal/plain"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestASeatsNameIsCleanedOnTheWayIn covers the door wire.Hello.UnmarshalJSON
// cannot: a hello built in Go never passed a decoder, so a test, an in-process
// client or anything the room is handed directly would put raw bytes on a seat
// that outlives the message they came in.
//
// ⚠️ **It is deliberately NOT decoded first.** Marshalling the hostile name and
// decoding it before handing it over would make this test pass with the gate's
// call deleted, because the decoder would already have done the work — which is
// the whole class of vacuous test this file exists to avoid.
func TestASeatsNameIsCleanedOnTheWayIn(t *testing.T) {
	// A drafting room, so the hello may bring no squad at all: what is under test
	// is the name on the seat, and building a legal squad here would be a fixture
	// that moves with the balance data for no reason. → Config.Drafts.
	opened := draftingRoom(t)
	hostile := "Bảo\x1b]52;c;cm0gLXJmIH4=\x07"
	admission, _, err := opened.Join(wire.Hello{Version: localVersion(t), Name: hostile})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if !admission.Seat.Valid() {
		t.Fatalf("the hello was refused, so nothing was seated: %+v", admission)
	}
	seated, found := "", false
	for _, each := range opened.seated {
		if each.taken {
			seated, found = each.name, true
		}
	}
	if !found {
		t.Fatal("the gate reported a seat and the room holds no taken one")
	}
	if seated == hostile {
		t.Fatalf("the seat holds the hello's bytes unchanged: %q", seated)
	}
	if !plain.Inert(seated) {
		t.Errorf("a seat holds %q, which a terminal would obey", seated)
	}
	if want := wire.CleanName(hostile); seated != want {
		t.Errorf("the seat holds %q, want %q — the gate is not using wire.CleanName", seated, want)
	}
}

// TestAnOrdinaryNameReachesTheSeatUnchanged is the other half, and it is what
// stops the cleaning from being a silent change to what a player is called.
func TestAnOrdinaryNameReachesTheSeatUnchanged(t *testing.T) {
	opened := draftingRoom(t)
	if _, _, err := opened.Join(wire.Hello{Version: localVersion(t), Name: "Nguyễn Thị Ánh"}); err != nil {
		t.Fatalf("join: %v", err)
	}
	for _, each := range opened.seated {
		if each.taken && each.name != "Nguyễn Thị Ánh" {
			t.Errorf("an ordinary name reached the seat as %q", each.name)
		}
	}
}

func draftingRoom(t *testing.T) *Room {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	opened, err := New(Config{
		Format:    wire.Format3v3,
		Battles:   1,
		Allowance: DefaultAllowance,
		Seed:      11,
		TurnCap:   DefaultTurnCap,
		Drafts:    true,
	}, Deps{Books: books, Characters: characters, Version: localVersion(t)})
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	return opened
}

func localVersion(t *testing.T) wire.Version {
	t.Helper()
	version, err := wire.Local("seat-name-test")
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	return version
}
