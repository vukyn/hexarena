package composition_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/status"
)

func statuses(t *testing.T) *status.Book {
	t.Helper()
	book, err := status.ParseBook([]byte(`{
	  "max_stacks": 5, "max_duration": 6,
	  "kinds": [
	    {"id": "kinship", "category": "buff", "permanent": true, "max_stacks": 2,
	     "modifiers": [{"target": "attack", "mode": "percent", "amount": 100}]},
	    {"id": "resolve", "category": "buff", "permanent": true, "max_stacks": 1,
	     "modifiers": [{"target": "defense", "mode": "percent", "amount": 100}]},
	    {"id": "fury", "category": "buff", "max_stacks": 3, "duration": 3,
	     "modifiers": [{"target": "attack", "mode": "percent", "amount": 300}]}
	  ]
	}`))
	if err != nil {
		t.Fatalf("parse the status fixture: %v", err)
	}
	return book
}

func chart(t *testing.T) *element.Chart {
	t.Helper()
	parsed, err := element.ParseChart([]byte(`{
	  "multipliers": {"advantage": 1500, "neutral": 1000, "disadvantage": 667},
	  "cycles": [
	    {"name": "organic", "chain": ["water", "fire", "grass", "ground"]},
	    {"name": "industrial", "chain": ["ice", "metal", "wind", "electric"]}
	  ],
	  "mutual": [["light", "dark"]],
	  "inert": ["neutral"]
	}`))
	if err != nil {
		t.Fatalf("parse the chart fixture: %v", err)
	}
	return parsed
}

func parse(t *testing.T, body string) (*composition.Book, error) {
	t.Helper()
	return composition.ParseBook([]byte(body), composition.Deps{Statuses: statuses(t), Chart: chart(t)})
}

func mustParse(t *testing.T, body string) *composition.Book {
	t.Helper()
	book, err := parse(t, body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return book
}

// theSharedElement is the shape the shipped bonus has: two rungs, sharers only.
const theSharedElement = `{"bonuses": [
  {"id": "kin", "name": "đồng hệ", "axis": "element", "scope": "sharers", "rungs": [
    {"at": 2, "grants": [{"status": "kinship", "stacks": 1}]},
    {"at": 3, "grants": [{"status": "kinship", "stacks": 2}]}
  ]}
]}`

func member(t *testing.T, id string, elements ...string) composition.Member {
	t.Helper()
	parsed := make([]element.Element, 0, len(elements))
	for _, name := range elements {
		one, err := element.Parse(name)
		if err != nil {
			t.Fatalf("parse the element %q: %v", name, err)
		}
		parsed = append(parsed, one)
	}
	switch len(parsed) {
	case 1:
		affinity, err := element.Single(parsed[0])
		if err != nil {
			t.Fatalf("build a single affinity: %v", err)
		}
		return composition.Member{ID: id, Affinity: affinity}
	case 2:
		affinity, err := element.Dual(parsed[0], parsed[1])
		if err != nil {
			t.Fatalf("build a dual affinity: %v", err)
		}
		return composition.Member{ID: id, Affinity: affinity}
	}
	t.Fatalf("a member carries one or two elements, not %d", len(parsed))
	return composition.Member{}
}

func awarded(awards []composition.Award, unit string) []composition.Award {
	var mine []composition.Award
	for _, award := range awards {
		if award.Unit == unit {
			mine = append(mine, award)
		}
	}
	return mine
}

// TestARungIsALadderRatherThanASet is the arithmetic the whole table rests on: a
// count reaches the highest rung it satisfies and that one only. Read
// cumulatively, three sharers would take rung two's stack *and* rung three's,
// which makes the top rung worth the sum of a table nobody wrote down — and it
// would read as working, because the figure only ever goes up.
func TestARungIsALadderRatherThanASet(t *testing.T) {
	book := mustParse(t, theSharedElement)
	kin, err := book.Lookup("kin")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		count  int
		at     int
		stacks int
		fires  bool
	}{
		{count: 0, fires: false},
		{count: 1, fires: false},
		{count: 2, at: 2, stacks: 1, fires: true},
		{count: 3, at: 3, stacks: 2, fires: true},
		{count: 5, at: 3, stacks: 2, fires: true},
	} {
		rung, reached := kin.Reached(want.count)
		if reached != want.fires {
			t.Fatalf("a count of %d reached %v, wanted %v", want.count, reached, want.fires)
		}
		if !reached {
			continue
		}
		if rung.At != want.at {
			t.Errorf("a count of %d landed on the rung at %d, wanted the one at %d", want.count, rung.At, want.at)
		}
		if len(rung.Grants) != 1 || rung.Grants[0].Stacks != want.stacks {
			t.Errorf("a count of %d granted %v, wanted %d stack(s) of one status",
				want.count, rung.Grants, want.stacks)
		}
	}
	if top := kin.Top(); top != 3 {
		t.Errorf("the top rung reads %d, wanted 3", top)
	}
}

// TestOnlyTheSharersAreAwardedUnderTheSharersScope holds the half of decision 3
// that a screen has to be able to draw: a sharers-only bonus lands on the units
// that carry the value and on nobody else, even though the whole side brought it.
func TestOnlyTheSharersAreAwardedUnderTheSharersScope(t *testing.T) {
	book := mustParse(t, theSharedElement)
	awards := book.Awards(chart(t), []composition.Member{
		member(t, "a", "water"),
		member(t, "b", "fire"),
		member(t, "c", "water"),
	})
	if len(awards) != 2 {
		t.Fatalf("two of three share water and %d awards came back: %v", len(awards), awards)
	}
	for _, award := range awards {
		if award.Value != "water" || award.Count != 2 || award.Scope != composition.ScopeSharers {
			t.Errorf("award %+v does not say what fired", award)
		}
	}
	if got := awarded(awards, "b"); len(got) != 0 {
		t.Errorf("the fire unit was awarded %v, and it shares nothing", got)
	}
	if got := awarded(awards, "a"); len(got) != 1 {
		t.Errorf("a water unit took %d awards, wanted 1", len(got))
	}
}

// TestASquadScopedBonusReachesTheUnitsThatShareNothing is the other kind, and
// the pair of them is why Scope exists at all: the same count, the same rung,
// and a different set of units receiving it.
func TestASquadScopedBonusReachesTheUnitsThatShareNothing(t *testing.T) {
	book := mustParse(t, `{"bonuses": [
	  {"id": "band", "axis": "element", "scope": "squad", "rungs": [
	    {"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}
	  ]}
	]}`)
	members := []composition.Member{
		member(t, "a", "water"),
		member(t, "b", "fire"),
		member(t, "c", "water"),
	}
	awards := book.Awards(chart(t), members)
	if len(awards) != 3 {
		t.Fatalf("a squad-wide rung awarded %d of 3 units: %v", len(awards), awards)
	}
	if got := awarded(awards, "b"); len(got) != 1 || got[0].Value != "water" {
		t.Errorf("the unit sharing nothing took %v, and a squad-wide bonus is the side's", got)
	}
}

// TestADualAffinityCountsTowardBothHalves is decision 2, and it is the first
// thing in the game that pays a dual for being one.
func TestADualAffinityCountsTowardBothHalves(t *testing.T) {
	book := mustParse(t, theSharedElement)
	awards := book.Awards(chart(t), []composition.Member{
		member(t, "lapras", "water", "ice"),
		member(t, "squirtle", "water"),
		member(t, "magnemite", "electric", "metal"),
	})
	// water is shared by two; ice, electric and metal are carried alone.
	if len(awards) != 2 {
		t.Fatalf("water is the only shared element and %d awards came back: %v", len(awards), awards)
	}
	for _, award := range awards {
		if award.Value != "water" {
			t.Errorf("award %+v fired on something no two units share", award)
		}
	}
	// And a dual sharing both halves takes one award per half rather than one
	// for being a dual: two bonuses' worth for a unit that is glue twice.
	both := book.Awards(chart(t), []composition.Member{
		member(t, "lapras", "water", "ice"),
		member(t, "other", "water", "ice"),
	})
	if len(both) != 4 {
		t.Fatalf("two duals sharing both halves took %d awards, wanted 4: %v", len(both), both)
	}
	values := []string{}
	for _, award := range awarded(both, "lapras") {
		values = append(values, award.Value)
	}
	slices.Sort(values)
	if !slices.Equal(values, []string{"ice", "water"}) {
		t.Errorf("the dual was awarded for %v, wanted both of its halves", values)
	}
}

// TestAnInertElementFormsNoTribe is the one exclusion in the counting rule, and
// it is read off the chart rather than written down here: sharing the element
// with no strengths and no weaknesses is sharing the absence of one.
func TestAnInertElementFormsNoTribe(t *testing.T) {
	book := mustParse(t, theSharedElement)
	awards := book.Awards(chart(t), []composition.Member{
		member(t, "a", "neutral"),
		member(t, "b", "neutral"),
		member(t, "c", "neutral"),
	})
	if len(awards) != 0 {
		t.Fatalf("three unaligned units were awarded %v", awards)
	}
	// The same three with a real element between two of them do fire, so what is
	// being measured is the inertness rather than the counting.
	awards = book.Awards(chart(t), []composition.Member{
		member(t, "a", "neutral"),
		member(t, "b", "dark"),
		member(t, "c", "dark"),
	})
	if len(awards) != 2 {
		t.Fatalf("two units of dark took %d awards, wanted 2: %v", len(awards), awards)
	}
}

// TestTheAwardsAreOrderedByTheRosterAndTheBook is the determinism claim, and it
// is asserted by repetition because a map walk is only wrong sometimes. Go
// randomises that order per range, so a rule built on one would pass a single
// reading and diverge on a later replay of the same seed.
func TestTheAwardsAreOrderedByTheRosterAndTheBook(t *testing.T) {
	book := mustParse(t, `{"bonuses": [
	  {"id": "kin", "axis": "element", "scope": "sharers", "rungs": [
	    {"at": 2, "grants": [{"status": "kinship", "stacks": 1}]}
	  ]},
	  {"id": "band", "axis": "element", "scope": "squad", "rungs": [
	    {"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}
	  ]}
	]}`)
	members := []composition.Member{
		member(t, "one", "fire", "dark"),
		member(t, "two", "dark"),
		member(t, "three", "fire"),
	}
	first := book.Awards(chart(t), members)
	if len(first) == 0 {
		t.Fatal("the fixture awarded nothing, so it measures no order")
	}
	shape := func(awards []composition.Award) string {
		var out []string
		for _, award := range awards {
			out = append(out, award.Bonus+"/"+award.Value+"/"+award.Unit)
		}
		return strings.Join(out, " ")
	}
	want := shape(first)
	for range 40 {
		if got := shape(book.Awards(chart(t), members)); got != want {
			t.Fatalf("the same roster came back in a different order:\n%s\n%s", want, got)
		}
	}
	// Declaration order outer, first-appearance order of the value inner: the
	// fire tribe is met before the dark one because the first member carries
	// fire first.
	if !strings.HasPrefix(want, "kin/fire/one kin/fire/three kin/dark/one") {
		t.Errorf("the order is not the book's then the roster's: %s", want)
	}
}

// TestWithoutIsTheOnlyThingThatCanPriceARung holds the instrument: the same
// members and the same book, one bonus gone and the others left standing.
func TestWithoutIsTheOnlyThingThatCanPriceARung(t *testing.T) {
	book := mustParse(t, `{"bonuses": [
	  {"id": "kin", "axis": "element", "scope": "sharers", "rungs": [
	    {"at": 2, "grants": [{"status": "kinship", "stacks": 1}]}
	  ]},
	  {"id": "band", "axis": "element", "scope": "squad", "rungs": [
	    {"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}
	  ]}
	]}`)
	members := []composition.Member{member(t, "a", "water"), member(t, "b", "water")}
	if got := len(book.Awards(chart(t), members)); got != 4 {
		t.Fatalf("both bonuses on a sharing pair came to %d awards, wanted 4", got)
	}
	without := book.Without("kin")
	awards := without.Awards(chart(t), members)
	if len(awards) != 2 {
		t.Fatalf("one bonus taken out left %d awards, wanted 2: %v", len(awards), awards)
	}
	for _, award := range awards {
		if award.Bonus != "band" {
			t.Errorf("the disabled bonus still fired: %+v", award)
		}
	}
	// ⚠️ The book it was taken out of is untouched, so a measurement cannot
	// disable a bonus for the rest of the process by taking one reading.
	if got := len(book.Awards(chart(t), members)); got != 4 {
		t.Errorf("Without mutated the book it was called on: %d awards left", got)
	}
	if got := len(book.Without("nobody").Awards(chart(t), members)); got != 4 {
		t.Errorf("naming an undeclared bonus changed the awards: %d", got)
	}
	if all := without.All(); len(all) != 1 || all[0].ID != "band" {
		t.Errorf("the filtered book holds %v", all)
	}
}

// TestANilBookAwardsNothing is what lets a battle run without the file, the way
// one runs without the passive book when no unit names a trait.
func TestANilBookAwardsNothing(t *testing.T) {
	var book *composition.Book
	if got := book.Awards(chart(t), []composition.Member{member(t, "a", "water"), member(t, "b", "water")}); got != nil {
		t.Errorf("a nil book awarded %v", got)
	}
	if got := book.All(); got != nil {
		t.Errorf("a nil book holds %v", got)
	}
	if _, err := book.Lookup("kin"); err == nil {
		t.Error("a nil book found a bonus")
	}
	if got := book.Without("kin"); got != nil {
		t.Errorf("filtering a nil book made %v", got)
	}
	// A book with no chart cannot count, and answering nothing is better than
	// answering as though every element were inert.
	real := mustParse(t, theSharedElement)
	if got := real.Awards(nil, []composition.Member{member(t, "a", "water"), member(t, "b", "water")}); got != nil {
		t.Errorf("counting without a chart awarded %v", got)
	}
}

func TestParseBookAcceptsWhatItShould(t *testing.T) {
	book := mustParse(t, theSharedElement)
	all := book.All()
	if len(all) != 1 {
		t.Fatalf("the fixture declared one bonus and %d came back", len(all))
	}
	kin := all[0]
	if kin.ID != "kin" || kin.Name != "đồng hệ" {
		t.Errorf("the bonus reads %q / %q", kin.ID, kin.Name)
	}
	if kin.Axis != composition.AxisElement || kin.Scope != composition.ScopeSharers {
		t.Errorf("the bonus counts %s for %s", kin.Axis, kin.Scope)
	}
	if len(kin.Rungs) != 2 || kin.Rungs[0].At != 2 || kin.Rungs[1].At != 3 {
		t.Errorf("the rungs read %v", kin.Rungs)
	}
	// An empty file is a legal file: the game runs with no bonus declared, which
	// is what every battle before this one did.
	empty, err := parse(t, `{"bonuses": []}`)
	if err != nil {
		t.Fatalf("an empty book was refused: %v", err)
	}
	if got := empty.All(); len(got) != 0 {
		t.Errorf("an empty book holds %v", got)
	}
}

func TestParseBookRejects(t *testing.T) {
	for _, refusal := range []struct {
		name, body, wants string
	}{
		{
			name:  "a bonus with no id",
			body:  `{"bonuses": [{"axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}]}]}`,
			wants: "needs an id",
		},
		{
			name: "two bonuses of one id",
			body: `{"bonuses": [
			  {"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}]},
			  {"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 3, "grants": [{"status": "resolve", "stacks": 1}]}]}
			]}`,
			wants: "chooses neither",
		},
		{
			name:  "an axis nobody counts",
			body:  `{"bonuses": [{"id": "kin", "axis": "origin", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}]}]}`,
			wants: "no axis is called",
		},
		{
			name:  "no axis at all",
			body:  `{"bonuses": [{"id": "kin", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}]}]}`,
			wants: "no axis is called",
		},
		{
			name:  "a scope nobody hands out",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "everyone", "rungs": [{"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}]}]}`,
			wants: "no scope is called",
		},
		{
			name:  "a bonus with no rung",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": []}]}`,
			wants: "declares no rung",
		},
		{
			name:  "a rung of one",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 1, "grants": [{"status": "resolve", "stacks": 1}]}]}]}`,
			wants: "one unit shares nothing",
		},
		{
			name:  "a rung no side can reach",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 6, "grants": [{"status": "resolve", "stacks": 1}]}]}]}`,
			wants: "which no side of",
		},
		{
			name: "rungs written downwards",
			body: `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [
			  {"at": 3, "grants": [{"status": "resolve", "stacks": 1}]},
			  {"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}
			]}]}`,
			wants: "out of order",
		},
		{
			name: "one rung twice",
			body: `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [
			  {"at": 2, "grants": [{"status": "resolve", "stacks": 1}]},
			  {"at": 2, "grants": [{"status": "resolve", "stacks": 1}]}
			]}]}`,
			wants: "out of order",
		},
		{
			name:  "a rung that grants nothing",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": []}]}]}`,
			wants: "grants nothing",
		},
		{
			name:  "a status nobody declared",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "nowhere", "stacks": 1}]}]}]}`,
			wants: "nowhere",
		},
		{
			name:  "a status that expires",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "fury", "stacks": 1}]}]}]}`,
			wants: "not permanent",
		},
		{
			name:  "a grant of no stacks",
			body:  `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": [{"status": "resolve"}]}]}]}`,
			wants: "0 times",
		},
		{
			name: "one status twice in a rung",
			body: `{"bonuses": [{"id": "kin", "axis": "element", "scope": "squad", "rungs": [{"at": 2, "grants": [
			  {"status": "resolve", "stacks": 1}, {"status": "resolve", "stacks": 1}
			]}]}]}`,
			wants: "twice",
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			_, err := parse(t, refusal.body)
			if err == nil {
				t.Fatalf("%s was accepted", refusal.name)
			}
			if !strings.Contains(err.Error(), refusal.wants) {
				t.Errorf("the refusal reads %q, wanted it to mention %q", err, refusal.wants)
			}
		})
	}
}

// TestParsingNeedsBothBooks is the cross-book rule, and it refuses up front
// rather than at the first grant: a book parsed without the status book could
// only discover a bad name for the statuses somebody happens to have declared.
func TestParsingNeedsBothBooks(t *testing.T) {
	if _, err := composition.ParseBook([]byte(theSharedElement), composition.Deps{Chart: chart(t)}); err == nil {
		t.Error("a book parsed with no status book")
	}
	if _, err := composition.ParseBook([]byte(theSharedElement), composition.Deps{Statuses: statuses(t)}); err == nil {
		t.Error("a book parsed with no element chart")
	}
}
