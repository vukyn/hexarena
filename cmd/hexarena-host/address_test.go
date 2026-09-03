package main

import (
	"net/netip"
	"strings"
	"testing"
)

// TestTheAddressInARoomCodeIsPickedOrRefused is the whole of the picker's
// decision, table-driven and with no network anywhere near it — the shape this
// repository already uses for the terminal's GOOS rules, and the reason pick
// takes a slice instead of calling net.Interfaces itself.
//
// Every row is a machine somebody actually has. The refusals are as much the
// design as the choices: a code carrying an address the other player cannot dial
// is twelve characters that simply do not work, with nothing on screen to say
// why, and that is strictly worse than being asked for -advertise.
func TestTheAddressInARoomCodeIsPickedOrRefused(t *testing.T) {
	address := netip.MustParseAddr
	for _, each := range []struct {
		machine    string
		candidates []netip.Addr
		want       string
		// refused is a fragment the refusal has to carry, so a test that only
		// asserted "an error happened" cannot pass on the wrong error.
		refused []string
	}{
		{
			machine:    "an ordinary laptop on a home router",
			candidates: []netip.Addr{address("192.168.1.5")},
			want:       "192.168.1.5",
		},
		{
			// ⚠️ The machine this was written on, measured: its real LAN address
			// is inside 172.16/12, the same RFC 1918 block docker's default
			// bridge sits in. This row exists to fail any future rule that
			// deprioritises that block — which is the tempting way to break the
			// docker tie and is wrong here.
			machine:    "a LAN handing out 172.16/12, which is also docker's block",
			candidates: []netip.Addr{address("172.16.32.222")},
			want:       "172.16.32.222",
		},
		{
			machine:    "an interface that never got a DHCP answer",
			candidates: []netip.Addr{address("169.254.13.7"), address("192.168.1.5")},
			want:       "192.168.1.5",
		},
		{
			machine:    "a dual-stack interface",
			candidates: []netip.Addr{address("fe80::1"), address("2001:db8::1"), address("10.0.0.7")},
			want:       "10.0.0.7",
		},
		{
			// A v4-mapped v6 is what a dual-stack listener commonly reports for
			// an ordinary v4 socket, so it is unwrapped rather than refused —
			// and unwrapping is also what stops it counting as a second address
			// beside the plain form, which would be a tie between one address
			// and itself.
			machine:    "the same address reported twice, once v4-mapped",
			candidates: []netip.Addr{address("::ffff:192.168.1.5"), address("192.168.1.5")},
			want:       "192.168.1.5",
		},
		{
			machine:    "one interface, listed twice",
			candidates: []netip.Addr{address("10.0.0.7"), address("10.0.0.7")},
			want:       "10.0.0.7",
		},
		{
			machine:    "nothing at all",
			candidates: nil,
			refused:    []string{"no addresses at all", "-advertise"},
		},
		{
			machine:    "loopback only, which is a machine with no network",
			candidates: []netip.Addr{address("127.0.0.1")},
			refused:    []string{"127.0.0.1", "could dial", "-advertise"},
		},
		{
			machine:    "a listener bound to everything",
			candidates: []netip.Addr{address("0.0.0.0")},
			refused:    []string{"0.0.0.0", "-advertise"},
		},
		{
			machine:    "IPv6 only, which a room code cannot carry",
			candidates: []netip.Addr{address("2001:db8::1")},
			refused:    []string{"2001:db8::1", "-advertise"},
		},
		{
			// ⚠️ **The tie this refuses rather than breaks.** 172.17.0.1 is
			// docker0: up, not loopback, IPv4, private — and unreachable from the
			// other player's laptop. Nothing in an address distinguishes it from
			// the 192.168.1.5 beside it, so both are named and -advertise is
			// asked for. The refusal must carry *both*, or a host has one address
			// on screen and no way to know it was the wrong one.
			machine:    "a machine running docker",
			candidates: []netip.Addr{address("172.17.0.1"), address("192.168.1.5")},
			refused:    []string{"172.17.0.1", "192.168.1.5", "-advertise", "container bridge"},
		},
		{
			machine:    "wifi and ethernet both up on different segments",
			candidates: []netip.Addr{address("192.168.1.5"), address("10.0.0.7")},
			refused:    []string{"192.168.1.5", "10.0.0.7", "-advertise"},
		},
	} {
		t.Run(each.machine, func(t *testing.T) {
			got, err := pick(each.candidates)
			if len(each.refused) > 0 {
				if err == nil {
					t.Fatalf("picked %s, and this machine has no address the other player could dial", got)
				}
				for _, wanted := range each.refused {
					if !strings.Contains(err.Error(), wanted) {
						t.Errorf("the refusal does not mention %q, so a host reading it cannot act on it: %v", wanted, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("refused a machine that has exactly one dialable address: %v", err)
			}
			if got.String() != each.want {
				t.Errorf("picked %s, want %s", got, each.want)
			}
		})
	}
}

// TestThePickerDoesNotDependOnTheOrderTheMachineListedItsAddresses holds the one
// property a table of fixed slices cannot: net.Interfaces walks in whatever order
// the operating system feels like, and neither the address that is chosen nor the
// refusal that names the ones that were not may move with it.
func TestThePickerDoesNotDependOnTheOrderTheMachineListedItsAddresses(t *testing.T) {
	address := netip.MustParseAddr
	one := []netip.Addr{address("127.0.0.1"), address("169.254.1.1"), address("10.0.0.7")}
	other := []netip.Addr{address("10.0.0.7"), address("127.0.0.1"), address("169.254.1.1")}
	first, err := pick(one)
	if err != nil {
		t.Fatalf("pick: %v", err)
	}
	second, err := pick(other)
	if err != nil {
		t.Fatalf("pick, reordered: %v", err)
	}
	if first != second {
		t.Errorf("the same machine listed in two orders picked %s and %s", first, second)
	}

	tied := []netip.Addr{address("172.17.0.1"), address("192.168.1.5")}
	reversed := []netip.Addr{address("192.168.1.5"), address("172.17.0.1")}
	_, forward := pick(tied)
	_, backward := pick(reversed)
	if forward == nil || backward == nil {
		t.Fatal("a machine with two dialable addresses was not refused")
	}
	if forward.Error() != backward.Error() {
		t.Errorf("the same tie reported two ways round reads differently:\n  %v\n  %v", forward, backward)
	}
}

// TestTheProbeAndTheWalkAgreeOnThisMachine is the one test here that touches the
// machine it runs on, and it is written so that it **can only fail on a
// disagreement**, never on an absence.
//
// ⚠️ What it cannot do is assert an address: a build machine may have no network
// at all, may be a container with only a bridge, or may have two interfaces up —
// and every one of those is a legitimate machine, so pinning a value would be
// pinning this developer's laptop into the suite. What it can do is check that
// when both gatherers answer, they answer the same thing, which is the claim the
// probe-first-walk-second ordering rests on. On the machine this was written on
// both return 172.16.32.222.
func TestTheProbeAndTheWalkAgreeOnThisMachine(t *testing.T) {
	fromProbe, probeErr := probed()
	candidates, walkErr := walked()
	if walkErr != nil {
		t.Logf("this machine would not list its interfaces (%v), so only the probe is measured here", walkErr)
	}
	fromWalk, pickErr := pick(candidates)

	switch {
	case probeErr != nil && pickErr != nil:
		t.Skipf("this machine has no address a room code could carry, which is a legitimate machine: "+
			"probe %v; walk %v", probeErr, pickErr)
	case probeErr != nil:
		t.Logf("the routing table would not answer (%v); the walk chose %s", probeErr, fromWalk)
	case pickErr != nil:
		t.Logf("the walk found no single answer (%v); the probe chose %s", pickErr, fromProbe)
	case fromProbe != fromWalk:
		t.Errorf("the two ways of asking disagree: the routing table says %s and the interface walk says %s. "+
			"That is not a bug on its own — it is what -advertise is for — but it means this machine would "+
			"hand out a different code depending on which gatherer answered", fromProbe, fromWalk)
	default:
		t.Logf("both ways of asking chose %s", fromProbe)
	}

	// Whatever the machine is, autodetect either answers or says what to do. A
	// refusal with no -advertise in it is a dead end on somebody's screen.
	chosen, how, err := autodetect()
	if err != nil {
		if !strings.Contains(err.Error(), "-advertise") {
			t.Errorf("autodetect refused without naming the flag that fixes it: %v", err)
		}
		return
	}
	if !chosen.Is4() {
		t.Errorf("autodetect chose %s, which is not an address a room code can carry", chosen)
	}
	if how == "" {
		t.Error("autodetect chose an address and did not say how, so a surprising code has nothing beside it")
	}
}

// TestAdvertiseOverrulesThePickerButNotPhysics holds the one place this binary is
// deliberately **more** permissive than pick, and the line it still will not
// cross.
//
// ⚠️ pick refuses loopback and link-local because it is choosing on the host's
// behalf; this flag exists to overrule pick, so refusing the same addresses here
// would be the escape hatch declining to be one. `-advertise 127.0.0.1` is how
// somebody tries the thing out with two clients on one machine — it works, and it
// is a real thing to want. What is still refused is an address *nothing* can
// dial, because there is no reading under which those are anything but a typo.
func TestAdvertiseOverrulesThePickerButNotPhysics(t *testing.T) {
	for _, each := range []struct{ given, because string }{
		{"nonsense", "not an address at all"},
		{"2001:db8::1", "IPv6, which twelve characters cannot carry"},
		{"0.0.0.0", "the unspecified address, which nobody dials"},
		{"224.0.0.1", "a multicast group, which no connection opens to"},
		{"255.255.255.255", "the broadcast address"},
	} {
		if _, _, err := advertising(each.given); err == nil {
			t.Errorf("-advertise %s was accepted; it is %s", each.given, each.because)
		}
	}
	for _, each := range []struct{ given, noted string }{
		{"10.1.2.3", ""},
		// Allowed, and the banner says what it means rather than the flag
		// refusing it. The note is the assertion: an accepted loopback with
		// nothing beside it would be a host wondering why nobody can join.
		{"127.0.0.1", "only this machine"},
		{"169.254.9.9", "only this segment"},
	} {
		got, how, err := advertising(each.given)
		if err != nil {
			t.Errorf("-advertise %s was refused: %v", each.given, err)
			continue
		}
		if got.String() != each.given {
			t.Errorf("-advertise %s became %s", each.given, got)
		}
		if !strings.Contains(how, "advertise") {
			t.Errorf("an address that came from the flag is described as %q, so the banner would not say the host chose it", how)
		}
		if each.noted != "" && !strings.Contains(how, each.noted) {
			t.Errorf("-advertise %s is described as %q and does not say %q, so a host would not know why nobody can join",
				each.given, how, each.noted)
		}
	}
}
