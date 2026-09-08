package discovery

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"strings"

	"github.com/libp2p/zeroconf/v2"

	"github.com/vukyn/hexarena/internal/wire"
)

// Advertisement is a room announced on the local network, and closing it takes
// the announcement back.
//
// ⚠️ **Closing is not optional and is not only tidiness.** zeroconf sends
// goodbye packets on shutdown — records with a zero TTL — and a browser that
// never receives them keeps offering a room that is gone until the TTL expires.
// A player picking that row pastes a code at nothing. So the host closes this
// before it exits, on every path, the same way it closes its listener.
type Advertisement struct {
	server *zeroconf.Server
	// closed is whether the withdrawal has happened, and it is here so a caller
	// can be tested for doing it.
	//
	// ⚠️ A shutdown is not observable from outside the library — it sends packets
	// and returns nothing — so without this flag, "the host withdraws its
	// advertisement when it stops" is a sentence no test can hold. Measured: a
	// mutation deleting the Close from the host's stop path changed no test.
	closed bool
}

// Closed reports whether the withdrawal has happened. A nil advertisement — a
// room that never announced — reads as closed, because there is nothing left to
// take back.
func (a *Advertisement) Closed() bool { return a == nil || a.closed }

// Advertise publishes one room under the DNS-SD service type.
//
// ⚠️ **The room code is the INSTANCE NAME**, which is two decisions at once.
// DNS-SD requires an instance name to be unique on the network, and a room code
// already is by construction: it encodes the host's address, the port and the
// room's index, so two rooms cannot share one unless they are the same room. And
// a browser that receives the PTR and SRV records but loses the TXT still has the
// one field a client can act on. Naming the instance after the player, or after
// the machine, would have needed a uniqueness rule of its own and would have put
// the code somewhere it could go missing.
//
// ⚠️ **It registers as a PROXY for one address rather than letting the library
// advertise the machine**, and both halves of that are deliberate.
//
// The address: zeroconf's own Register publishes *every* address on *every*
// interface, which is precisely the ambiguity cmd/hexarena-host's `pick` refuses
// to guess about — beside a real LAN address, a container bridge is up, is IPv4
// and is unreachable from the other player's laptop. The host has already made
// that decision once, and the answer is in the room code. Publishing the same
// address the code carries means the SRV record and the code cannot disagree.
//
// The host name: `os.Hostname()` on macOS answers `vukynMac.local`, and zeroconf
// appends the domain to whatever it is given — measured, the SRV target came out
// `vukynMac.local.local.`, which resolves nowhere. `dns-sd -L` shows it and our
// own Browse never reads it, so the game worked and every other DNS-SD tool on
// the network was handed a name it could not follow. hostLabel is the trim.
func Advertise(held Room, at netip.AddrPort) (*Advertisement, error) {
	if _, _, err := held.Code.Decode(); err != nil {
		return nil, fmt.Errorf("advertise a room: %w", err)
	}
	// ⚠️ **Not the only thing that refuses this, and the only one that says
	// why.** RegisterProxy fails on an empty address list of its own accord —
	// measured, a mutation deleting these three lines leaves the refusal test
	// green — but its message is about a DNS record, and the caller's mistake is
	// that it had no address to announce. Same shape as every guard in this
	// repository that duplicates a library's refusal: kept for the sentence.
	if !at.IsValid() {
		return nil, fmt.Errorf("advertise room %s: %s is not an address to announce", held.Code, at)
	}
	name, err := hostLabel()
	if err != nil {
		return nil, fmt.Errorf("advertise room %s: %w", held.Code, err)
	}
	server, err := zeroconf.RegisterProxy(string(held.Code), Service, Domain, int(at.Port()),
		name, []string{at.Addr().Unmap().String()}, held.text(), nil)
	if err != nil {
		return nil, fmt.Errorf("announce room %s on the network: %w", held.Code, err)
	}
	return &Advertisement{server: server}, nil
}

// hostLabel is this machine's name with any domain taken off it, which is the
// form a DNS-SD registration wants: the library adds the domain itself.
//
// ⚠️ **It trims at the first dot rather than trimming a `.local` suffix**, and
// the difference is a machine on a corporate network answering
// `laptop.corp.example.com`. Stripping only `.local` would leave that whole name
// to have `.local.` appended to it. The label is what the SRV record needs and a
// domain is never part of one.
//
// An empty name is refused rather than defaulted. A registration with no host to
// point at is a service record naming nothing, and a machine that cannot say what
// it is called is a machine something else is wrong with.
func hostLabel() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("read this machine's name: %w", err)
	}
	label, _, _ := strings.Cut(strings.TrimSpace(name), ".")
	if label == "" {
		return "", fmt.Errorf("this machine reported %q as its name, which has no label in it", name)
	}
	return label, nil
}

// Close withdraws the announcement. It is safe on a nil receiver, so a caller
// that never managed to advertise can defer it unconditionally.
func (a *Advertisement) Close() {
	if a == nil {
		return
	}
	if a.server != nil {
		a.server.Shutdown()
	}
	a.closed = true
}

// Browse listens for rooms until the context is done and hands back what it
// heard, one entry per room code, in code order.
//
// ⚠️ **It listens for a fixed stretch rather than returning on the first
// answer**, and the caller's context is what sets that stretch. mDNS is a
// question shouted at a network: an answer may arrive in a millisecond or in
// half a second, and there is no moment at which "everybody has replied" is a
// fact. So a browse is a *window*, and a listing drawn from it is what was heard
// in that window rather than what exists.
//
// A record that cannot be read is skipped rather than failing the browse. One
// misbehaving announcer on the LAN — or one from a build that words a code
// differently — must not empty a list that has three good rooms in it.
func Browse(ctx context.Context) ([]Room, error) {
	entries := make(chan *zeroconf.ServiceEntry, browseBuffer)
	heard := map[wire.RoomCode]Room{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for entry := range entries {
			held, err := parseRoom(entry.Instance, entry.Text, addressOf(entry))
			if err != nil {
				continue
			}
			// Last one wins. A host re-announces while its room is open, so a
			// later record is a fresher reading of the same room rather than a
			// second room.
			heard[held.Code] = held
		}
	}()
	if err := zeroconf.Browse(ctx, Service, Domain, entries); err != nil {
		return nil, fmt.Errorf("browse the network for rooms: %w", err)
	}
	<-ctx.Done()
	<-done
	rooms := make([]Room, 0, len(heard))
	for _, held := range heard {
		rooms = append(rooms, held)
	}
	sortRooms(rooms)
	return rooms, nil
}

// browseBuffer is how many entries the library may hand over before this package
// has read the last one.
//
// ⚠️ **An unbuffered channel here deadlocks a browse**, and the shape is worth
// naming because it is not obvious from the call: zeroconf writes entries from
// its own goroutine and closes the channel when the context ends, so a reader
// that is not already waiting stalls the library rather than itself. The buffer
// is not a performance choice; it is slack for a burst of answers arriving in
// one instant, which is exactly what a multicast question produces.
const browseBuffer = 32

// addressOf is the address a record arrived from, preferring IPv4 for the reason
// wire.EncodeRoom is IPv4 only: this game's premise is one LAN, and a LAN hands
// out v4.
//
// It is a reading rather than a dialling address — see Room.At — so an entry
// with no usable address at all is a zero value rather than a refusal.
func addressOf(entry *zeroconf.ServiceEntry) netip.AddrPort {
	for _, held := range entry.AddrIPv4 {
		if address, ok := netip.AddrFromSlice(held); ok {
			// #nosec G115 -- a DNS SRV port is a uint16 and the field is an int
			// holding one.
			return netip.AddrPortFrom(address.Unmap(), uint16(entry.Port))
		}
	}
	for _, held := range entry.AddrIPv6 {
		if address, ok := netip.AddrFromSlice(held); ok {
			// #nosec G115 -- as above.
			return netip.AddrPortFrom(address.Unmap(), uint16(entry.Port))
		}
	}
	return netip.AddrPort{}
}
