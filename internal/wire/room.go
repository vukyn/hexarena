package wire

import (
	"encoding/base32"
	"fmt"
	"net/netip"
	"strings"
)

// RoomCodeLength is how many characters a room code is: always twelve, never
// eleven or thirteen.
//
// Seven bytes — four of address, two of port and one of **room** — is fifty-six
// bits, and base32 spends five bits a character, so twelve characters carry
// exactly sixty. The four spare bits are why the padding is off: with padding
// the encoder would write sixteen characters, four of them '='.
//
// ⚠️ **It was ten, and the ten-character claim is retired.** A code carried six
// bytes while the open reading was one listener *per room*; one process now runs
// many rooms behind **one** listener, so the address and the port name the
// process and the seventh byte names the room. → EncodeRoom for why that is the
// cheaper answer, and README.md § *A room, and getting into one*.
//
// ⚠️ Two spare bits became four, so the number of strings base32 will decode to
// one room's bytes went from four to sixteen. That is not a curiosity: a code is
// a **map key** in the registry. → Decode, which refuses every variant but one.
const RoomCodeLength = 12

// roomCodeBytes is what a code decodes to: four of address, two of port, one of
// room. It is named because the encoder and the decoder have to agree about it,
// and a literal 7 in each of them is where they would stop agreeing.
const roomCodeBytes = 7

// RoomsPerProcess is how many rooms behind one address a code can tell apart:
// the room is one byte, so 256. The bound is the **width of the field** rather
// than a policy anybody picked, and it is far past what a LAN wants.
const RoomsPerProcess = 256

// roomCodes is RFC 4648 base32 with the padding off.
//
// ⚠️ The alphabet is worth knowing before anybody proposes a friendlier one:
// A–Z and 2–7. It already excludes 0, 1, 8 and 9 — which is most of what people
// mishear reading a code out loud, and all of what they confuse with O, I and B
// — so the "confusable characters" problem this format would otherwise have was
// solved by choosing the standard encoding rather than by inventing an alphabet
// nothing else can decode.
var roomCodes = base32.StdEncoding.WithPadding(base32.NoPadding)

// RoomCode is the twelve characters a player pastes to join a room, and it
// **carries both the address and the room**: base32 of a four-byte address, a
// two-byte port and a one-byte room. So pasting the code is enough to connect
// *and* enough to say which of the rooms behind that address is meant, and
// nobody has to read an IP down a phone.
//
// It is not a secret and not a name. It is an address in a form a person can
// retype, which is the whole of it — the password beside it is what keeps
// strangers in the house off the board, and even that is not security.
type RoomCode string

// EncodeRoom is the code for one room behind one address.
//
// It refuses anything that is not IPv4. That is a real limit rather than an
// oversight: sixteen bytes of IPv6 plus a port and a room is base32 of nineteen
// bytes, which is thirty-one characters and no longer a code anybody retypes.
// The premise is one LAN, and a LAN hands out v4. An address that arrived as a
// v4-mapped v6 (::ffff:a.b.c.d) is unwrapped rather than refused, because that
// is what a listener commonly reports for a perfectly ordinary v4 socket.
//
// # Why the room is a byte of the code, rather than a port per room
//
// One process, many rooms, **one listener**. The address and the port therefore
// name the *process*, and without the seventh byte a code would name the process
// rather than the room — pasting one would be a coin toss between the rooms
// running in it. The other way out was a listener per room, and it is the more
// expensive answer for reasons that have nothing to do with the wire: a port is
// a finite OS resource that wants a firewall hole, one leaks per crashed room,
// and it conflates a **room** (an application idea) with a **listener** (an OS
// one), so a registry keyed by code would be shadowed by a second registry keyed
// by port — and socket lifetime would become room lifetime, in the one component
// that has no I/O precisely so that it can be tested. A port per "room" is what
// appears where a room is a whole *process* (Quake, CS, an Agones fleet), which
// is the opposite architecture to this one. What it cost is written down: ten
// characters became twelve. → README.md § *A room, and getting into one*.
//
// The room is a uint8 because that **is** the bound: there is no out-of-range
// room to refuse, which is a check that cannot be forgotten rather than one
// written correctly.
func EncodeRoom(at netip.AddrPort, room uint8) (RoomCode, error) {
	address := at.Addr().Unmap()
	if !address.Is4() {
		return "", fmt.Errorf("room code: %s is not an IPv4 address", at)
	}
	octets := address.As4()
	// #nosec G115 -- a port is a uint16 and two of the bytes below are its
	// halves, so the narrowing is the encoding rather than a loss.
	raw := []byte{octets[0], octets[1], octets[2], octets[3], byte(at.Port() >> 8), byte(at.Port()), room}
	code := RoomCode(roomCodes.EncodeToString(raw))
	// A length that is not twelve means the arithmetic above stopped being true,
	// which is worth a refusal rather than a code half the players in the room
	// cannot use.
	if len(code) != RoomCodeLength {
		return "", fmt.Errorf("room code: encoding %s room %d gave %d characters, want %d", at, room, len(code), RoomCodeLength)
	}
	return code, nil
}

// Decode is everything a code carries: the address to connect to, and which of
// the rooms behind that address is meant.
//
// The code is upper-cased before it is decoded, so a player who typed it in
// lower case is not told their code is wrong: the alphabet is upper-case only,
// so the fold is total and loses nothing. Anything else — a length that is not
// twelve, or a character outside A–Z and 2–7 — is an error, because a code that
// decoded to *some* address would send the player at whatever machine happened
// to be there.
//
// # ⚠️ A non-canonical code is refused, and that is about a map key
//
// Twelve characters carry sixty bits and seven bytes are fifty-six, so **four
// bits are spare** and sixteen distinct strings decode to any one room's bytes.
// encoding/base32 has no Strict() — unlike encoding/base64 — so it simply
// ignores the trailing bits, and this was a hole at six bytes too, where two
// spare bits made four variants.
//
// It matters because the registry keys its map on the **string**. A joiner who
// pasted one of the other fifteen would look up a key that is not in the map and
// be told the room is unknown while the room sat right there — the worst shape a
// bug can have, a correct-looking refusal. So a code is decoded, re-encoded, and
// refused when the two differ: a variant becomes a clear refusal that names the
// code which does work, and the canonical string is a sound key.
func (c RoomCode) Decode() (netip.AddrPort, uint8, error) {
	if len(c) != RoomCodeLength {
		return netip.AddrPort{}, 0, fmt.Errorf("room code %q: %d characters, want %d", string(c), len(c), RoomCodeLength)
	}
	// The case fold is total, so the canonical form is compared against the
	// folded string rather than against what was typed.
	written := strings.ToUpper(string(c))
	raw, err := roomCodes.DecodeString(written)
	if err != nil {
		return netip.AddrPort{}, 0, fmt.Errorf("room code %q: %w", string(c), err)
	}
	if len(raw) != roomCodeBytes {
		return netip.AddrPort{}, 0, fmt.Errorf("room code %q: %d bytes, want %d", string(c), len(raw), roomCodeBytes)
	}
	if canonical := roomCodes.EncodeToString(raw); canonical != written {
		return netip.AddrPort{}, 0, fmt.Errorf("room code %q is not the code for the room it names; %q is", string(c), canonical)
	}
	address := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
	return netip.AddrPortFrom(address, uint16(raw[4])<<8|uint16(raw[5])), raw[6], nil
}

// AddrPort is the address half of a code, for the caller that is about to
// connect and does not care yet which of the rooms behind it is meant. Every
// refusal is Decode's — this adds none of its own and drops none.
func (c RoomCode) AddrPort() (netip.AddrPort, error) {
	at, _, err := c.Decode()
	return at, err
}
