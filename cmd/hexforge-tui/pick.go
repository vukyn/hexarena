package main

import (
	tea "charm.land/bubbletea/v2"

	draw "github.com/vukyn/hexarena/internal/screen"
)

// What this client kept when the multi-select moved into internal/screen.
//
// The picker itself is draw.PickState now — the list, the cursor, the filter,
// the toggle, the reading pane and the whole of the drawing — because a second
// full-screen client raises the same lists and must not have a second answer to
// any of them. Three things could not go with it, and each is here for the same
// reason draw.Action exists: they are this client's own vocabulary.
//
//   - pickDest, below, which names one field of one of *these* screens.
//   - the destination dispatch (pickLanding, pickedInto, model.answerPick), in
//     model.go beside the guard's, because it writes this model.
//   - model.pick, which is where a raised picker is put in front, since only
//     this client has a field to put it in.
//
// Two forwarders are here rather than deleted, on the rule this package already
// follows for pad, clip, clamp and window: numberKey below and numberField in
// style.go are still spent by the squad builder, which has not moved. Anything
// the move left with no caller went, and traitRoom is the one that did.
//
// ⚠️ **Six of the ten destinations have since left too.** The skill form moved
// into internal/screen in the step after this one, and the five allowlists and
// the inflicts field went with it as draw.SkillsPick — a destination names a
// field of one screen, and that screen is no longer a client's. What is left
// below is the character form's two and the squad builder's two.

// The picker's own vocabulary, named here under the spellings this package
// already used, for the reason the six reference screens are aliased in
// model.go: an alias is one declaration rather than two, and the ten raise
// sites, the three landings and this client's fixtures go on reading as they
// read.
//
// ⚠️ Aliases and not wrappers. A wrapper would be a second place a cursor, a
// filter or a chosen list could live, and there is one.
type (
	pickState  = draw.PickState
	pickOption = draw.PickOption
	pickAnswer = draw.PickAnswer
)

// The eight sorts of list a picker draws, named here for the reason the types
// above are.
const (
	pickSkills     = draw.PickSkills
	pickElements   = draw.PickElements
	pickArchetypes = draw.PickArchetypes
	pickCharacters = draw.PickCharacters
	pickSpecies    = draw.PickSpecies
	pickOrigins    = draw.PickOrigins
	pickStatuses   = draw.PickStatuses
	pickPassives   = draw.PickPassives
)

// pick raises a picker over whatever screen is in front. Nothing is applied
// until it is closed with enter.
//
// The defaults a raise did not fill in are draw.PickState.Raise's, so the ten
// literals in form.go, skills.go and squads.go get the same three answers; what
// is left here is the one thing that package cannot do, which is knowing where a
// picker in front is kept.
func (m model) pick(state *pickState) model {
	m.picker = state.Raise()
	return m
}

// numberKey reports whether a keystroke belongs in a number field: a digit, or
// one of the keys that edit or move within one.
//
// Forwarded rather than left behind, because the squad builder's level field
// asks the same question and that screen has not moved. The house pattern pad,
// clip, clamp and window already follow: the call sites read unchanged, and
// there is still exactly one body.
func numberKey(message tea.KeyPressMsg) bool { return draw.NumberKey(message) }

// pickDest is where a finished pick lands: one named field of one of *this
// client's* screens.
//
// A closed enum rather than a screen alone, because a screen cannot say which of
// its own fields was being filled — the character form's kit and species are one
// keystroke apart and differ in nothing else. It is the same division
// guardSubject makes for the squad builder's two questions, taken one step finer.
//
// ⚠️ **It used to name ten and names four.** The other six were the skill form's
// five allowlists and its inflicts field, and they went with that screen into
// internal/screen as draw.SkillsPick — the reason PickState.Into is an `any` is
// that a destination names a field of a client's screen, and the skill form
// stopped being one. What is left names the character form's two and the squad
// builder's two, which have not moved. This client is the one thing that knows
// both vocabularies, which is what a client is for; see pickedInto.
type pickDest uint8

const (
	// pickNowhere is the zero value: a picker whose answer goes nowhere. It is
	// what a pickState built by hand carries, and no raise in this client uses
	// it — enter on such a picker closes the list and writes nothing, which is
	// exactly what the nil callback this replaced did.
	//
	// ⚠️ It is deliberately outside the dispatch, and
	// TestEveryPickDestinationLandsSomewhere asserts that as well: it is the one
	// destination for which swallowing an answer is the definition rather than
	// the defect.
	pickNowhere pickDest = iota
	// The character form's two, from form.go.
	pickIntoKit
	pickIntoSpecies
	// The squad builder's two halves of a loadout, from squads.go.
	pickIntoSquadKit
	pickIntoSquadTrait
	// pickDestCount is the count the dispatch is held total against, in the
	// shape draw.SubjectKindCount and draw.TargetCount already have: a
	// destination added above it enters the walk without anybody remembering to
	// list it, which a hand-written list of ten would not give.
	pickDestCount
)
