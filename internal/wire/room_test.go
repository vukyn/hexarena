package wire

import (
	"net/netip"
	"strings"
	"testing"
)

// TestARoomCodeRoundTripsAtTenCharacters is the design record's "a room code
// carries its own address" as a measurement, and it sweeps rather than spot
// checks: a code that came back as a different address is somebody joining a
// machine that is not the one that sent them the code, which is worse than a
// refusal.
//
// The two edges are in the table on purpose. `0.0.0.0:0` is every byte nought,
// which base32 encodes as ten A's — the one code a reader would mistake for a
// placeholder — and `255.255.255.255:65535` is every bit set, which is where a
// sign error or a byte order mistake shows up.
func TestARoomCodeRoundTripsAtTenCharacters(t *testing.T) {
	edges := []netip.AddrPort{
		netip.MustParseAddrPort("0.0.0.0:0"),
		netip.MustParseAddrPort("255.255.255.255:65535"),
		netip.MustParseAddrPort("192.168.1.42:7777"),
		netip.MustParseAddrPort("10.0.0.1:1"),
		netip.MustParseAddrPort("127.0.0.1:8080"),
	}
	swept := 0
	seen := make(map[RoomCode]netip.AddrPort)
	check := func(t *testing.T, at netip.AddrPort) {
		t.Helper()
		code, err := EncodeRoom(at)
		if err != nil {
			t.Fatalf("encode %s: %v", at, err)
		}
		if len(code) != RoomCodeLength {
			t.Fatalf("%s encoded to %q, %d characters, want %d", at, code, len(code), RoomCodeLength)
		}
		if first, taken := seen[code]; taken && first != at {
			t.Fatalf("%s and %s both encode to %q", first, at, code)
		}
		seen[code] = at
		back, err := code.AddrPort()
		if err != nil {
			t.Fatalf("decode %q (from %s): %v", code, at, err)
		}
		if back != at {
			t.Fatalf("%s round-tripped as %s through %q", at, back, code)
		}
		// A code is retyped by a person, so the case it is typed in must not
		// decide whether it works. The alphabet is upper case only, so the fold
		// is total.
		lowered, err := RoomCode(strings.ToLower(string(code))).AddrPort()
		if err != nil {
			t.Fatalf("decode %q in lower case: %v", code, err)
		}
		if lowered != at {
			t.Errorf("%q in lower case decoded to %s", code, lowered)
		}
		swept++
	}
	for _, at := range edges {
		t.Run(at.String(), func(t *testing.T) { check(t, at) })
	}
	// And a sweep wide enough that a byte in the wrong place cannot survive it:
	// every octet and both halves of the port are varied.
	t.Run("many addresses and ports", func(t *testing.T) {
		for a := range 8 {
			for port := range 8 {
				at := netip.AddrPortFrom(
					netip.AddrFrom4([4]byte{byte(a * 31), byte(a*17 + 3), byte(255 - a*29), byte(a * 7)}),
					uint16(port*8191+a),
				)
				check(t, at)
			}
		}
	})
	if swept < 64 {
		t.Fatalf("the sweep checked %d addresses, which is not enough to say a byte cannot be misplaced", swept)
	}
	t.Logf("%d addresses round-tripped, all at %d characters", swept, RoomCodeLength)
}

// TestARoomCodeRefusesWhatItCannotBe covers the two ways a typed code goes
// wrong, and both are refusals rather than *some* address: a code that decoded
// to whatever it could would send a player at whichever machine happened to be
// at that address.
//
// The invalid characters are chosen from what RFC 4648 leaves out — 0, 1, 8 and
// 9 — which is also the reason the alphabet was not replaced with a friendlier
// one: those are the digits people confuse reading a code out loud, and they are
// already absent.
func TestARoomCodeRefusesWhatItCannotBe(t *testing.T) {
	good, err := EncodeRoom(netip.MustParseAddrPort("192.168.1.42:7777"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	cases := map[string]RoomCode{
		"empty":                       "",
		"one short":                   good[:RoomCodeLength-1],
		"one long":                    good + "A",
		"a whole sentence":            "JOIN MY ROOM PLEASE",
		"a zero in it":                RoomCode("0" + string(good)[1:]),
		"a one in it":                 RoomCode("1" + string(good)[1:]),
		"an eight in it":              RoomCode(string(good)[:9] + "8"),
		"a nine in it":                RoomCode(string(good)[:9] + "9"),
		"punctuation":                 RoomCode(string(good)[:9] + "-"),
		"base32 padding":              RoomCode(string(good)[:9] + "="),
		"the right length of nothing": "          ",
	}
	for name, code := range cases {
		t.Run(name, func(t *testing.T) {
			at, err := code.AddrPort()
			if err == nil {
				t.Fatalf("%q decoded to %s with no error", string(code), at)
			}
			if at.IsValid() {
				t.Errorf("%q returned a usable-looking address alongside its error: %s", string(code), at)
			}
		})
	}
	// The confusable digits are absent from the format rather than filtered by
	// it, which is the claim the comment on roomCodes makes. Measured over the
	// codes the sweep above produced rather than asserted about the alphabet.
	for value := range 64 {
		code, err := EncodeRoom(netip.AddrPortFrom(netip.AddrFrom4([4]byte{byte(value), 0, 0, byte(value)}), uint16(value*997)))
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		if strings.ContainsAny(string(code), "0189") {
			t.Errorf("%q holds a digit the alphabet is supposed to exclude", code)
		}
	}
}

// TestARoomCodeRefusesAnAddressItCannotCarry is the stated limit rather than a
// bug: sixteen bytes of IPv6 and a port is twenty-nine base32 characters, which
// is not a code anybody retypes, and the premise of the whole design is one LAN.
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
		if code, err := EncodeRoom(at); err == nil {
			t.Errorf("%s encoded to %q", at, code)
		}
	}
	mapped := netip.AddrPortFrom(netip.MustParseAddr("::ffff:192.168.1.42"), 7777)
	code, err := EncodeRoom(mapped)
	if err != nil {
		t.Fatalf("a v4-mapped address is an ordinary v4 socket and was refused: %v", err)
	}
	back, err := code.AddrPort()
	if err != nil {
		t.Fatalf("decode %q: %v", code, err)
	}
	if want := netip.MustParseAddrPort("192.168.1.42:7777"); back != want {
		t.Errorf("a v4-mapped address came back as %s, want %s", back, want)
	}
}
