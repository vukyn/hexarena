package tui_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

// TestAHealedLineSaysTheHealingWasCut is the log half of the mechanic, and the
// claim is that without it the line lies.
//
// A reader sees `heals 244` where the book says nine hundred and every figure they
// could check it against — the skill's own restores, the status's tick power, the
// drain share already on the same event — says the log is wrong. That is the trap
// Pierce, Refused, Drained and Gradient are all on their events for.
//
// All three Healed arms are tabled, because the branch is one `switch` deep and
// each arm builds its own sentence: a regeneration names the status, a drain names
// its share, and a restore names neither. A note added to one of the three is
// exactly the shape of gap this repository has paid for before.
//
// # Both languages
//
// This package builds an event line out of the event alone and is never handed a
// Lang — a name is a caller-supplied MAP — so "both languages" here is the two
// maps there are: nil, which is what English and a bookless replay both get, and
// i18n.Vi's. The note is the package's own English wording either way, and the
// point of running both is that it must be there in both and must not disturb the
// gloss brackets around it.
func TestAHealedLineSaysTheHealingWasCut(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	vi := i18n.Vi.LogGlosses(books.Skills.Skills(), books.Statuses.Kinds(), books.Passives.All())
	if vi["regrowth"] == "" {
		t.Fatal("the shipped books gloss no regrowth, so the Vietnamese arm would read a bare id")
	}

	base := battle.Event{
		Kind: battle.Healed, At: 5000, Turn: 3, Actor: "a",
		Amount: 244, Stacks: 2, Remaining: 1900,
	}
	for _, arm := range []struct {
		name    string
		edit    func(*battle.Event)
		glossed string
	}{
		{"a regeneration", func(e *battle.Event) { e.Status = "regrowth" }, "regrowth"},
		{"a drain", func(e *battle.Event) { e.Drained = 600 }, ""},
		{"a restore", func(e *battle.Event) {}, ""},
	} {
		t.Run(arm.name, func(t *testing.T) {
			for _, language := range []struct {
				name    string
				glosses map[string]string
			}{
				{"en", nil}, {"vi", vi},
			} {
				t.Run(language.name, func(t *testing.T) {
					cut := base
					arm.edit(&cut)
					cut.Reduced = 800

					clean := base
					arm.edit(&clean)

					line := tui.Line(cut, nil, language.glosses)
					plain := tui.Line(clean, nil, language.glosses)

					// The control first: the same arm with nothing cut says nothing
					// about a cut, so what is found below is the field rather than a
					// word that was always on the line.
					if strings.Contains(plain, "cut") {
						t.Fatalf("an uncut %s already says %q, so this arm proves nothing", arm.name, plain)
					}
					if line == plain {
						t.Fatalf("a %s with 800 per mille cut renders identically to one with none: %q",
							arm.name, line)
					}
					if !strings.Contains(line, "80%") {
						t.Errorf("a %s cut by 800 per mille renders %q, which does not name the 80%% it lost",
							arm.name, line)
					}
					if !strings.Contains(line, "cut") {
						t.Errorf("a %s renders %q, which does not say the healing was cut", arm.name, line)
					}
					// The rest of the line survives: the amount that landed is still
					// there, and a glossing arm still glosses.
					if !strings.Contains(line, "244") {
						t.Errorf("a %s lost its own amount: %q", arm.name, line)
					}
					if arm.glossed != "" && language.glosses != nil &&
						!strings.Contains(line, i18n.GlossBracket(arm.glossed, vi[arm.glossed])) {
						t.Errorf("a %s dropped the gloss on %s: %q", arm.name, arm.glossed, line)
					}
				})
			}
		})
	}
}
