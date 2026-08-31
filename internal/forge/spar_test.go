package forge

import (
	"slices"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// sparSeeds is how many battles a test fights each pairing over.
//
// Small on purpose. Every property asserted here is exact — a mirror is even to
// the last part in a thousand, two characters agree about each other to the last
// part in a thousand — so nothing here needs the confidence a tuning run needs,
// and a test that fought a thousand seeds would only be a slower way to prove
// the same equality.
const sparSeeds = 20

// sparLibrary is the fixture cast plus the shipped one, which is what every test
// here measures against.
func sparLibrary(t *testing.T) *Library {
	t.Helper()
	lib, err := Load(scratchData(t))
	if err != nil {
		t.Fatalf("load the scratch library: %v", err)
	}
	return lib
}

// TestABothWaysMirrorIsExactlyEven is the test of the method rather than of a
// character, and it is the reason the spar fights every pairing twice.
//
// A character duelling an identical copy of itself has no advantage over it, so
// anything other than an even split is the measurement rather than the subject.
// One way round it is not even at all — the turn queue breaks a tie by placement
// — so this failing means the halves have stopped cancelling.
func TestABothWaysMirrorIsExactlyEven(t *testing.T) {
	lib := sparLibrary(t)
	for _, character := range lib.Characters().All() {
		report, err := lib.Spar(character.ID, progression.LevelCap, sparSeeds)
		if err != nil {
			t.Fatalf("spar %s: %v", character.ID, err)
		}
		mirror, found := mirrorRow(report)
		if !found {
			t.Fatalf("%s never met itself, so nothing controlled the measurement", character.ID)
		}
		if rate := mirror.Rate(); rate != scale.Base/2 {
			t.Errorf("%s against itself comes to %d rather than an even %d: %+v",
				character.ID, rate, scale.Base/2, mirror.Total())
		}
		// The stronger statement, and the one that says *why* it is even: the two
		// halves are the same battle with the sides swapped, so every win from
		// one slot is a loss from the other. A rate that came out even by two
		// errors cancelling would pass the check above and fail this one.
		if mirror.First.Wins != mirror.Second.Losses ||
			mirror.First.Losses != mirror.Second.Wins ||
			mirror.First.Draws != mirror.Second.Draws {
			t.Errorf("%s's two halves are not each other's reflection: first %+v, second %+v",
				character.ID, mirror.First, mirror.Second)
		}
	}
}

// TestTheFirstSlotIsWorthSomethingToCancel is what stops the test above from
// passing on a measurement that has nothing in it.
//
// If placement were worth nothing, fighting both ways would cancel nothing and
// an even mirror would prove only that zero plus zero is zero. So at least one
// character has to win from the first slot at a rate the second slot does not
// reach — and the report has to say by how much, because a reader comparing two
// rows needs to know how much of the gap between them a slot could account for.
func TestTheFirstSlotIsWorthSomethingToCancel(t *testing.T) {
	lib := sparLibrary(t)
	widest := 0
	for _, character := range lib.Characters().All() {
		report, err := lib.Spar(character.ID, progression.LevelCap, sparSeeds)
		if err != nil {
			t.Fatalf("spar %s: %v", character.ID, err)
		}
		if mirror, found := mirrorRow(report); found && abs(mirror.Edge()) > widest {
			widest = abs(mirror.Edge())
		}
	}
	if widest == 0 {
		t.Error("no character wins its mirror more often from one slot than the other, " +
			"so fighting both ways is cancelling nothing and the control row says nothing")
	}
}

// TestTwoCharactersAgreeAboutWhichOfThemIsBetter is the same balance seen from
// outside a single report.
//
// A's row for B and B's row for A are the same duels counted from opposite
// sides, so their rates have to add to a whole. They would not if either report
// favoured the slot it happened to place its own challenger in — which is the
// bug fighting both ways exists to make impossible, stated without reference to
// how it is implemented.
func TestTwoCharactersAgreeAboutWhichOfThemIsBetter(t *testing.T) {
	lib := sparLibrary(t)
	rates := map[string]map[string]int{}
	rows := map[string]map[string]Matchup{}
	for _, character := range lib.Characters().All() {
		report, err := lib.Spar(character.ID, progression.LevelCap, sparSeeds)
		if err != nil {
			t.Fatalf("spar %s: %v", character.ID, err)
		}
		against := map[string]int{}
		halves := map[string]Matchup{}
		for _, matchup := range report.Matchups {
			against[matchup.Against.ID] = matchup.Rate()
			halves[matchup.Against.ID] = matchup
		}
		rates[character.ID] = against
		rows[character.ID] = halves
	}
	for mine, against := range rates {
		for theirs, rate := range against {
			back, fought := rates[theirs][mine]
			if !fought {
				continue
			}
			if rate+back != scale.Base {
				t.Errorf("%s reports %d against %s while %s reports %d back, which comes to %d rather than %d",
					mine, rate, theirs, theirs, back, rate+back, scale.Base)
			}
		}
	}

	// And which half is which. A's battles from the first slot against B are B's
	// battles from the second slot against A — the same battles, counted from the
	// other side — so every win in one is a loss in the other. Nothing above
	// would notice if the two halves were filed the wrong way round, because
	// adding them up hides it; this is the assertion that looks at them apart.
	for mine, halves := range rows {
		for theirs, matchup := range halves {
			back, fought := rows[theirs][mine]
			if !fought || mine == theirs {
				continue
			}
			if matchup.First.Wins != back.Second.Losses || matchup.First.Losses != back.Second.Wins {
				t.Errorf("%s from the first slot against %s is %+v, and %s from the second slot against %s is %+v",
					mine, theirs, matchup.First, theirs, mine, back.Second)
			}
		}
	}
}

// TestASparFieldsTheKitItReports is the promise the screen makes when it prints
// a loadout above a win rate. A report of four skills beside figures produced by
// four others would be worse than no report at all.
func TestASparFieldsTheKitItReports(t *testing.T) {
	lib := sparLibrary(t)
	for _, character := range lib.Characters().All() {
		report, err := lib.Spar(character.ID, progression.LevelCap, sparSeeds)
		if err != nil {
			t.Fatalf("spar %s: %v", character.ID, err)
		}
		_, stage, err := character.Resolve(progression.LevelCap, progression.Furthest)
		if err != nil {
			t.Fatalf("resolve %s: %v", character.ID, err)
		}
		known := character.SkillsAt(progression.LevelCap, stage.Name)
		wanted := known
		if len(wanted) > cast.SkillSlots {
			wanted = wanted[:cast.SkillSlots]
		}
		if !slices.Equal(report.Challenger.Skills, wanted) {
			t.Errorf("%s fielded %v, and the first %d it declares are %v (of %v)",
				character.ID, report.Challenger.Skills, cast.SkillSlots, wanted, known)
		}
		if report.Challenger.Stage != stage.Name {
			t.Errorf("%s was fielded as %q rather than the furthest form %q",
				character.ID, report.Challenger.Stage, stage.Name)
		}
	}
}

// TestASparIsRepeatable is the engine's determinism seen from the tool: the same
// character over the same seeds is the same report, so a figure an author writes
// down stays true until they change the data.
func TestASparIsRepeatable(t *testing.T) {
	lib := sparLibrary(t)
	id := lib.Characters().All()[0].ID
	first, err := lib.Spar(id, progression.LevelCap, sparSeeds)
	if err != nil {
		t.Fatalf("spar %s: %v", id, err)
	}
	again, err := lib.Spar(id, progression.LevelCap, sparSeeds)
	if err != nil {
		t.Fatalf("spar %s a second time: %v", id, err)
	}
	if first.Rate() != again.Rate() || len(first.Matchups) != len(again.Matchups) {
		t.Fatalf("two runs of %s disagree: %d over %d rows, then %d over %d",
			id, first.Rate(), len(first.Matchups), again.Rate(), len(again.Matchups))
	}
	for i := range first.Matchups {
		if first.Matchups[i].Total() != again.Matchups[i].Total() {
			t.Errorf("%s against %s came to %+v, then %+v",
				id, first.Matchups[i].Against.ID, first.Matchups[i].Total(), again.Matchups[i].Total())
		}
	}
}

// TestTheHeadlineRateLeavesTheControlOut is why the mirror is a row rather than
// an opponent. It is even whatever the character is, so counting it drags every
// answer towards the middle — most of all the lopsided ones, which are the only
// answers worth having.
func TestTheHeadlineRateLeavesTheControlOut(t *testing.T) {
	lib := sparLibrary(t)
	for _, character := range lib.Characters().All() {
		report, err := lib.Spar(character.ID, progression.LevelCap, sparSeeds)
		if err != nil {
			t.Fatalf("spar %s: %v", character.ID, err)
		}
		everything := Tally{}
		for _, matchup := range report.Matchups {
			everything = everything.add(matchup.Total())
		}
		if report.Opponents() == 0 || report.Rate() == scale.Base/2 {
			// A character that is even against the cast is even with the control
			// folded in too, so it has nothing to say about which was counted.
			continue
		}
		if report.Rate() == everything.Rate() {
			t.Errorf("%s reports %d whether or not its own row is counted, so the control is in the headline",
				character.ID, report.Rate())
		}
		return
	}
	t.Skip("no character in the book is lopsided enough for the control to move its headline")
}

// TestNoKitIsUnaimableInADuel is the assumption that lets a refused pairing be
// an error rather than a row.
//
// ⚠️ It used to be TestTheDuelSlotAsksTheLeastOfAKit, and it asked a question
// the board no longer answers: which column demands the shortest range. Reach is
// depth into the enemy's half now, so no column demands more than another and
// the claim has to be made about the **kits** instead.
//
// A duel stands one unit on each side, so there is exactly one occupied rank to
// reach and it is at depth one. Every skill in the book is either aimed at its
// own caster — a cell that is always occupied — or declares a range of at least
// one, so no legal kit can be unaimable here, battle.New cannot refuse one
// pairing while accepting another, and what is left it can refuse is a fault in
// the books that every row would share.
//
// The day somebody writes a rule that makes a range of one insufficient, this is
// what says so, and whoever does it decides then whether a spar should carry a
// failed row again.
func TestNoKitIsUnaimableInADuel(t *testing.T) {
	lib := sparLibrary(t)
	for _, declared := range lib.Skills().Skills() {
		if declared.Target == skill.Self {
			continue
		}
		if declared.Range < 1 {
			t.Errorf("%s declares a range of %d, so a character holding only it could not "+
				"be measured from any slot", declared.ID, declared.Range)
		}
	}
}

// TestAnUnaimableKitIsRefusedRatherThanCounted is the other half: the branch
// above proves nothing can reach it, so this proves what it would do.
//
// A duellist with nothing to swing is not something Spar can build — a character
// that learns nothing at level one is refused when the book is parsed — so this
// goes at duel directly. What matters is that the refusal comes back whole
// rather than as a row of zeroes, which would read on screen as a pairing that
// was fought and never won.
func TestAnUnaimableKitIsRefusedRatherThanCounted(t *testing.T) {
	lib := sparLibrary(t)
	armed, err := lib.duellist(lib.Characters().All()[0], progression.LevelCap)
	if err != nil {
		t.Fatalf("field a character: %v", err)
	}
	unarmed := armed
	unarmed.ID, unarmed.Name, unarmed.Skills, unarmed.Passives = "empty", "Empty", nil, nil

	fought, err := duel(lib.Books(), armed, unarmed, sparSeeds, false, "")
	if err == nil {
		t.Fatalf("a duellist with no skill at all was counted as %+v", fought.Total())
	}
	if fought.Total().Battles() != 0 {
		t.Errorf("a refused pairing came back carrying %d battles", fought.Total().Battles())
	}
}

// TestARateCountsADrawAsHalfAndNeverCountsWhatDidNotEnd is the arithmetic on its
// own, away from any battle.
//
// The endless case is the one worth stating: a battle that hit the turn limit is
// the absence of a result, so it belongs in neither half of the fraction.
// Scoring it as a loss would let a pair that can never resolve read as a pair
// that reliably loses, which is the opposite of what it means.
func TestARateCountsADrawAsHalfAndNeverCountsWhatDidNotEnd(t *testing.T) {
	for _, test := range []struct {
		name  string
		tally Tally
		want  int
	}{
		{"every win", Tally{Wins: 8}, scale.Base},
		{"every loss", Tally{Losses: 8}, 0},
		{"every draw", Tally{Draws: 8}, scale.Base / 2},
		{"half and half", Tally{Wins: 4, Losses: 4}, scale.Base / 2},
		{"a draw is half a win", Tally{Wins: 3, Losses: 3, Draws: 2}, scale.Base / 2},
		{"an endless battle is not a loss", Tally{Wins: 4, Endless: 4}, scale.Base},
		{"nothing decided at all", Tally{Endless: 8}, 0},
		{"nothing at all", Tally{}, 0},
	} {
		if got := test.tally.Rate(); got != test.want {
			t.Errorf("%s: %+v came to %d, wanted %d", test.name, test.tally, got, test.want)
		}
	}
}

// TestAnEdgeIsPositiveWhenTheFirstSlotIsTheBetterOne.
//
// The sign is the whole of what an edge says, and it is the one thing adding the
// halves together cannot check: a report with the two halves filed the wrong way
// round has the same rates, the same totals, and every finding backwards.
func TestAnEdgeIsPositiveWhenTheFirstSlotIsTheBetterOne(t *testing.T) {
	for _, test := range []struct {
		name   string
		first  Tally
		second Tally
		want   int
	}{
		{"the first slot wins everything", Tally{Wins: 10}, Tally{Losses: 10}, scale.Base},
		{"the second slot wins everything", Tally{Losses: 10}, Tally{Wins: 10}, -scale.Base},
		{"the slot decides nothing", Tally{Wins: 5, Losses: 5}, Tally{Wins: 5, Losses: 5}, 0},
	} {
		matchup := Matchup{First: test.first, Second: test.second}
		if got := matchup.Edge(); got != test.want {
			t.Errorf("%s: the edge came to %d, wanted %d", test.name, got, test.want)
		}
	}
}

// TestTheMedianTurnCountIsTheMiddleOne, because a mean would be dragged by the
// one duel in fifty that went twenty rounds, and the number is on screen to say
// what a typical fight between these two feels like.
func TestTheMedianTurnCountIsTheMiddleOne(t *testing.T) {
	for _, test := range []struct {
		lengths []int
		want    int
	}{
		{nil, 0},
		{[]int{7}, 7},
		{[]int{9, 1, 5}, 5},
		{[]int{4, 2}, 2},
		{[]int{1, 2, 3, 400}, 2},
	} {
		if got := median(test.lengths); got != test.want {
			t.Errorf("the middle of %v is %d, and median said %d", test.lengths, test.want, got)
		}
	}
}

// TestASparRefusesWhatItCannotMeasure. Each of these would otherwise produce a
// report that looked like every other report and meant nothing.
func TestASparRefusesWhatItCannotMeasure(t *testing.T) {
	lib := sparLibrary(t)
	id := lib.Characters().All()[0].ID
	for _, test := range []struct {
		name  string
		id    string
		level int
		seeds int
		says  string
	}{
		{"no battles at all", id, progression.LevelCap, 0, "measures nothing"},
		{"a negative count", id, progression.LevelCap, -1, "measures nothing"},
		{"below the first level", id, 0, sparSeeds, "outside"},
		{"past the cap", id, progression.LevelCap + 1, sparSeeds, "outside"},
		{"nobody by that name", "nobody.at.all", progression.LevelCap, sparSeeds, "no character"},
	} {
		_, err := lib.Spar(test.id, test.level, test.seeds)
		if err == nil {
			t.Errorf("%s was accepted", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.says) {
			t.Errorf("%s said %q, which does not mention %q", test.name, err, test.says)
		}
	}
}

// mirrorRow is the row where the challenger met itself.
func mirrorRow(report SparReport) (Matchup, bool) {
	for _, matchup := range report.Matchups {
		if matchup.Mirror {
			return matchup, true
		}
	}
	return Matchup{}, false
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// TestAMatchupCountsTheChallengersStrikesAndNotTheOpponents is the assertion
// that catches the one bug shape a strike tally can have and never show.
//
// A Damaged event carries an Actor and **no Side at all**, and the challenger
// stands in the ally slot for one half of a matchup and the enemy slot for the
// other — so it is enlisted under one roster id in the first arrangement and the
// other in the second. A fold that read the wrong id would file the *opponent's*
// attacks under the challenger's name, and every column on the screen would look
// exactly as it looks now.
//
// It is asserted by counting a skill only one of the two brings. The challenger
// cannot have cast what it does not carry, so a count above nought against it is
// the opponent's, stated without any reference to how the fold is written.
func TestAMatchupCountsTheChallengersStrikesAndNotTheOpponents(t *testing.T) {
	lib := sparLibrary(t)
	adept, err := lib.duellist(mustCharacter(t, lib, "fixture-anime.adept"), progression.LevelCap)
	if err != nil {
		t.Fatalf("field the adept: %v", err)
	}
	sprout, err := lib.duellist(mustCharacter(t, lib, "fixture-game.sprout"), progression.LevelCap)
	if err != nil {
		t.Fatalf("field the sprout: %v", err)
	}
	mine, theirs := adept.Skills[0], sprout.Skills[0]
	if mine == theirs {
		t.Fatalf("both fixtures lead with %s, so nothing here can tell the two apart", mine)
	}
	if slices.Contains(adept.Skills, theirs) || slices.Contains(sprout.Skills, mine) {
		t.Fatalf("the two kits overlap on %s or %s, so a count cannot say whose it was", mine, theirs)
	}

	for _, test := range []struct {
		name                 string
		challenger, opponent Duellist
		ours, theirs         string
	}{
		{"the adept challenging", adept, sprout, mine, theirs},
		{"the sprout challenging", sprout, adept, theirs, mine},
	} {
		ours, err := duel(lib.Books(), test.challenger, test.opponent, sparSeeds, false, test.ours)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if ours.Strikes.Cast == 0 {
			t.Errorf("%s never cast %s, which it brings, so the tally is reading nobody",
				test.name, test.ours)
		}
		notOurs, err := duel(lib.Books(), test.challenger, test.opponent, sparSeeds, false, test.theirs)
		if err != nil {
			t.Fatalf("%s: %v", test.name, err)
		}
		if notOurs.Strikes.Cast != 0 || notOurs.Strikes.Landed != 0 {
			t.Errorf("%s was credited with %d cast(s) and %d landing(s) of %s, which only its opponent brings",
				test.name, notOurs.Strikes.Cast, notOurs.Strikes.Landed, test.theirs)
		}
	}
}

// TestCountingOneSkillAgreesWithCountingThemAll is the other half: the filter
// only ever narrows, and never drops or duplicates.
//
// The whole kit counted at once has to be the kit's skills counted one at a
// time and added up. A fold that let a reply, a summon's turn or a status tick
// through would break this in one direction, and one that dropped a strike would
// break it in the other.
func TestCountingOneSkillAgreesWithCountingThemAll(t *testing.T) {
	lib := sparLibrary(t)
	challenger, err := lib.duellist(mustCharacter(t, lib, "fixture-anime.adept"), progression.LevelCap)
	if err != nil {
		t.Fatalf("field the challenger: %v", err)
	}
	opponent, err := lib.duellist(mustCharacter(t, lib, "fixture-game.sprout"), progression.LevelCap)
	if err != nil {
		t.Fatalf("field the opponent: %v", err)
	}
	everything, err := duel(lib.Books(), challenger, opponent, sparSeeds, false, "")
	if err != nil {
		t.Fatalf("count them all: %v", err)
	}
	added := Strikes{}
	for _, held := range challenger.Skills {
		one, err := duel(lib.Books(), challenger, opponent, sparSeeds, false, held)
		if err != nil {
			t.Fatalf("count %s: %v", held, err)
		}
		added = added.add(one.Strikes)
	}
	if added != everything.Strikes {
		t.Errorf("the kit counted one skill at a time comes to %+v and counted at once to %+v",
			added, everything.Strikes)
	}
	if everything.Strikes.Cast == 0 {
		t.Error("nothing was cast at all, so this test compared two empty tallies")
	}
}

// TestADuellistTakesTheFieldUnderItsFormsName is the spar's copy of the rule a
// placement and a roster entry both obey: what the engine is handed is a form's
// stat line, so the name the log prints beside those numbers has to be the
// form's. A duel naming the character puts a first form's name on a last form's
// health, which is the one pairing a reader cannot tell from the real thing.
//
// A Duellist carries both names because a spar's own report has a column for
// each. That is the report's business; only one of the two may reach a battle.
func TestADuellistTakesTheFieldUnderItsFormsName(t *testing.T) {
	lib := sparLibrary(t)
	evolved := 0
	for _, character := range lib.Characters().All() {
		who, err := lib.duellist(character, progression.LevelCap)
		if err != nil {
			t.Fatalf("resolve %s: %v", character.ID, err)
		}
		if who.Stage == "" {
			t.Fatalf("%s resolves to no form, so a duel has no name to field it under", character.ID)
		}
		fielded := place("a", who, hex.SideAlly)
		if fielded.Name != who.Stage {
			t.Errorf("%s takes the field as %q, want the form it fields, %q",
				character.ID, fielded.Name, who.Stage)
		}
		if who.Stage != who.Name {
			evolved++
		}
	}
	// Every character in the fixture whose last form is its own name proves
	// nothing above: the two names it could have picked are one string. At least
	// one has to differ, or this passes on the behaviour it exists to refuse.
	if evolved == 0 {
		t.Error("no character in the library reaches a form named differently from itself, " +
			"so naming the form and naming the character cannot be told apart here")
	}
}
