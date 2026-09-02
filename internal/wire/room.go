package wire

import (
	"encoding/base32"
	"fmt"
	"net/netip"
	"strings"
)

// RoomCodeLength is how many characters a room code is: always ten, never nine
// or eleven.
//
// Six bytes — four of address and two of port — is forty-eight bits, and base32
// spends five bits a character, so ten characters carry exactly fifty. The two
// spare bits are why the padding is off: with padding the encoder would write
// sixteen characters, six of them '='.
const RoomCodeLength = 10

// roomCodes is RFC 4648 base32 with the padding off.
//
// ⚠️ The alphabet is worth knowing before anybody proposes a friendlier one:
// A–Z and 2–7. It already excludes 0, 1, 8 and 9 — which is most of what people
// mishear reading a code out loud, and all of what they confuse with O, I and B
// — so the "confusable characters" problem this format would otherwise have was
// solved by choosing the standard encoding rather than by inventing an alphabet
// nothing else can decode.
var roomCodes = base32.StdEncoding.WithPadding(base32.NoPadding)

// RoomCode is the ten characters a player pastes to join a room, and it **carries
// its own address**: base32 of a four-byte address and a two-byte port. So
// pasting the code is enough to connect and nobody has to read an IP down a
// phone.
//
// It is not a secret and not a name. It is an address in a form a person can
// retype, which is the whole of it — the password beside it is what keeps
// strangers in the house off the board, and even that is not security.
type RoomCode string

// EncodeRoom is the code for an address.
//
// It refuses anything that is not IPv4. That is a real limit rather than an
// oversight: sixteen bytes of IPv6 plus a port is base32 of eighteen bytes,
// which is twenty-nine characters and no longer a code anybody retypes. The
// premise is one LAN, and a LAN hands out v4. An address that arrived as a
// v4-mapped v6 (::ffff:a.b.c.d) is unwrapped rather than refused, because that
// is what a listener commonly reports for a perfectly ordinary v4 socket.
func EncodeRoom(at netip.AddrPort) (RoomCode, error) {
	address := at.Addr().Unmap()
	if !address.Is4() {
		return "", fmt.Errorf("room code: %s is not an IPv4 address", at)
	}
	octets := address.As4()
	// #nosec G115 -- a port is a uint16 and the two bytes below are its halves,
	// so the narrowing is the encoding rather than a loss.
	raw := []byte{octets[0], octets[1], octets[2], octets[3], byte(at.Port() >> 8), byte(at.Port())}
	code := RoomCode(roomCodes.EncodeToString(raw))
	// A length that is not ten means the arithmetic above stopped being true,
	// which is worth a refusal rather than a code half the players in the room
	// cannot use.
	if len(code) != RoomCodeLength {
		return "", fmt.Errorf("room code: encoding %s gave %d characters, want %d", at, len(code), RoomCodeLength)
	}
	return code, nil
}

// AddrPort is the address a code carries.
//
// The code is upper-cased before it is decoded, so a player who typed it in
// lower case is not told their code is wrong: the alphabet is upper-case only,
// so the fold is total and loses nothing. Anything else — a length that is not
// ten, or a character outside A–Z and 2–7 — is an error, because a code that
// decoded to *some* address would send the player at whatever machine happened
// to be there.
func (c RoomCode) AddrPort() (netip.AddrPort, error) {
	if len(c) != RoomCodeLength {
		return netip.AddrPort{}, fmt.Errorf("room code %q: %d characters, want %d", string(c), len(c), RoomCodeLength)
	}
	raw, err := roomCodes.DecodeString(strings.ToUpper(string(c)))
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("room code %q: %w", string(c), err)
	}
	if len(raw) != 6 {
		return netip.AddrPort{}, fmt.Errorf("room code %q: %d bytes, want 6", string(c), len(raw))
	}
	address := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
	return netip.AddrPortFrom(address, uint16(raw[4])<<8|uint16(raw[5])), nil
}
