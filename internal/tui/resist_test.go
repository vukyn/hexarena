package tui_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/tui"
)

// TestARefusedApplicationSaysWhichKindOfRefusalItWas is the renderer's half of
// the field, and the reason the field exists at all.
//
// One kind covers two different things — the roll failed, or the target's traits
// refused it — and "resists" was being printed for both. A reader given that word
// and nothing else cannot tell a piece of luck from a property of the unit, which
// is exactly the confusion the mechanism was explained out of.
func TestARefusedApplicationSaysWhichKindOfRefusalItWas(t *testing.T) {
	tags := map[string]string{"a": "A1", "f": "E1"}
	line := func(chance, refused int) string {
		return tui.Line(battle.Event{
			Kind: battle.StatusResisted, At: 5000, Turn: 3,
			Actor: "a", Target: "f", Skill: "envenom", Status: "poison",
			Chance: chance, Refused: refused,
		}, tags)
	}

	// Nothing refused: the roll simply failed, and the wording is what it always
	// was, so a battle with no resistances in it reads unchanged.
	unlucky := line(400, 0)
	if !strings.Contains(unlucky, "resists") {
		t.Errorf("a failed roll rendered as %q", unlucky)
	}
	if strings.Contains(unlucky, "immune") || strings.Contains(unlucky, "refused") {
		t.Errorf("a failed roll was reported as a refusal: %q", unlucky)
	}

	// Refused outright. "Immune" rather than a percentage, because a thousand
	// refused is not a share worth reading — it is a fact about the unit.
	immune := line(0, 1000)
	if !strings.Contains(immune, "immune") {
		t.Errorf("an outright refusal rendered as %q", immune)
	}
	if !strings.Contains(immune, "poison") {
		t.Errorf("the immunity does not name the status: %q", immune)
	}

	// Partly refused: both numbers, because either alone is misleading. The
	// chance says how likely it was and the share says why it was that low.
	partial := line(400, 600)
	if !strings.Contains(partial, "40%") || !strings.Contains(partial, "60%") {
		t.Errorf("a partial refusal rendered as %q, want both figures", partial)
	}

	// The three are three different lines. Two of them reading the same would
	// mean the distinction never reaches a reader however the event is filled in.
	if unlucky == immune || unlucky == partial || immune == partial {
		t.Errorf("two of the three refusals render identically:\n%s\n%s\n%s",
			unlucky, immune, partial)
	}
}
