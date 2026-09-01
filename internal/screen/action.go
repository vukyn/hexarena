package screen

import (
	"fmt"

	"github.com/vukyn/hexarena/internal/i18n"
)

// Action is what a screen asks for once it has read a keystroke: it says what it
// wants and leaves what that means to whichever client is in front.
//
// A screen used to write the answer itself — `m.screen = screenMenu` — and
// `screen` is one client's own enum, so a screen that named a menu entry could
// only ever live in the client that had that menu. Two clients draw these
// screens and the second one's menu is a different menu, which is why the naming
// moved out rather than being shared: a shared enum would be one client's menu
// declared where the other one could read it.
//
// **The vocabulary was measured rather than invented.** Over the six screens
// this arrived for — the affinity chart, the elements listing, the statuses and
// traits references, the species listing and the build catalogue — every key
// does exactly one of three things: it quits, it goes back one step, or it
// raises another reference. Two of them raise (the elements listing opens the
// chart with `g`, the traits listing opens the statuses reference with `?`,
// landing it on the status the trait just named) and the other four do not. So
// there were four Kinds, and a fifth would have been a claim that a screen
// wanted something none of those six wanted.
//
// **Two screens later it does.** The skill form is the first moved screen with
// something to lose, so it asks before throwing it away, and the first with
// lists to fill in, so it opens a picker over itself. Neither could be said in
// the four: an Ask is not a Raise (nothing is being navigated to — the question
// is drawn over whatever is in front and the reader comes straight back), and a
// Pick is not one either (the picker is not a screen a client names, it is a
// state the screen built and handed over). So there are six, and the count
// below is what stops the seventh arriving unhandled.
type Action struct {
	Kind   Kind
	Target Target

	// Subject is what the raise is about, and the zero value is a raise about
	// nothing.
	//
	// ⚠️ Applying it is the **client's** job, and that is the point rather than a
	// division of labour. The traits listing used to reach into the statuses
	// screen's own state and move its cursor — the only cross-screen read among
	// the six, and the one thing a screen could not have carried with it into a
	// package that knows nothing about which other screens exist. A raise names
	// what it wants and asks nobody where it lives.
	//
	// ⚠️ It replaced a bare `Focus string`, which was the same mechanism with one
	// case in it, and the case was undeclared: a Focus at any target but the
	// statuses reference made the client's applier answer "not found", and a
	// declined raise is silent by design — so a raise carrying a subject to the
	// wrong screen looked exactly like a reader who had not pressed the key. The
	// kinds are counted now, for the reason TargetCount is counted.
	Subject Subject

	// Question is the wording an Ask puts to the reader, and is a key rather
	// than a sentence so that switching language with one pending redraws it.
	//
	// ⚠️ **What the question is *about* is deliberately not carried, and this is
	// the field the squad builder will have to widen.** The one asking screen in
	// this package names nothing — a form throwing its own draft away is about
	// the screen that asked, so the client fills in its own zero subject — while
	// the squad builder asks two questions told apart by a squad id. That screen
	// is still in cmd/hexforge-tui and still calls the client's own `ask`, so
	// there is nothing here for a subject to be about yet, and a field only nil
	// is ever passed for is a field no test can discriminate.
	//
	// ⚠️ **Subject above is NOT the carrier and may not become one.** Its Kind is
	// the closed list of things a *raise* may land on, counted by
	// SubjectKindCount and held total by a client's raise applier — so a squad
	// kind added there would demand a raise applier for a kind no raise carries.
	// That was measured in #217's step and the answer was a client-local
	// guardSubject; the same answer belongs here, carried as an `any` the way
	// PickState.Into is, on the day a screen that asks about something moves.
	Question i18n.Key

	// Picker is the list a Pick wants put in front, built by the screen and
	// owned by the client.
	//
	// ⚠️ **A pointer, and it stays one.** The client keeps a `*PickState` and
	// *that* pointer is its presence flag — nil is no picker — so a value here
	// would be a second shape for one thing and would break
	// (*PickState).Toggle, which mutates in place. The screen says *open this*;
	// the client owns *while it is open*, because the client is the only thing
	// with a field to keep one in and the only thing that knows what closing one
	// costs.
	//
	// It is handed over **unraised**: the screen builds the literal and the
	// client calls Raise, which is exactly the division the ten raise sites
	// already had when they all lived in one binary.
	Picker *PickState
}

// Subject is what a raise is about: the thing the screen being raised is being
// asked to land on, or to describe.
//
// ⚠️ **A tagged struct rather than an interface**, and the reason is Action
// rather than taste. An Action is a comparable value a screen returns and a test
// writes out as a literal; an interface field would make the zero Action hold a
// nil the applier has to test for separately from "no subject", which is the
// absence-encoded-into-a-value mistake this repository has paid for twice (a
// queue reading of nought meaning soonest, a sentinel log offset meaning
// following). A Kind at nought means nothing was asked for, exactly as NoTarget
// does, and SubjectKindCount is what lets a client prove it handles the rest.
//
// **Every kind carries an ID and that is the whole idiom**: a raise names what it
// wants by id and asks nobody where it lives, so the describer looks the id up in
// the books it was handed. Handing the resolved value over instead would put the
// lookup — and its refusal — on the raiser, which is the screen with nowhere to
// draw one.
type Subject struct {
	Kind SubjectKind

	// ID is a status id, a skill id or a character id, depending on Kind.
	ID string

	// Level is the level a character is being read at, and only a
	// CharacterSubject spends it. A trait comes in at a level, so "what is this
	// unit carrying" has no answer without one.
	Level int

	// At and Of are where the subject sat in the list it was chosen out of, At
	// counting from one.
	//
	// ⚠️ **Of is nought when that list was empty**, which is how a raiser says
	// there was nothing to describe without the describer reaching back into the
	// screen behind it to count. A describer handed nought draws the same "none
	// of these" line it used to draw after looking.
	At, Of int
}

// SubjectKind is what sort of thing a Subject names.
type SubjectKind uint8

const (
	// NoSubject is the zero value and is what a raise about nothing carries — the
	// elements listing opening the chart is a whole screen rather than a thing on
	// one, so it names none.
	NoSubject SubjectKind = iota
	// StatusSubject is a status the raised screen should put its cursor on.
	//
	// ⚠️ An id the raised screen cannot find is a raise that does **not happen**:
	// the reader stays where they are. A trait naming a status the book has lost
	// is a trait already printing a bare id, and landing a cursor on whatever
	// sorted next would answer a question nobody asked.
	StatusSubject
	// SkillSubject is a skill to describe, and where it sat in the list it came
	// out of.
	//
	// ⚠️ **One kind covers the authoring listing and the battle's option list**,
	// which were measured as two and are one. Both hand over a skill id and a
	// position, both are drawn by the same paragraph under the same heading with
	// the same footer, and the only difference — which list the position counts —
	// is already carried in At and Of. A second kind nothing could branch on
	// would be two vocabularies for one idea.
	SkillSubject
	// CharacterSubject is a character read at a level.
	//
	// ⚠️ **Two describers answer it and that is why it is one kind.** The traits
	// blurb says what the character carries at that level and the art preview
	// draws the form it wears there; both are readings of the same subject, so
	// naming them apart would be naming the *describer* rather than the thing,
	// which is what Target already does.
	CharacterSubject
)

// SubjectKindCount is how many SubjectKind values are declared, NoSubject
// included.
//
// ⚠️ It exists for the reason TargetCount does: a client can prove its applier is
// **total**. A kind nothing applies makes a raise hand the describer nothing and
// draw an empty screen, and a test that walks the kinds somebody remembered to
// list cannot see the one they forgot. Walk the count.
const SubjectKindCount = int(CharacterSubject) + 1

var subjectKindNames = [SubjectKindCount]string{
	NoSubject:        "none",
	StatusSubject:    "status",
	SkillSubject:     "skill",
	CharacterSubject: "character",
}

// String names a SubjectKind for a diagnostic, and nothing draws it — the same
// shape, and the same reason, as Target.String below.
func (k SubjectKind) String() string {
	if int(k) >= SubjectKindCount {
		return fmt.Sprintf("subject(%d)", uint8(k))
	}
	return subjectKindNames[k]
}

// Kind is what a screen wants done.
type Kind uint8

const (
	// Stay is the zero value, and that is load-bearing rather than tidy: a
	// screen handed a key it has nothing to do about returns an Action it never
	// filled in, and the client does nothing with it. A "do nothing" that had to
	// be spelled out is one keystroke handler away from being forgotten, and a
	// forgotten one would fall into whichever Kind sat at nought.
	Stay Kind = iota
	// Back is one step towards wherever the reader came from, and the client is
	// what remembers where that was.
	//
	// ⚠️ A screen may not name its own way back even when it has only one
	// raiser. The chart's did — it went to the elements listing by name, on the
	// true and fragile ground that nothing else raises it — and a screen that
	// spells out the one door it has today is a screen that is wrong the day it
	// has two, in a client it was not written for.
	Back
	// Quit ends the session.
	//
	// An action rather than a tea.Cmd, and that is not ceremony: it lets the
	// client decide what quitting means. The authoring tool ends there; a game
	// client with a battle half played may well want to ask first, and a screen
	// that returned the command itself would have taken that decision away from
	// it.
	Quit
	// Raise opens another screen, named by Target and about whatever Subject
	// names.
	Raise
	// Ask puts a confirmation to the reader, worded by Question, and leaves the
	// screen where it is.
	//
	// ⚠️ **Not a Raise, and the difference is not bookkeeping.** A raise names a
	// Target — one of a closed list of *screens* — and the client remembers
	// where it came from so Back can return. A question is drawn over whatever
	// is already in front, takes one keystroke, and hands the answer back to the
	// screen that asked through that screen's own Confirmed. There is nothing to
	// name and nowhere to go back to, so a Target would have to be invented for
	// a screen a client does not have.
	Ask
	// Pick puts the list in Picker in front, over whatever raised it.
	//
	// ⚠️ **Also not a Raise**, for a sharper reason: a Target is a *request* the
	// client turns into one of its own screens, and a picker is not one of them.
	// It is a value this package built, holding the rows, the cursor and the
	// destination the answer belongs to, and the client's only job is to keep it
	// while it is open. There is no map from Pick to anything.
	Pick
)

// KindCount is how many Kind values are declared, Stay included.
//
// ⚠️ **It exists because the two kinds above are new and a client that silently
// ignored one would swallow a keystroke.** That is the shape this repository
// has now recorded five times in TODO.md and measured four more — a target with
// no screen, a subject kind with no applier, a pick destination with no landing,
// a screen slipping out of everyScreen — and each time the fix was to walk a
// count rather than a list somebody remembered to write.
//
// ⚠️ **And a count is the smaller half.** It proves a kind is *handled*; it
// cannot prove it is handled *right*, which #207, #214, #216 and #218 each
// measured again. Every kind therefore has a behaviour test that presses the
// real key beside its entry in the walk.
const KindCount = int(Pick) + 1

var kindNames = [KindCount]string{
	Stay:  "stay",
	Back:  "back",
	Quit:  "quit",
	Raise: "raise",
	Ask:   "ask",
	Pick:  "pick",
}

// String names a Kind for a diagnostic, and nothing draws it — the same shape,
// and the same reason, as Target.String and SubjectKind.String.
func (k Kind) String() string {
	if int(k) >= KindCount {
		return fmt.Sprintf("kind(%d)", uint8(k))
	}
	return kindNames[k]
}

// Target is a screen a raise may name.
//
// A short closed list rather than a screen id of any kind, because the whole
// point is that this package does not know what views the client in front has: a
// Target is a **request**, and the client's own map is what turns it into
// whichever of its screens answers to it.
type Target uint8

const (
	// NoTarget is the zero value and is what every Action that is not a Raise
	// carries.
	NoTarget Target = iota
	// Chart is the affinity chart drawn as the rings it was declared in.
	Chart
	// Statuses is the reference for the timed effects, raised on a
	// StatusSubject so it lands on the one that was named.
	Statuses
	// Blurb is the description screen: what the thing under the cursor behind it
	// does, in the sentences a player reads.
	//
	// ⚠️ **Two of its three raisers name it now** — the cast browser and the
	// skill listing — and the third, the played battle, still writes the
	// client's own enum because that screen has not moved. Declaring the target
	// ahead of any raiser is what put both of them under
	// TestEveryRaiseTargetNamesAScreenInThisClient before they arrived rather
	// than after.
	Blurb
	// Preview is the art a character shows at the level it is being read at, and
	// is declared ahead of its raiser for the reason Blurb is.
	Preview
)

// TargetCount is how many Target values are declared, NoTarget included.
//
// ⚠️ It exists so a client can prove its map is **total**. A Target with no
// entry makes a raise silently do nothing — the same shape as a screen slipping
// out of everyScreen, which this repository has now recorded five times — and a
// test that walks the values somebody remembered to list cannot see the one they
// forgot. Walk the count.
const TargetCount = int(Preview) + 1

var targetNames = [TargetCount]string{
	NoTarget: "none",
	Chart:    "chart",
	Statuses: "statuses",
	Blurb:    "blurb",
	Preview:  "preview",
}

// String names a Target for a diagnostic, and nothing draws it.
//
// It is not wording: a client's screens are named in internal/i18n like
// everything a reader sees, and these are the ids a failing test prints so the
// value it is complaining about can be found. Same shape, and the same reason,
// as the fallback on every other enum here.
func (t Target) String() string {
	if int(t) >= TargetCount {
		return fmt.Sprintf("target(%d)", uint8(t))
	}
	return targetNames[t]
}
