package main

import (
	"bytes"
	"net/netip"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
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

// TestADraftingSeriesIsRefusedByTheFlagsRatherThanByTheRoom is the refusal this
// binary owes over the one the room already has.
//
// ⚠️ **room.Config.Validate refuses this too, and its sentence is the authority
// on WHY** — decision (d), a ban lasts the match, so drafting a series is a
// different game. What it cannot be is the sentence a person reads here: it names
// `Drafts` and `Battles`, which are fields of a struct nobody at a terminal
// typed. Two flags were typed, so the refusal names two flags and says which
// pairs are legal.
//
// That is the one place this binary is allowed to reword a room refusal, and the
// bound is written into the check: everything else is surfaced word for word,
// because a second wording of "a series of 2 battles is even" would be a second
// place to keep it true.
func TestADraftingSeriesIsRefusedByTheFlagsRatherThanByTheRoom(t *testing.T) {
	chosen := aRoom()
	chosen.draft, chosen.battles = true, 3

	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	var out, errs bytes.Buffer
	held, err := open(chosen, netip.MustParseAddr("10.0.0.7"), dependencies, &out, &errs)
	if err == nil {
		t.Cleanup(func() { _ = held.stop() })
		t.Fatal("-draft with -battles 3 opened a room, and a ban cannot last a series")
	}
	said := err.Error()
	for _, want := range []string{"-draft", "-battles"} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal does not name %s, so it names a struct field rather than "+
				"what somebody typed: %q", want, said)
		}
	}
	// ⚠️ The vacuity guard, and the first version of it was itself vacuous.
	//
	// It asserted the refusal does not contain "Drafts" or "Battles", on the
	// premise that the room's sentence names its own fields. It does not — it
	// reads "a drafting room of 3 battles has no rule to run under", which is
	// good prose and mentions neither identifier, so that check could never fire
	// and the two flag-name assertions above were carrying the whole test.
	//
	// What actually discriminates is the sentence itself: the room's refusal is a
	// real refusal of the same thing, so the claim worth holding is that this is
	// **not that sentence**. If the flags stop refusing first, Open surfaces the
	// room's words word for word and this fails by equality rather than by a
	// substring nobody guaranteed.
	configuration := room.Config{
		Drafts: true, Battles: 3, Format: wire.Format(chosen.format),
		Allowance: chosen.allowance, TurnCap: chosen.turns,
	}
	refusedByTheRoom := configuration.Validate()
	if refusedByTheRoom == nil {
		t.Fatal("the room accepts a drafting bo3, so this test measures nothing about the flags " +
			"coming first — the room's own refusal is what makes the ordering a claim")
	}
	if said == refusedByTheRoom.Error() {
		t.Errorf("the refusal is the room's own sentence surfaced from Open rather than the "+
			"flags' refusal firing first: %q", said)
	}
}
