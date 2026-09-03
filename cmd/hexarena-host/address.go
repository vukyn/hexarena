package main

import (
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
)

// probeTarget is the address the route probe pretends it is about to talk to.
//
// It is **192.0.2.1**, which is TEST-NET-1 (RFC 5737): a block reserved for
// documentation that no real host may use, so nothing can be there and nothing
// can be disturbed by naming it. Port 9 is discard.
//
// ⚠️ **No packet is sent.** net.Dial on a UDP socket does not talk to anybody —
// it asks the kernel to *connect* the socket, which means picking the route and
// binding a local address, and then the local address is read back off it. That
// is why this answers the question it does: not "who can I reach" but "which of
// my own addresses would this machine use to leave itself".
const probeTarget = "192.0.2.1:9"

// pick is the address that goes in the room code, chosen from whatever the
// machine reported. It is **pure** — a slice in, an address or a refusal out —
// so the decision is table-testable without a network, which is the shape this
// repository already uses for the terminal's GOOS rules.
//
// # What it refuses, and why each one is a refusal rather than a fallback
//
// A room code carries four address bytes, so it is IPv4 only, and the address in
// it has to be one **the other machine can dial**. That rules out more than it
// first looks like:
//
//   - **Loopback.** 127.0.0.1 encodes into a perfectly valid code that works on
//     the host's own machine and nowhere else, which is the worst kind of wrong:
//     the host can test it and the guest cannot use it.
//   - **The unspecified address.** 0.0.0.0 is what a listener on every interface
//     reports, and it is not an address anybody dials.
//   - **Link-local, 169.254.0.0/16.** An address a machine gave itself because
//     DHCP did not answer. Two machines on one segment can sometimes reach each
//     other on these, and "sometimes" is not what a code a person retypes should
//     rest on.
//   - **Multicast and the broadcast address.** Not somewhere a TCP connection
//     goes.
//   - **IPv6.** Sixteen bytes plus a port and a room is thirty-one base32
//     characters and no longer a code anybody retypes. → wire.EncodeRoom, where
//     the limit is argued. A v4-mapped v6 (::ffff:a.b.c.d) is unwrapped rather
//     than refused, because that is what a dual-stack listener commonly reports
//     for an ordinary v4 socket.
//
// # ⚠️ More than one survivor is an ERROR, and that is the decision
//
// This is the sharp part and it is a refusal on purpose. On a machine running
// docker, `docker0` is **172.17.0.1**: it is up, it is not loopback, it is IPv4,
// `IsPrivate` says yes — and it is unreachable from the other player's laptop.
// Beside a real `192.168.1.5` there are two survivors and **nothing in an address
// distinguishes them**. The rules that suggest themselves were tried and all of
// them are wrong:
//
//   - **"Prefer 192.168/16 over 172.16/12"** is the tempting one and it is
//     wrong on the machine this was written on, whose real LAN address is
//     **172.16.32.222** — inside 172.16/12, the same RFC 1918 block docker's
//     default bridge sits in. A rule that deprioritised that block would refuse
//     the correct answer here.
//   - **"Prefer the lowest"** or **"prefer the first"** is not a principle, it is
//     a coin toss with a tidy implementation.
//   - **The interface name** (`docker0`, `br-…`, `veth…`, `utun…`) genuinely does
//     distinguish them — and it is not an address, so it cannot be read here, and
//     a list of name prefixes is a guess about somebody else's naming that goes
//     stale the first time a container runtime changes.
//
// So the tie is not broken: it is **reported**, naming every candidate and
// asking for -advertise. Guessing wrong prints a twelve-character code that
// simply does not work, with nothing on screen to explain why — refusing prints
// the addresses and the flag that fixes it. And the cost of the refusal is small,
// because this function is the *fallback*: the route probe answers first and
// answers correctly on any machine with a default route.
func pick(candidates []netip.Addr) (netip.Addr, error) {
	if len(candidates) == 0 {
		return netip.Addr{}, fmt.Errorf(
			"this machine reported no addresses at all, so there is nothing to put in a room code; " +
				"pass -advertise with the IPv4 address the other player should connect to")
	}
	usable := make([]netip.Addr, 0, len(candidates))
	for _, candidate := range candidates {
		address := candidate.Unmap()
		if !address.Is4() || !dialable(address) {
			continue
		}
		// Two interfaces reporting one address is one address, not a tie. A
		// dual-stack listener and a v4-mapped duplicate reach here as the same
		// value because of the Unmap above.
		if !slices.Contains(usable, address) {
			usable = append(usable, address)
		}
	}
	// Sorted, so that what this function returns and what its refusal names do
	// not depend on the order net.Interfaces happened to walk. The engine's rule
	// one layer out: nothing whose order is incidental may reach an output.
	slices.SortFunc(usable, netip.Addr.Compare)
	switch len(usable) {
	case 1:
		return usable[0], nil
	case 0:
		return netip.Addr{}, fmt.Errorf(
			"none of this machine's %d address(es) is one another machine could dial (%s); "+
				"pass -advertise with the IPv4 address the other player should connect to",
			len(candidates), listed(candidates))
	}
	return netip.Addr{}, fmt.Errorf(
		"this machine has %d addresses another machine could dial (%s) and nothing about an address says "+
			"which of them the other player is on — a container bridge looks exactly like a LAN; "+
			"pass -advertise with the one they should connect to",
		len(usable), listed(usable))
}

// dialable reports whether an IPv4 address is one another machine could open a
// connection to. → pick, where every exclusion is argued.
func dialable(address netip.Addr) bool {
	switch {
	case address.IsUnspecified():
		return false
	case address.IsLoopback():
		return false
	case address.IsLinkLocalUnicast():
		return false
	case address.IsMulticast(), address.IsLinkLocalMulticast():
		return false
	case address == netip.AddrFrom4([4]byte{255, 255, 255, 255}):
		return false
	}
	return true
}

// listed is a run of addresses as a person would read them out, for a refusal
// that has to name what it could not choose between.
func listed(addresses []netip.Addr) string {
	written := make([]string, 0, len(addresses))
	for _, address := range addresses {
		written = append(written, address.Unmap().String())
	}
	return strings.Join(written, ", ")
}

// probed is the address this machine would leave itself by, asked of the routing
// table rather than of the interface list.
//
// ⚠️ **It fails on a machine with no default route**, which a LAN behind a bare
// switch genuinely can be, and that failure is the reason walked exists. It is
// tried first anyway because it answers the right question exactly — "which of my
// addresses would the OS pick to reach off this machine" — where the interface
// walk answers a different and looser one.
func probed() (netip.Addr, error) {
	// #nosec G102 -- no listener is opened and no packet is sent; a connected UDP
	// socket only picks a route. → probeTarget.
	conn, err := net.Dial("udp4", probeTarget)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("ask this machine which address it would leave itself by: %w", err)
	}
	defer func() { _ = conn.Close() }()
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return netip.Addr{}, fmt.Errorf("a udp4 socket reported a %T as its local address", conn.LocalAddr())
	}
	address, ok := netip.AddrFromSlice(local.IP)
	if !ok {
		return netip.Addr{}, fmt.Errorf("this machine reported %q as its own address", local.IP)
	}
	return address.Unmap(), nil
}

// walked is every address on an interface that is up and is not loopback, which
// is the fallback for a machine the probe could not answer for.
//
// It hands back candidates and judges none of them — the judging is pick's, and
// it is separate precisely because ⚠️ this list is where the ambiguity lives: a
// container bridge is up, is not loopback and is IPv4, and is unreachable from
// anywhere that matters.
func walked() ([]netip.Addr, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("read this machine's network interfaces: %w", err)
	}
	var candidates []netip.Addr
	for _, each := range interfaces {
		if each.Flags&net.FlagUp == 0 || each.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := each.Addrs()
		if err != nil {
			// One interface refusing to describe itself is not a reason to
			// abandon the others; a machine with none left is pick's refusal.
			continue
		}
		for _, held := range addresses {
			network, ok := held.(*net.IPNet)
			if !ok {
				continue
			}
			if address, ok := netip.AddrFromSlice(network.IP); ok {
				candidates = append(candidates, address.Unmap())
			}
		}
	}
	return candidates, nil
}

// autodetect is the address a room code will carry when -advertise was not
// given: the route probe, then the interface walk, and a refusal that names the
// flag if neither can answer.
//
// It also reports **how** it decided, because that goes on screen: a host whose
// code carries an address they did not expect should be able to see which of the
// two answered without reading this file.
func autodetect() (netip.Addr, string, error) {
	if found, err := probed(); err == nil {
		// Run through pick even though it is one address: a probe on a machine
		// with no route can answer with loopback, and that is a code that works
		// only where it was made.
		if chosen, err := pick([]netip.Addr{found}); err == nil {
			return chosen, "asked the routing table", nil
		}
	}
	candidates, err := walked()
	if err != nil {
		return netip.Addr{}, "", err
	}
	chosen, err := pick(candidates)
	if err != nil {
		return netip.Addr{}, "", err
	}
	return chosen, "walked this machine's interfaces", nil
}
