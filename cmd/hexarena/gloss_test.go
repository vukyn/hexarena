package main

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/seed"
)

// TestLogGlossesSurviveABattleWithNoTraitBook holds the one branch logGlosses has
// of its own.
//
// ⚠️ **The guard is load-bearing and reads as tidy**, which is the combination that
// gets one deleted. `passive.Book.All` builds its result off the book's own slice,
// so it **panics** on a nil receiver rather than answering with nothing —
// battle.Books.validate requires four books and not this one, and a battle whose
// units hold no traits is exactly the case the field was made optional for. The
// mutation this exists for is dropping the nil check: nothing in the shipped game
// reaches it, because seed.Books always supplies a passive book.
//
// This is the first test in this package, and it is deliberately the only one: the
// rest of the file is a prompt loop over stdin, which is why the client has never
// had any and why cmd/hexforge is what a script drives.
func TestLogGlossesSurviveABattleWithNoTraitBook(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	full := logGlosses(books)
	if len(full) == 0 {
		t.Fatal("the shipped books produced no names at all")
	}

	traitless := books
	traitless.Passives = nil
	bare := logGlosses(traitless)
	if len(bare) == 0 {
		t.Fatal("a battle with no trait book produced no names, so the skills and " +
			"statuses were lost with the traits")
	}
	// The traits are gone and nothing else is: a missing book costs its own names.
	for _, held := range books.Passives.All() {
		if bare[held.ID] != "" {
			t.Errorf("the trait %s was named without a passive book", held.ID)
		}
	}
	for _, one := range books.Skills.Skills() {
		if bare[one.ID] == "" {
			t.Errorf("the skill %s lost its name with the trait book", one.ID)
		}
	}
}

// TestTheReplayRenderNamesTheIDsItPrints is the half of this change that is only
// reachable through the replay path: books are loaded for the render rather than
// left to --verify, so a saved battle printed without -verify still reads.
func TestTheReplayRenderNamesTheIDsItPrints(t *testing.T) {
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	glosses := logGlosses(books)
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load roster: %v", err)
	}
	fight, err := battle.New(books, 11, roster)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	fight.Begin()
	if _, err := fight.RunToEnd(4000); err != nil {
		t.Fatalf("run: %v", err)
	}
	// Every skill a real battle casts must have arrived with a name, which is the
	// claim a replay of that battle rests on.
	cast := 0
	for _, event := range fight.Drain() {
		if event.Kind != battle.SkillUsed {
			continue
		}
		cast++
		if glosses[event.Skill] == "" {
			t.Errorf("the battle cast %s and the replay would print it bare", event.Skill)
		}
	}
	if cast == 0 {
		t.Fatal("the battle cast nothing, so no skill id was measured")
	}
}
