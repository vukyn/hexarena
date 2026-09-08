// Package discovery is mDNS room browsing: a host announces its open room on the
// local network, and a client lists what it hears without anybody reading a code
// out loud.
//
// # ⚠️ The dependency is confined here, and that is the package's first job
//
// This is the second boundary in the repository written to hold a third-party
// network library in one place — `internal/socket` holds the WebSocket one — and
// the rule is the same: nothing outside this package imports `zeroconf`, and
// nothing inside it leaks a `zeroconf` type through an exported signature. What
// crosses the boundary is a Room, which is this game's own vocabulary.
//
// Real mDNS rather than a private multicast beacon, chosen deliberately. A
// beacon on a group of our own would be smaller and would need no dependency at
// all, and it would also be invisible to `dns-sd -B _hexarena._tcp` and to
// `avahi-browse` — so a host who cannot see their own room in the list would have
// no way to tell whether the game or the network was at fault. Speaking the
// standard means the machine's own tooling is the second opinion.
//
// # ⚠️ A record is read leniently and written strictly
//
// Everything else in this repository parses its data with DisallowUnknownFields,
// because a field nobody reads is a field an author thinks is doing something.
// That rule is exactly inverted here and the reason is who wrote the bytes: a
// data file is authored by *this* build, and a TXT record arrives from *another
// machine's* build, which may be a version ahead. A key this build does not know
// is therefore a fact about the sender rather than a mistake, and refusing the
// record over it would make every added key a flag day on a LAN.
//
// What is NOT lenient is the code. A record with no room code in it is refused
// outright: the code is the only thing a client can act on, and an entry a player
// can see and cannot join is worse than an entry that never appeared.
package discovery

import (
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/wire"
)

// Service is the DNS-SD service type a room is announced under, and Domain is
// the only domain mDNS has.
//
// The underscore prefix and the `._tcp` suffix are DNS-SD's, not a convention of
// ours: they are what makes `dns-sd -B _hexarena._tcp` and `avahi-browse
// _hexarena._tcp` find a room without either of them being told anything about
// this game.
const (
	Service = "_hexarena._tcp"
	Domain  = "local."
)

// Room is one announcement: what a host publishes, and what a browse hands back.
//
// ⚠️ **The code is the whole of what a client acts on.** Everything else on this
// struct is for the listing to draw — a player picks a row and the join that
// follows is the ordinary one, pasting the ordinary code, over the ordinary
// path. Browsing adds a way to *find* a room and no way to *enter* one, which is
// what keeps this package out of the join's way entirely.
type Room struct {
	// Code is the twelve characters a player would otherwise have been read out.
	// It already carries the address, the port and the room index, so a listing
	// needs nothing else to act.
	Code wire.RoomCode
	// Format, Battles, Draft and Watch are what the host's own banner prints, so
	// a player can tell a drafting bo3 from a squad bo1 before joining one.
	Format  wire.Format
	Battles int
	Draft   bool
	Watch   bool
	// Data is the host's short data digest and Protocol its protocol number.
	// Both are what the join screen already compares, and both are here so a
	// listing can say "this one will refuse you" before the dial rather than
	// after it.
	Data     string
	Protocol int
	// At is what mDNS itself reported, and it is set by a browse and ignored by
	// an advertisement.
	//
	// ⚠️ It is NOT what a client dials — the code is — and the two can honestly
	// differ: a host picks one address for the code (see cmd/hexarena-host's
	// `pick`, which refuses to guess between a LAN address and a container
	// bridge) while mDNS answers with every address the machine has. It is kept
	// so a listing can show a room whose code names an address the browser can
	// see is not the one the packet came from, which is the shape of every
	// "the code does not work" report this design can produce.
	At netip.AddrPort
}

// text is a Room as DNS-SD TXT strings, one `key=value` each.
//
// Written in a fixed order rather than by ranging anything, for the reason every
// output in this repository is: two hosts advertising the same room must produce
// the same bytes, or a diff of two captures says something moved when nothing
// did.
//
// The code is not here. It is the DNS-SD **instance name** — see Advertise — so
// a browser that loses the TXT record entirely still has the one field that
// matters.
func (r Room) text() []string {
	return []string{
		"format=" + strconv.Itoa(int(r.Format)),
		"battles=" + strconv.Itoa(r.Battles),
		"draft=" + flag(r.Draft),
		"watch=" + flag(r.Watch),
		"data=" + r.Data,
		"proto=" + strconv.Itoa(r.Protocol),
	}
}

func flag(set bool) string {
	if set {
		return "1"
	}
	return "0"
}

// parseRoom is one browse result as a Room: the instance name is the code, and
// the TXT strings fill in the rest.
//
// ⚠️ **Only the code can fail.** Every other field falls back to its zero value,
// because a record from a build one version ahead may word a value in a way this
// one cannot read, and dropping a joinable room over a field a listing merely
// draws would be the lenient rule (see the package comment) thrown away at the
// one site that implements it.
func parseRoom(instance string, text []string, at netip.AddrPort) (Room, error) {
	code := wire.RoomCode(strings.TrimSpace(instance))
	if _, _, err := code.Decode(); err != nil {
		return Room{}, fmt.Errorf("a room announced itself as %q, which is no room code: %w", instance, err)
	}
	held := Room{Code: code, At: at}
	for _, entry := range text {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		switch key {
		case "format":
			if number, err := strconv.Atoi(value); err == nil {
				held.Format = wire.Format(number)
			}
		case "battles":
			if number, err := strconv.Atoi(value); err == nil {
				held.Battles = number
			}
		case "draft":
			held.Draft = value == "1"
		case "watch":
			held.Watch = value == "1"
		case "data":
			held.Data = value
		case "proto":
			if number, err := strconv.Atoi(value); err == nil {
				held.Protocol = number
			}
		}
	}
	return held, nil
}

// sortRooms puts a listing in code order.
//
// A browse hears whatever arrives in whatever order the network delivered it,
// and that order is incidental — the same rule `internal/core` states for a map
// walk, one layer out. A listing whose rows moved between two browses of the
// same LAN would be a listing nobody could point at.
func sortRooms(rooms []Room) {
	sort.Slice(rooms, func(a, b int) bool { return rooms[a].Code < rooms[b].Code })
}
