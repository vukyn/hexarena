package discovery

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/wire"
)

// aRoomCode is a well-formed code for the fixtures below, made rather than
// written down so it stays a code the wire package agrees with.
func aRoomCode(t *testing.T, at string, index uint8) wire.RoomCode {
	t.Helper()
	code, err := wire.EncodeRoom(netip.MustParseAddrPort(at), index)
	if err != nil {
		t.Fatalf("encode a room code: %v", err)
	}
	return code
}

// TestARoomSurvivesItsOwnTextRecord is the round trip, and it is the whole of
// what a browse and an advertisement have to agree about.
//
// Every field is set to something distinguishable, because the failure this
// catches is a field written under one key and read under another — two spellings
// of one name, which compiles and loses exactly one column of the listing.
func TestARoomSurvivesItsOwnTextRecord(t *testing.T) {
	code := aRoomCode(t, "192.168.1.5:13579", 3)
	sent := Room{
		Code: code, Format: wire.Format5v5, Battles: 3,
		Draft: true, Watch: true, Data: "6537b5d935f7", Protocol: 7,
	}
	at := netip.MustParseAddrPort("192.168.1.5:13579")
	heard, err := parseRoom(string(code), sent.text(), at)
	if err != nil {
		t.Fatalf("read back a record this package wrote: %v", err)
	}
	sent.At = at
	if heard != sent {
		t.Errorf("a room went out as %+v and came back as %+v", sent, heard)
	}
}

// TestTheFalseHalfOfEveryFlagSurvivesToo is the other arm, and it is separate
// because a boolean read as `value == "1"` is right for both arms only by
// accident of one of them.
func TestTheFalseHalfOfEveryFlagSurvivesToo(t *testing.T) {
	code := aRoomCode(t, "10.0.0.9:4444", 0)
	sent := Room{Code: code, Format: wire.Format3v3, Battles: 1, Data: "aaaaaaaaaaaa", Protocol: 1}
	heard, err := parseRoom(string(code), sent.text(), netip.AddrPort{})
	if err != nil {
		t.Fatalf("read back a record: %v", err)
	}
	if heard.Draft || heard.Watch {
		t.Errorf("a room that drafts nothing and watches nothing came back as %+v", heard)
	}
}

// TestARecordWithNoRoomCodeIsRefused is the one thing that may fail a record.
//
// The code is the only field a client can act on, so an entry a player can see
// and cannot join is worse than an entry that never appeared. → the package
// comment, where the lenient rule and its one exception are argued together.
func TestARecordWithNoRoomCodeIsRefused(t *testing.T) {
	for _, instance := range []string{"", "not a code", "YCUACBJVBMA", "YCUACBJVBMAQQ", "01234567890!"} {
		if _, err := parseRoom(instance, nil, netip.AddrPort{}); err == nil {
			t.Errorf("%q was accepted as a room code", instance)
		}
	}
}

// TestAKeyThisBuildDoesNotKnowIsIgnoredRatherThanRefused is the lenient rule, and
// it is the rule this repository takes the other way round everywhere else.
//
// A data file is authored by this build, so an unknown field there is a mistake
// and DisallowUnknownFields says so. A TXT record is written by another machine,
// which may be a version ahead — so refusing it over a key this build has not
// heard of would make every added key a flag day on a LAN.
func TestAKeyThisBuildDoesNotKnowIsIgnoredRatherThanRefused(t *testing.T) {
	code := aRoomCode(t, "192.168.1.5:13579", 1)
	text := append([]string{"chess_clock=1", "rejoin=45", "notakeyvalue"},
		Room{Format: wire.Format3v3, Battles: 1, Data: "d"}.text()...)
	heard, err := parseRoom(string(code), text, netip.AddrPort{})
	if err != nil {
		t.Fatalf("a record from a newer build was refused: %v", err)
	}
	if heard.Format != wire.Format3v3 || heard.Battles != 1 {
		t.Errorf("the keys this build does know were not read: %+v", heard)
	}
}

// TestAValueThisBuildCannotReadLeavesItsFieldAlone is the same rule one level
// down, and it is the half a lenient reader gets wrong.
//
// Skipping an unknown KEY is easy. A key this build knows, carrying a value it
// cannot parse, is the case where "be lenient" and "refuse what is broken" pull
// apart — and the answer is the same one: the room is still joinable, so it is
// still listed, with one column blank.
func TestAValueThisBuildCannotReadLeavesItsFieldAlone(t *testing.T) {
	code := aRoomCode(t, "192.168.1.5:13579", 1)
	heard, err := parseRoom(string(code),
		[]string{"format=huge", "battles=", "proto=x", "data=6537b5d935f7"}, netip.AddrPort{})
	if err != nil {
		t.Fatalf("a record with an unreadable value was refused: %v", err)
	}
	if heard.Code != code {
		t.Errorf("the code was lost: %+v", heard)
	}
	if heard.Data != "6537b5d935f7" {
		t.Errorf("a readable field beside an unreadable one was dropped: %+v", heard)
	}
	if heard.Format != 0 || heard.Battles != 0 || heard.Protocol != 0 {
		t.Errorf("an unreadable value became a number: %+v", heard)
	}
}

// TestTheTextRecordIsWrittenInAFixedOrder.
//
// Two hosts advertising the same room must produce the same bytes, or a capture
// of one and a capture of the other differ for a reason nothing did. Ranging a
// map would be the ordinary way to write this function and Go randomises that.
func TestTheTextRecordIsWrittenInAFixedOrder(t *testing.T) {
	held := Room{Format: wire.Format3v3, Battles: 3, Draft: true, Data: "abc", Protocol: 1}
	first := strings.Join(held.text(), "|")
	for range 32 {
		if again := strings.Join(held.text(), "|"); again != first {
			t.Fatalf("one room wrote two records:\n%s\n%s", first, again)
		}
	}
	// And the keys are the ones a reader expects, so a rename shows up here
	// rather than as a blank column on a screen.
	for _, key := range []string{"format=", "battles=", "draft=", "watch=", "data=", "proto="} {
		if !strings.Contains(first, key) {
			t.Errorf("the record carries no %s: %s", key, first)
		}
	}
}

// TestABrowseListIsInCodeOrder.
//
// A browse hears whatever the network delivered in whatever order it arrived,
// and that order is incidental. A listing whose rows moved between two browses of
// one unchanged LAN is a listing nobody can point at.
func TestABrowseListIsInCodeOrder(t *testing.T) {
	rooms := []Room{{Code: "CCCC"}, {Code: "AAAA"}, {Code: "BBBB"}}
	sortRooms(rooms)
	for i, want := range []wire.RoomCode{"AAAA", "BBBB", "CCCC"} {
		if rooms[i].Code != want {
			t.Errorf("row %d is %s, want %s", i, rooms[i].Code, want)
		}
	}
}

// TestAdvertiseRefusesWhatItCannotAnnounce is checked before anything touches
// the network, and both arms are a record that would exist and mean nothing.
//
// The code is the DNS-SD instance name, so a room without one registers a service
// nobody can name and nobody can join. The address is the SRV record's whole
// content, so a room without one publishes a service that resolves to nowhere —
// which is worse than not publishing, because it is a row on somebody's screen.
func TestAdvertiseRefusesWhatItCannotAnnounce(t *testing.T) {
	good := aRoomCode(t, "192.168.1.5:13579", 1)
	at := netip.MustParseAddrPort("192.168.1.5:13579")
	if _, err := Advertise(Room{Code: "not a code"}, at); err == nil {
		t.Error("a room with no usable code was advertised")
	}
	if _, err := Advertise(Room{Code: good}, netip.AddrPort{}); err == nil {
		t.Error("a room with no address was advertised")
	}
}

// TestTheHostLabelCarriesNoDomain is the trim, and it is asserted because the
// bug it fixes is invisible from inside this game.
//
// zeroconf appends the domain to whatever host name it is handed, so a machine
// answering `vukynMac.local` produced the SRV target `vukynMac.local.local.` —
// measured with `dns-sd -L`. Browse never reads that field, so every test here
// passed and every other DNS-SD tool on the network was handed a name it could
// not follow.
func TestTheHostLabelCarriesNoDomain(t *testing.T) {
	label, err := hostLabel()
	if err != nil {
		t.Fatalf("read this machine's label: %v", err)
	}
	if label == "" {
		t.Fatal("this machine has no label")
	}
	if strings.Contains(label, ".") {
		t.Errorf("the label is %q and a label has no dot in it: the library appends the "+
			"domain, so this would register as %q", label, label+"."+Domain)
	}
}
