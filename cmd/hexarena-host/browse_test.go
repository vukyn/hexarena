package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/discovery"
)

// TestTheBrowseFlagIsWhatDecidesTheAnnouncement is -watch's shape for -browse,
// and it goes in through the **flag set** for that test's reason: setting the
// struct field proves nothing about a flag that was never registered, or was
// registered against another variable.
//
// ⚠️ **It asserts on `browseAsked` rather than on `announced`**, and that is the
// only honest thing it can assert here. Whether an announcement actually goes out
// depends on the machine — a container, or a laptop whose operating system does
// not deliver multicast to this process — and a test that demanded a live mDNS
// registration would be red for a reason that is not this binary's. What is this
// binary's is whether the flag reached the decision, and that is what this reads.
//
// ⚠️ **The `-browse` arm does put a record on the real network for the length of
// the test**, and that is worth knowing rather than hiding. It is bounded on both
// ends: the address announced is `documented`, 192.0.2.7, which is TEST-NET-1 and
// reachable from nowhere, and `hosting`'s cleanup calls `stop`, which withdraws
// the record with a goodbye packet. A colleague browsing at that instant would see
// one unreachable room appear and go. The alternative — keeping the flag away from
// `open` — would leave the wiring between the flag and the announcement untested,
// which is the only thing this test is for.
func TestTheBrowseFlagIsWhatDecidesTheAnnouncement(t *testing.T) {
	for _, one := range []struct {
		name      string
		arguments []string
		want      bool
	}{
		{name: "typed", arguments: []string{"-browse"}, want: true},
		{name: "left off", arguments: nil, want: false},
	} {
		t.Run(one.name, func(t *testing.T) {
			var chosen settings
			set := flags(&chosen)
			set.SetOutput(io.Discard)
			if err := set.Parse(one.arguments); err != nil {
				t.Fatalf("parse %v: %v", one.arguments, err)
			}
			if chosen.browse != one.want {
				t.Fatalf("%v left the flag at %t, want %t", one.arguments, chosen.browse, one.want)
			}
			// An ephemeral port, for aRoom's reason: nothing in this suite may
			// fight the default port or another session for 13579.
			chosen.port = 0
			var out, errs bytes.Buffer
			held := hosting(t, chosen, documented, &out, &errs)
			if held.browseAsked != one.want {
				t.Errorf("%v parsed to browse=%t and opened a room that recorded %t: the flag "+
					"is not what decides the announcement", one.arguments, one.want, held.browseAsked)
			}
			if !one.want {
				if held.announced != nil || held.announceErr != nil {
					t.Errorf("a room nobody asked to announce ended up with announced=%v err=%v",
						held.announced != nil, held.announceErr)
				}
				return
			}
			// ⚠️ **Exactly one of the two, and this is what catches a flag that
			// reaches the struct and nothing else.** Asserting `browseAsked`
			// alone is green with the whole `if chosen.browse` block deleted —
			// measured — because that field is set beside it rather than inside
			// it. Whether the announcement *succeeds* is the machine's business
			// and is not asserted; that it was *attempted* is this binary's, and
			// an attempt leaves either an advertisement or a reason.
			switch {
			case held.announced != nil && held.announceErr != nil:
				t.Error("a room both announced itself and reported why it could not")
			case held.announced == nil && held.announceErr == nil:
				t.Error("-browse produced neither an advertisement nor a reason, so nothing " +
					"was attempted at all")
			}
		})
	}
}

// TestStoppingWithdrawsTheAdvertisement is the goodbye packet, and it is its own
// test because of when it has to be asserted.
//
// ⚠️ **A shutdown is not observable from outside the library** — it sends packets
// and returns nothing — so discovery.Advertisement.Closed is the only reading
// there is, and a mutation deleting the Close from `stop` changed no test before
// it existed. And the assertion has to come **after** the cleanup that stops the
// host, which `t.Cleanup` cannot do from inside the same test: cleanups run last
// in, first out, so one registered after `hosting`'s runs before it. The room is
// therefore hosted inside a subtest, whose cleanups have all run by the time
// `t.Run` returns.
func TestStoppingWithdrawsTheAdvertisement(t *testing.T) {
	var held *hosted
	t.Run("hosting", func(t *testing.T) {
		chosen := aRoom()
		chosen.browse = true
		var out, errs bytes.Buffer
		held = hosting(t, chosen, documented, &out, &errs)
		if held.announced == nil {
			t.Skipf("this machine could not announce (%v), so there is nothing to withdraw",
				held.announceErr)
		}
		if held.announced.Closed() {
			t.Fatal("the advertisement was already withdrawn before the host stopped")
		}
	})
	if held == nil || held.announced == nil {
		return
	}
	if !held.announced.Closed() {
		t.Error("the host stopped with its advertisement still on the network: a browser " +
			"keeps offering a room whose listener is closing")
	}
}

// TestTheBannerSeparatesAnnouncedFromAskedFor is the line, and it is three states
// rather than two because that is what a host can be in.
//
// ⚠️ **"asked for and failed" is the state that has to be drawn**, and it is the
// reason `browseAsked` is a field beside `announced` rather than a nil check. The
// symptom of a failed announcement is on the OTHER machine — an empty list — and
// an empty list is indistinguishable from nobody hosting. So the host, who is the
// one person who can see both, has to be told here.
func TestTheBannerSeparatesAnnouncedFromAskedFor(t *testing.T) {
	chosen := aRoom()
	var out, errs bytes.Buffer
	held := hosting(t, chosen, documented, &out, &errs)

	for _, one := range []struct {
		name      string
		asked     bool
		announced *discovery.Advertisement
		want      string
		refuse    string
	}{
		{name: "never asked", want: "", refuse: "browse "},
		{name: "asked and failed", asked: true, want: "ASKED FOR AND FAILED", refuse: ""},
		{
			name: "announced", asked: true, announced: &discovery.Advertisement{},
			want: "announced on the local network", refuse: "ASKED FOR AND FAILED",
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			held.browseAsked, held.announced = one.asked, one.announced
			var banners bytes.Buffer
			banner(held, "was told by -advertise", &banners)
			drawn := banners.String()
			if one.want != "" && !strings.Contains(drawn, one.want) {
				t.Errorf("the banner does not say %q:\n%s", one.want, drawn)
			}
			if one.refuse != "" && strings.Contains(drawn, one.refuse) {
				t.Errorf("the banner says %q when it should not:\n%s", one.refuse, drawn)
			}
		})
	}
}

// TestTheBrowseFlagIsInTheUsageAndInTheSettingsLine.
//
// A flag missing from the usage is a flag nobody finds, and one missing from
// settings.String is a flag missing from every failure this binary reports its
// own configuration in. Neither is checked by anything else here, and each is one
// line.
func TestTheBrowseFlagIsInTheUsageAndInTheSettingsLine(t *testing.T) {
	var chosen settings
	set := flags(&chosen)
	var usage bytes.Buffer
	set.SetOutput(&usage)
	set.Usage()
	if !strings.Contains(usage.String(), "-browse") {
		t.Errorf("the usage never mentions -browse:\n%s", usage.String())
	}
	chosen.browse = true
	if !strings.Contains(chosen.String(), "browse true") {
		t.Errorf("the settings line does not carry the flag: %s", chosen.String())
	}
}
