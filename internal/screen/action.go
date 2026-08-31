package screen

import "fmt"

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
// there are four Kinds and no more, and a fifth would be a claim that a screen
// wants something none of these six wanted.
type Action struct {
	Kind   Kind
	Target Target

	// Focus is an id the raised screen should land on, empty for none.
	//
	// ⚠️ Applying it is the **client's** job, and that is the point rather than a
	// division of labour. The traits listing used to reach into the statuses
	// screen's own state and move its cursor — the only cross-screen read among
	// the six, and the one thing a screen could not have carried with it into a
	// package that knows nothing about which other screens exist. A raise names
	// what it wants and asks nobody where it lives.
	//
	// ⚠️ An id the raised screen cannot find is a raise that does **not
	// happen**: the reader stays where they are. A trait naming a status the
	// book has lost is a trait already printing a bare id, and landing a cursor
	// on whatever sorted next would answer a question nobody asked.
	Focus string
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
	// Raise opens another screen, named by Target and optionally landed on Focus.
	Raise
)

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
	// Statuses is the reference for the timed effects, which is the one raise
	// that carries a Focus.
	Statuses
)

// TargetCount is how many Target values are declared, NoTarget included.
//
// ⚠️ It exists so a client can prove its map is **total**. A Target with no
// entry makes a raise silently do nothing — the same shape as a screen slipping
// out of everyScreen, which this repository has now recorded five times — and a
// test that walks the values somebody remembered to list cannot see the one they
// forgot. Walk the count.
const TargetCount = int(Statuses) + 1

var targetNames = [TargetCount]string{
	NoTarget: "none",
	Chart:    "chart",
	Statuses: "statuses",
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
