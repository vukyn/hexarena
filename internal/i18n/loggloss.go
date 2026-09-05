package i18n

import (
	"sort"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// LogGlosses is the names the battle log may put beside the ids it prints: data
// id -> Vietnamese name, for skills, statuses and traits together.
//
// It lives here because it is a wording table and nothing else — the same three
// accessors every other screen reads (SkillName, Gloss, PassiveName), collected
// into the one shape a renderer can take without being handed a language. That
// indirection is the point rather than a cost: internal/tui builds its event
// lines from the event alone, so it may not be given books, a battle or a Lang,
// and a caller-supplied map is how tags and Summary's names already reach it.
// Reading the same three accessors is also what stops a name authored through
// hexforge from drifting away from the log — there is no second table here.
//
// # The nil is the English answer
//
// English returns **nil**, and that is a positive answer rather than a failure to
// produce one: in English a data id is shown exactly as the data writes it, which
// is the rule Gloss, SkillName and PassiveName all state for themselves. A nil
// map is also what a replay rendered without books gets, and tui.Line draws the
// bare id for both — one code path, and the property the English goldens hold.
//
// # Why a skill cannot go through Gloss
//
// Gloss answers off the compiled tables in gloss.go, and skillGloss holds the
// nineteen ids that shipped **before** skill.Skill carried a name. Not one of
// them is in skills.json any more: all 43 shipped skills carry an authored name
// and 0 of them are in that table, so a log glossed through Gloss would name no
// skill at all. Traits are the same shape (11 shipped, 11 authored names, no
// table). Statuses are the one kind the id tables cover completely — 22 shipped,
// 22 glossed, none carrying a name field — which is why the three kinds are read
// through three different accessors rather than one.
func (l Lang) LogGlosses(carried []skill.Skill, kinds []status.Kind, held []passive.Passive,
	awarded []composition.Bonus) map[string]string {
	if l != Vi {
		return nil
	}
	// ⚠️ An id claimed by two of the three kinds is LEFT OUT, never picked
	// between. This map is one namespace over three books that do not share one,
	// so nothing here can tell which kind the event printing the id meant — and a
	// wrong name is worse than a missing one, which is exactly the reasoning
	// TestNoIDIsGlossedTwice applies to the compiled tables. A bare id is this
	// package's declared behaviour for an unglossed id, so leaving it out degrades
	// into something already normal; taking whichever book was walked last would
	// put a status's name on a skill with nothing on screen looking wrong. `taunt`
	// is a shipped skill id and `taunting` a shipped status id, so the near miss is
	// real rather than hypothetical.
	//
	// It is read off the **id** and not off which kinds offered a name, so the two
	// functions here agree exactly and a collision cannot become live later: an id
	// shared with a kind that has no name today would start answering for the wrong
	// thing the day somebody authors one. LogGlossCollisions is the loud half.
	collided := make(map[string]bool)
	for _, id := range LogGlossCollisions(carried, kinds, held, awarded) {
		collided[id] = true
	}
	out := make(map[string]string, len(carried)+len(kinds)+len(held)+len(awarded))
	put := func(id, name string) {
		if id == "" || name == "" || name == id || collided[id] {
			return
		}
		out[id] = name
	}
	for _, one := range carried {
		put(one.ID, l.SkillName(one))
	}
	for _, kind := range kinds {
		put(kind.ID, l.Gloss(kind.ID))
	}
	for _, one := range held {
		put(one.ID, l.PassiveName(one))
	}
	for _, one := range awarded {
		put(one.ID, l.BonusName(one))
	}
	return out
}

// LogGlossCollisions is every id declared by more than one of the four kinds,
// sorted.
//
// It exists because LogGlosses cannot be loud on its own: it is asked while a
// screen is being drawn, and this package's standing rule is that a data id
// nobody has named renders bare — never an error, never a placeholder — because
// skills, statuses and traits are added by editing JSON. So the map leaves a
// collision out and this reports it, which puts the failure where it can be one:
// TestNoLogGlossCollidesAcrossKinds runs it over the shipped books and names any
// id that turns up.
//
// Sorted rather than ranged out of the map, the same discipline internal/core
// holds: an order that reaches an output may not come from a map.
func LogGlossCollisions(carried []skill.Skill, kinds []status.Kind, held []passive.Passive,
	awarded []composition.Bonus) []string {
	claimed := make(map[string]int, len(carried)+len(kinds)+len(held)+len(awarded))
	for _, one := range carried {
		claimed[one.ID]++
	}
	for _, kind := range kinds {
		claimed[kind.ID]++
	}
	for _, one := range held {
		claimed[one.ID]++
	}
	for _, one := range awarded {
		claimed[one.ID]++
	}
	var out []string
	for id, count := range claimed {
		if count > 1 {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
