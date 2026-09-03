package wire

import (
	"bytes"
	"net/netip"
	"strings"
	"testing"
)

// theBase32Alphabet is RFC 4648's, written out because the variant sweep below
// has to try every character the last position can hold. → roomCodes for why it
// was chosen rather than replaced with a friendlier one.
const theBase32Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

// TestARoomCodeRoundTripsAtTwelveCharacters is the design record's "a room code
// carries its own address **and its room**" as a measurement, and it sweeps
// rather than spot checks: a code that came back as a different address is
// somebody joining a machine that is not the one that sent them the code, and a
// code that came back as a different room is somebody joining the wrong board on
// the right machine — both worse than a refusal.
//
// The two edges are in the table on purpose. `0.0.0.0:0` room 0 is every byte
// nought, which base32 encodes as twelve A's — the one code a reader would
// mistake for a placeholder — and `255.255.255.255:65535` room 255 is every bit
// set, which is where a sign error or a byte order mistake shows up. The room
// byte is at the **end** of the seven, so it is also the half of the round trip
// a truncation would eat first.
func TestARoomCodeRoundTripsAtTwelveCharacters(t *testing.T) {
	type opened struct {
		at   netip.AddrPort
		room uint8
	}
	edges := []opened{
		{netip.MustParseAddrPort("0.0.0.0:0"), 0},
		{netip.MustParseAddrPort("255.255.255.255:65535"), 255},
		{netip.MustParseAddrPort("192.168.1.42:7777"), 0},
		{netip.MustParseAddrPort("192.168.1.42:7777"), 1},
		{netip.MustParseAddrPort("10.0.0.1:1"), 255},
		{netip.MustParseAddrPort("127.0.0.1:8080"), 128},
	}
	swept := 0
	seen := make(map[RoomCode]opened)
	check := func(t *testing.T, one opened) {
		t.Helper()
		code, err := EncodeRoom(one.at, one.room)
		if err != nil {
			t.Fatalf("encode %s room %d: %v", one.at, one.room, err)
		}
		if len(code) != RoomCodeLength {
			t.Fatalf("%s room %d encoded to %q, %d characters, want %d", one.at, one.room, code, len(code), RoomCodeLength)
		}
		if first, taken := seen[code]; taken && first != one {
			t.Fatalf("%s room %d and %s room %d both encode to %q", first.at, first.room, one.at, one.room, code)
		}
		seen[code] = one
		back, room, err := code.Decode()
		if err != nil {
			t.Fatalf("decode %q (from %s room %d): %v", code, one.at, one.room, err)
		}
		if back != one.at || room != one.room {
			t.Fatalf("%s room %d round-tripped as %s room %d through %q", one.at, one.room, back, room, code)
		}
		// The address half on its own, for the caller that is about to connect.
		if only, err := code.AddrPort(); err != nil || only != one.at {
			t.Fatalf("%q gave the address %s (%v), want %s", code, only, err, one.at)
		}
		// A code is retyped by a person, so the case it is typed in must not
		// decide whether it works. The alphabet is upper case only, so the fold
		// is total.
		lowered, loweredRoom, err := RoomCode(strings.ToLower(string(code))).Decode()
		if err != nil {
			t.Fatalf("decode %q in lower case: %v", code, err)
		}
		if lowered != one.at || loweredRoom != one.room {
			t.Errorf("%q in lower case decoded to %s room %d", code, lowered, loweredRoom)
		}
		swept++
	}
	for _, one := range edges {
		t.Run(one.at.String(), func(t *testing.T) { check(t, one) })
	}
	// And a sweep wide enough that a byte in the wrong place cannot survive it:
	// every octet, both halves of the port, and the room varied together.
	t.Run("many addresses, ports and rooms", func(t *testing.T) {
		for a := range 8 {
			for port := range 8 {
				for room := range 4 {
					at := netip.AddrPortFrom(
						netip.AddrFrom4([4]byte{byte(a * 31), byte(a*17 + 3), byte(255 - a*29), byte(a * 7)}),
						uint16(port*8191+a),
					)
					// #nosec G115 -- room is a loop over 0..3.
					check(t, opened{at, uint8(room*79 + a)})
				}
			}
		}
	})
	if swept < 64 {
		t.Fatalf("the sweep checked %d rooms, which is not enough to say a byte cannot be misplaced", swept)
	}
	// Distinct (address, room) pairs must be distinct codes, which is the whole
	// point of the seventh byte: without it every room behind one listener would
	// encode the same string.
	if len(seen) != swept {
		t.Errorf("%d rooms produced %d distinct codes", swept, len(seen))
	}
	t.Logf("%d rooms round-tripped as %d distinct codes, all at %d characters", swept, len(seen), RoomCodeLength)
}

// TestANonCanonicalRoomCodeIsRefused is the hole the seventh byte made four
// times worse, closed and **measured**: twelve characters carry sixty bits and
// seven bytes are fifty-six, so four bits are spare and sixteen distinct strings
// decode to any one room's bytes. encoding/base32 has no Strict() — unlike
// encoding/base64 — so it ignores those trailing bits.
//
// ⚠️ The reason this is a refusal rather than a curiosity is that the registry
// keys its map on the **string**. A joiner pasting one of the other fifteen would
// look up a key that is not in the map and be told the room is unknown while the
// room sat right there: a correct-looking refusal, which is the worst shape a bug
// has. It was already true at six bytes, where two spare bits made four variants.
//
// The variant count is measured off the encoding rather than asserted about it —
// the sweep asks base32 itself which strings it will decode to the same bytes,
// and the number is then checked against the arithmetic the doc comment claims.
func TestANonCanonicalRoomCodeIsRefused(t *testing.T) {
	spare := RoomCodeLength*5 - roomCodeBytes*8
	rooms := []struct {
		at   netip.AddrPort
		room uint8
	}{
		{netip.MustParseAddrPort("192.168.1.50:9000"), 7},
		{netip.MustParseAddrPort("0.0.0.0:0"), 0},
		{netip.MustParseAddrPort("255.255.255.255:65535"), 255},
		{netip.MustParseAddrPort("10.0.0.1:7777"), 3},
	}
	variants, refused := 0, 0
	for _, one := range rooms {
		good, err := EncodeRoom(one.at, one.room)
		if err != nil {
			t.Fatalf("encode %s room %d: %v", one.at, one.room, err)
		}
		want, err := roomCodes.DecodeString(string(good))
		if err != nil {
			t.Fatalf("decode %q with the bare encoding: %v", good, err)
		}
		sameBytes := 0
		for _, character := range theBase32Alphabet {
			candidate := RoomCode(string(good)[:RoomCodeLength-1] + string(character))
			raw, err := roomCodes.DecodeString(string(candidate))
			if err != nil || !bytes.Equal(raw, want) {
				continue
			}
			sameBytes++
			at, room, err := candidate.Decode()
			if candidate == good {
				if err != nil || at != one.at || room != one.room {
					t.Fatalf("the canonical code %q was refused or misread: %s room %d (%v)", candidate, at, room, err)
				}
				continue
			}
			if err == nil {
				t.Errorf("%q decodes to the same bytes as %q and was accepted, so two strings name one room", candidate, good)
				continue
			}
			if at.IsValid() {
				t.Errorf("%q returned a usable-looking address alongside its error: %s", candidate, at)
			}
			// The refusal names the code that does work, because a player
			// holding a variant needs to be told what to paste instead.
			if !strings.Contains(err.Error(), string(good)) {
				t.Errorf("refusing %q did not name %q as the code for that room: %v", candidate, good, err)
			}
			// ⚠️ And the case fold must not walk a variant past the check:
			// lower case is folded before the canonical form is compared.
			if _, _, err := RoomCode(strings.ToLower(string(candidate))).Decode(); err == nil {
				t.Errorf("%q in lower case was accepted, so the fold outruns the canonical check", candidate)
			}
			refused++
		}
		if sameBytes != 1<<spare {
			t.Errorf("base32 decodes %d strings to %s room %d, and %d spare bits say %d",
				sameBytes, one.at, one.room, spare, 1<<spare)
		}
		variants += sameBytes
	}
	// A sweep that found one string per room would pass whether or not the
	// refusal held, so the measurement has to say it happened.
	if refused == 0 {
		t.Fatal("no variant was found to refuse, so this test measures nothing")
	}
	t.Logf("%d spare bits: base32 decodes %d strings to these %d rooms' bytes (%d per room) and %d of them are refused",
		spare, variants, len(rooms), 1<<spare, refused)
}

// TestARoomCodeRefusesWhatItCannotBe covers the ways a typed code goes wrong,
// and all of them are refusals rather than *some* address: a code that decoded
// to whatever it could would send a player at whichever machine happened to be
// at that address.
//
// The invalid characters are chosen from what RFC 4648 leaves out — 0, 1, 8 and
// 9 — which is also the reason the alphabet was not replaced with a friendlier
// one: those are the digits people confuse reading a code out loud, and they are
// already absent.
func TestARoomCodeRefusesWhatItCannotBe(t *testing.T) {
	good, err := EncodeRoom(netip.MustParseAddrPort("192.168.1.42:7777"), 3)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	last := string(good)[:RoomCodeLength-1]
	cases := map[string]RoomCode{
		"empty":                       "",
		"one short":                   good[:RoomCodeLength-1],
		"one long":                    good + "A",
		"the old ten-character shape": good[:10],
		"a whole sentence":            "JOIN MY ROOM PLEASE",
		"a zero in it":                RoomCode("0" + string(good)[1:]),
		"a one in it":                 RoomCode("1" + string(good)[1:]),
		"an eight in it":              RoomCode(last + "8"),
		"a nine in it":                RoomCode(last + "9"),
		"punctuation":                 RoomCode(last + "-"),
		"base32 padding":              RoomCode(last + "="),
		"the right length of nothing": "            ",
	}
	for name, code := range cases {
		t.Run(name, func(t *testing.T) {
			at, room, err := code.Decode()
			if err == nil {
				t.Fatalf("%q decoded to %s room %d with no error", string(code), at, room)
			}
			if at.IsValid() {
				t.Errorf("%q returned a usable-looking address alongside its error: %s", string(code), at)
			}
			if room != 0 {
				t.Errorf("%q returned room %d alongside its error", string(code), room)
			}
		})
	}
	// The confusable digits are absent from the format rather than filtered by
	// it, which is the claim the comment on roomCodes makes. Measured over the
	// codes the sweep above produced rather than asserted about the alphabet.
	for value := range 64 {
		// #nosec G115 -- value is a loop over 0..63.
		code, err := EncodeRoom(netip.AddrPortFrom(netip.AddrFrom4([4]byte{byte(value), 0, 0, byte(value)}), uint16(value*997)), uint8(value*3))
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		if strings.ContainsAny(string(code), "0189") {
			t.Errorf("%q holds a digit the alphabet is supposed to exclude", code)
		}
	}
}

// TestARoomCodeRefusesAnAddressItCannotCarry is the stated limit rather than a
// bug: sixteen bytes of IPv6, a port and a room is thirty-one base32 characters,
// which is not a code anybody retypes, and the premise of the whole design is
// one LAN.
//
// A v4-mapped v6 address is the case that must *not* be refused — that is what a
// listener commonly reports for an ordinary v4 socket, so refusing it would
// refuse a room on the machine that opened it.
func TestARoomCodeRefusesAnAddressItCannotCarry(t *testing.T) {
	for _, at := range []netip.AddrPort{
		netip.MustParseAddrPort("[::1]:7777"),
		netip.MustParseAddrPort("[fe80::1]:7777"),
		{},
	} {
		// Every room behind it, not only room nought: an address a code cannot
		// carry cannot carry any of its rooms.
		for _, room := range []uint8{0, 1, 255} {
			if code, err := EncodeRoom(at, room); err == nil {
				t.Errorf("%s room %d encoded to %q", at, room, code)
			}
		}
	}
	mapped := netip.AddrPortFrom(netip.MustParseAddr("::ffff:192.168.1.42"), 7777)
	code, err := EncodeRoom(mapped, 9)
	if err != nil {
		t.Fatalf("a v4-mapped address is an ordinary v4 socket and was refused: %v", err)
	}
	back, room, err := code.Decode()
	if err != nil {
		t.Fatalf("decode %q: %v", code, err)
	}
	if want := netip.MustParseAddrPort("192.168.1.42:7777"); back != want || room != 9 {
		t.Errorf("a v4-mapped address came back as %s room %d, want %s room 9", back, room, want)
	}
}
