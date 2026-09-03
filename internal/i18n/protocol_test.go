package i18n

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/wire"
)

// This file is the walk internal/wire's own totality guards say they cannot
// take. wire.TestEveryRefusalCodeHasANameAndTravels and
// wire.TestEveryClosureHasANameAndTravels hold the *count* and the wire names,
// and each says in its comment that the wording is beyond it because
// internal/wire must not import internal/i18n — the whole point of sending an
// id is that the sentence lives at the far end. This is the far end, and the
// import here is **test-only**: Lang.Refusal and Lang.Closure take the enum's
// name, exactly as Lang.StatusCategory takes a status.Category's, so nothing in
// this package's production build knows the protocol exists.
//
// ⚠️ **What none of the four tests below can see is whether anybody draws
// these.** No screen calls Lang.Refusal or Lang.Closure yet, and
// TestNoKeyIsOrphaned cannot tell: it counts an identifier named anywhere in
// the module, and each of the thirteen keys is named in protocol.go's own
// lookup. So the whole of this file measures that the words exist, are
// complete, are distinct and are not ids — not that a player is ever shown one.
// That is the "shipped dead" shape one step narrower rather than closed, and it
// is written down in TODO.md § The client rather than implied away.

// TestEveryRefusalCodeIsWorded holds the codes complete the way the status
// categories are held complete: a wire.Code is a Go enum rather than a data id,
// so a code with no wording is a gap in the catalog rather than a value nobody
// has reached yet.
//
// It walks wire.CodeCount rather than the lookup inside Lang.Refusal, which is
// the whole point — ranging over that map would ask the table whether it holds
// what it holds, and a code added to the protocol with no line here would sail
// through. Walking the count means the day an eleventh code lands, this is red
// until somebody writes two sentences for it.
//
// What it sees: an empty wording, a missing lookup row (which falls through to
// the id), and a wording left at its own enum spelling, in both languages.
// What it cannot see: whether the sentence is *true* of the code it words —
// two rows swapped in the lookup are both non-empty and neither is its own
// spelling. TestNoProtocolWordingIsAnyEnumSpelling widens the second clause as
// far as a test can take it; past that, the doc comment on each wire.Code is
// the record and a reader is the check.
func TestEveryRefusalCodeIsWorded(t *testing.T) {
	for _, lang := range Langs() {
		for value := range wire.CodeCount {
			name := wire.Code(value).String()
			worded := lang.Refusal(name)
			if strings.TrimSpace(worded) == "" {
				t.Errorf("%v has no wording for the %q refusal", lang, name)
			}
			if worded == name {
				t.Errorf("%v left the %q refusal at its enum spelling", lang, name)
			}
		}
	}
}

// TestEveryClosureIsWorded is the same walk over the other enum, and it is a
// separate test rather than a second loop because the two enums are separate
// gaps: wire.ClosureStopped was added on its own and widened the unworded
// surface by one, which is the way a value gets in here with nothing measuring
// it.
//
// Same sight lines as its neighbour: empty, missing and self-spelled are
// visible; a swapped row is not.
func TestEveryClosureIsWorded(t *testing.T) {
	for _, lang := range Langs() {
		for value := range wire.ClosureCount {
			name := wire.Closure(value).String()
			worded := lang.Closure(name)
			if strings.TrimSpace(worded) == "" {
				t.Errorf("%v has no wording for the %q closure", lang, name)
			}
			if worded == name {
				t.Errorf("%v left the %q closure at its enum spelling", lang, name)
			}
		}
	}
}

// TestNoProtocolWordingIsAnyEnumSpelling is the clause above widened by one
// step, for the reason TestNoStatusCategoryNounIsAnyEnumSpelling is: a wording
// must not be *any* value's enum spelling, not merely its own. "room_full"
// worded "room_full" is the obvious mistake; "room_full" worded "room_unknown"
// is the same mistake with a swapped row, and a per-value comparison cannot see
// it.
//
// The two enums are pooled into one set of spellings on purpose. They share the
// name "none", and a closure worded with a code's spelling is no better than
// one worded with its own.
//
// What it sees: any wording that is an id from either enum. What it cannot see:
// a wording that is prose but describes the wrong value.
func TestNoProtocolWordingIsAnyEnumSpelling(t *testing.T) {
	spellings := make(map[string]bool, wire.CodeCount+wire.ClosureCount)
	for value := range wire.CodeCount {
		spellings[wire.Code(value).String()] = true
	}
	for value := range wire.ClosureCount {
		spellings[wire.Closure(value).String()] = true
	}
	for _, lang := range Langs() {
		for value := range wire.CodeCount {
			name := wire.Code(value).String()
			if worded := lang.Refusal(name); spellings[worded] {
				t.Errorf("%v words the %q refusal %q, which is an enum spelling",
					lang, name, worded)
			}
		}
		for value := range wire.ClosureCount {
			name := wire.Closure(value).String()
			if worded := lang.Closure(name); spellings[worded] {
				t.Errorf("%v words the %q closure %q, which is an enum spelling",
					lang, name, worded)
			}
		}
	}
}

// TestNoTwoProtocolValuesShareAWording is the clause the status categories do
// not have and this case needs.
//
// Ten refusals exist so that a player can tell ten situations apart. Two of
// them reading identically is worth no more than one of them, and the way that
// gets written is not a subtle one: a table of ten near-identical lines is
// where a copy-paste that never got edited survives, and where two keys wired
// to the same Key in Lang.Refusal's lookup look perfectly ordinary.
//
// Per enum rather than across both, which is the honest claim: a refusal and a
// closure are never drawn beside each other — one ends a join and the other
// ends a match — so two of them agreeing is not the same defect.
//
// What it sees: a duplicated wording, and a lookup pointing two values at one
// key. What it cannot see: two wordings that differ by a comma and say the same
// thing to a reader.
func TestNoTwoProtocolValuesShareAWording(t *testing.T) {
	for _, lang := range Langs() {
		said := make(map[string]string, wire.CodeCount)
		for value := range wire.CodeCount {
			name := wire.Code(value).String()
			worded := lang.Refusal(name)
			if first, taken := said[worded]; taken {
				t.Errorf("%v words the %q and %q refusals the same way: %q",
					lang, first, name, worded)
				continue
			}
			said[worded] = name
		}
		closed := make(map[string]string, wire.ClosureCount)
		for value := range wire.ClosureCount {
			name := wire.Closure(value).String()
			worded := lang.Closure(name)
			if first, taken := closed[worded]; taken {
				t.Errorf("%v words the %q and %q closures the same way: %q",
					lang, first, name, worded)
				continue
			}
			closed[worded] = name
		}
	}
}
