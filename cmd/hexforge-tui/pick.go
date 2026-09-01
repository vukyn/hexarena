package main

import (
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
// Two forwarders used to be here on the rule this package follows for pad,
// clip, clamp and window — numberKey, and numberField in style.go — and both are
// gone: the squad builder's level field was the last caller either had in this
// package, and a forwarder with no call site is a body declared twice for
// nobody.
//
// ⚠️ **Eight of the ten destinations have since left too.** The skill form moved
// into internal/screen and took the five allowlists and the inflicts field with
// it as draw.SkillsPick; the squad builder followed and took both halves of a
// loadout as draw.SquadsPick. A destination names a field of one screen, and
// neither of those screens is a client's any more. What is left below is the
// character form's two, which have not moved.

// The picker's own vocabulary, named here under the spellings this package
// already used, for the reason the six reference screens are aliased in
// model.go: an alias is one declaration rather than two, and the raise sites,
// the three landings and this client's fixtures go on reading as they read.
//
// ⚠️ Aliases and not wrappers. A wrapper would be a second place a cursor, a
// filter or a chosen list could live, and there is one.
type (
	pickState  = draw.PickState
	pickOption = draw.PickOption
	pickAnswer = draw.PickAnswer
)

// The two sorts of list this client still raises, named here for the reason the
// types above are.
//
// ⚠️ **There were eight and there are two**, and the six that went were deleted
// rather than kept for symmetry: an alias is a spelling for a call site, so an
// alias with no call site is a second declaration of a constant for nobody. The
// six lists this client no longer raises are raised inside internal/screen now,
// under draw.PickElements and its five siblings, which is where their raisers
// went.
const (
	pickSkills  = draw.PickSkills
	pickSpecies = draw.PickSpecies
)

// pick raises a picker over whatever screen is in front. Nothing is applied
// until it is closed with enter.
//
// The defaults a raise did not fill in are draw.PickState.Raise's, so the two
// literals left in form.go and the eight that went into internal/screen with
// their screens all get the same three answers; what is left here is the one
// thing that package cannot do, which is knowing where a picker in front is
// kept.
func (m model) pick(state *pickState) model {
	m.picker = state.Raise()
	return m
}

// pickDest is where a finished pick lands: one named field of one of *this
// client's* screens.
//
// A closed enum rather than a screen alone, because a screen cannot say which of
// its own fields was being filled — the character form's kit and species are one
// keystroke apart and differ in nothing else. It is the same division
// draw.SquadsAsk makes for the squad builder's two questions, taken one step
// finer.
//
// ⚠️ **It used to name ten and names two.** Six were the skill form's five
// allowlists and its inflicts field and two were the squad builder's halves of a
// loadout; each pair went into internal/screen with the screen that raised it,
// as draw.SkillsPick and draw.SquadsPick. The reason PickState.Into is an `any`
// is exactly that — a destination names a field of *some* screen, and which
// package that screen lives in is not the destination's business. What is left
// names the character form's two, which have not moved. This client is the one
// thing that knows all three vocabularies, which is what a client is for; see
// pickedInto.
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
	// The character form's two, from form.go — the last two in this vocabulary,
	// and the only screen left in this package that raises a picker at all.
	pickIntoKit
	pickIntoSpecies
	// pickDestCount is the count the dispatch is held total against, in the
	// shape draw.SubjectKindCount and draw.TargetCount already have: a
	// destination added above it enters the walk without anybody remembering to
	// list it, which a hand-written list of two would not give.
	pickDestCount
)
