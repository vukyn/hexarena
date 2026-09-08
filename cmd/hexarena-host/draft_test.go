package main

import (
	"bytes"
	"net/netip"
	"strings"
	"testing"
)

// TestADraftingRoomIsOpenedAndSaysSo holds both halves, and the second one is
// why there are two.
//
// ⚠️ **A banner line that is always drawn passes the first half alone.** Every
// state this binary prints is one line in one block, so a `-draft` note wired to
// nothing — or to a constant — reads exactly like the real thing on the run that
// asked for it. What separates them is the run that did **not** ask: a host who
// never typed the flag must not be told to join with no squad, because the advice
// is wrong for their room and a banner that gives wrong advice once is a banner
// read past thereafter.
func TestADraftingRoomIsOpenedAndSaysSo(t *testing.T) {
	for _, one := range []struct {
		name   string
		draft  bool
		drafts bool
	}{
		{name: "with -draft", draft: true, drafts: true},
		{name: "without it", draft: false, drafts: false},
	} {
		t.Run(one.name, func(t *testing.T) {
			chosen := aRoom()
			chosen.draft = one.draft
			var out, errs bytes.Buffer
			held := hosting(t, chosen, "10.0.0.7", &out, &errs)
			if held.config.Drafts != one.drafts {
				t.Fatalf("-draft=%t opened a room with Drafts=%t, want %t",
					one.draft, held.config.Drafts, one.drafts)
			}
			var banners bytes.Buffer
			banner(held, "was told by -advertise", &banners)
			said := strings.Contains(banners.String(), "draft ")
			if said != one.drafts {
				t.Errorf("-draft=%t: the banner %s the draft line, want the other way:\n%s",
					one.draft, map[bool]string{true: "draws", false: "does not draw"}[said],
					banners.String())
			}
			// The advice is the point of the line rather than the word: a player
			// who joins a drafting room with a squad selected is refused, and
			// until the handshake is two-phase that refusal is the only other
			// thing that tells them.
			if one.drafts && !strings.Contains(banners.String(), "NO SQUAD") {
				t.Errorf("a drafting room's banner does not say to join with no squad:\n%s",
					banners.String())
			}
		})
	}
}

// TestADraftingSeriesOpensAndSaysWhatItIs is the refusal that was lifted, held
// from the other side.
//
// ⚠️ **This binary used to refuse `-draft -battles 3` in words of its own**,
// naming two flags where the room's sentence named two struct fields nobody at a
// terminal typed. It stood while "what a draft means across a series" was
// undecided; the decision is a draft a battle, out of a fresh pool each time, so
// there is nothing left to refuse.
//
// What replaces it is the banner. A host who typed both flags is opening three
// ban-and-picks rather than one, and a player who drafted a side they liked will
// otherwise expect it back in battle two — so the line is drawn only in a series,
// for the draft line's own reason: a line every ordinary drafting host reads past
// is how the one that matters stops being read.
func TestADraftingSeriesOpensAndSaysWhatItIs(t *testing.T) {
	chosen := aRoom()
	chosen.draft, chosen.battles = true, 3

	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	var out, errs bytes.Buffer
	held, err := open(chosen, netip.MustParseAddr("10.0.0.7"), dependencies, &out, &errs)
	if err != nil {
		t.Fatalf("-draft with -battles 3 was refused: %v", err)
	}
	t.Cleanup(func() { _ = held.stop() })

	var banners bytes.Buffer
	banner(held, "was told by -advertise", &banners)
	said := banners.String()
	if !strings.Contains(said, "once per battle") {
		t.Errorf("a drafting series' banner does not say the draft is re-run, so a host "+
			"reads it as the bo1 rule:\n%s", said)
	}
	if !strings.Contains(said, "fresh pool") {
		t.Errorf("the banner does not say the pool resets, and a player who liked their "+
			"drafted side will expect it back:\n%s", said)
	}

	// The line is drawn only in a series: a bo1 draft is the ordinary case and
	// says nothing extra.
	one := aRoom()
	one.draft, one.battles = true, 1
	var quiet bytes.Buffer
	single := hosting(t, one, documented, &quiet, &quiet)
	var singleBanner bytes.Buffer
	banner(single, "was told by -advertise", &singleBanner)
	if strings.Contains(singleBanner.String(), "once per battle") {
		t.Errorf("a bo1 draft's banner explains a series rule:\n%s", singleBanner.String())
	}
}
