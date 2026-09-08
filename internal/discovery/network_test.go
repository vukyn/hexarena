package discovery

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/wire"
)

// mdnsGroup is where mDNS lives, and it is here so the probe below listens
// exactly where a browse does.
var mdnsGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

// multicastArrives reports whether this machine will deliver mDNS traffic to a
// Go process at all, and it is the honest half of the skip below.
//
// ⚠️ **This is not a formality on macOS.** Measured 2026-09-08 on macOS 25.6: a
// raw `net.ListenMulticastUDP` on the mDNS group binds without error, reports a
// local address, and then receives **nothing** — while Apple's own `dns-sd -B`,
// running on the same machine at the same moment, lists four services on two
// interfaces. Sending is unaffected: an advertisement made by this package is
// visible to `dns-sd -B _hexarena._tcp` immediately. It is the *inbound* half
// that is withheld, silently and with no error anywhere, because macOS gates
// local-network receive on a per-application permission that a binary built and
// run by `go test` has never been granted.
//
// So a browse that hears nothing on such a machine is not a defect in Browse,
// and a test that failed on it would be a test about the operating system's
// privacy settings. It is also not something to assert away: the probe is what
// tells the two apart, so the skip below names a measured condition rather than
// standing in for one.
func multicastArrives(t *testing.T) bool {
	t.Helper()
	conn, err := net.ListenMulticastUDP("udp4", nil, mdnsGroup)
	if err != nil {
		t.Logf("this machine will not open the mDNS group at all: %v", err)
		return false
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetReadDeadline(time.Now().Add(probeWindow)); err != nil {
		return false
	}
	buffer := make([]byte, 1500)
	_, _, err = conn.ReadFromUDP(buffer)
	return err == nil
}

// probeWindow is how long the probe waits for any mDNS packet at all.
//
// A LAN with a phone, a printer or a laptop on it is never quiet for this long —
// mDNS is a chatty protocol and something announces itself every few seconds — so
// silence over this window is a machine that is not delivering rather than a
// network with nothing to say.
const probeWindow = 3 * time.Second

// browseWindow is how long the browse under test listens.
const browseWindow = 3 * time.Second

// TestARoomAnnouncedOnThisMachineIsHeardByABrowse is the end-to-end, over the
// real network stack, with no fake in it.
//
// There is no seam to fake here worth having: the whole of what this package
// does beyond parsing a record is talk to a multicast group, so a test with the
// network taken out would be a test of the parsing that record_test.go already
// holds. The cost of that honesty is the skip above.
func TestARoomAnnouncedOnThisMachineIsHeardByABrowse(t *testing.T) {
	if testing.Short() {
		t.Skip("this one talks to the network")
	}
	if !multicastArrives(t) {
		t.Skip("no mDNS packet reached this process in " + probeWindow.String() +
			", so this machine is not delivering multicast to a Go binary — see multicastArrives")
	}
	code, err := wire.EncodeRoom(netip.MustParseAddrPort("192.168.1.5:13579"), 2)
	if err != nil {
		t.Fatalf("encode a room code: %v", err)
	}
	sent := Room{
		Code: code, Format: wire.Format3v3, Battles: 3,
		Draft: true, Watch: true, Data: "6537b5d935f7", Protocol: wire.Protocol,
	}
	advertised, err := Advertise(sent, netip.MustParseAddrPort("192.168.1.5:13579"))
	if err != nil {
		t.Fatalf("advertise: %v", err)
	}
	defer advertised.Close()

	ctx, cancel := context.WithTimeout(context.Background(), browseWindow)
	defer cancel()
	rooms, err := Browse(ctx)
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	var found *Room
	for i := range rooms {
		if rooms[i].Code == code {
			found = &rooms[i]
		}
	}
	if found == nil {
		t.Fatalf("the room this test announced (%s) is not among the %d heard", code, len(rooms))
	}
	// Every field, because the point of the record is the listing it draws and a
	// field lost between the register and the browse is a blank column.
	if found.Format != sent.Format || found.Battles != sent.Battles ||
		found.Draft != sent.Draft || found.Watch != sent.Watch ||
		found.Data != sent.Data || found.Protocol != sent.Protocol {
		t.Errorf("the room came back as %+v, want the fields of %+v", *found, sent)
	}
}
