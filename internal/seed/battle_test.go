package seed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/testfixture"
)

// benchBooks is the shipped books with the bench's skill book in place of the
// shipped one.
//
// Only the skills are swapped. Every other book -- the chart, the rules, the
// bounds, the limits, the shapes, the statuses -- is a shipped constant that the
// cast being re-authored cannot touch, so an engine test should keep reading the
// real ones. What a four-skill cast cannot do is reach every mechanism, and that
// is what the bench is for.
func benchBooks(t *testing.T) battle.Books {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	patterns, err := seed.PatternBook()
	if err != nil {
		t.Fatalf("load shapes: %v", err)
	}
	statuses, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load statuses: %v", err)
	}
	parsed, err := skill.ParseBook([]byte(`{"skills":`+testfixture.Skills+`}`),
		skill.Deps{Patterns: patterns, Statuses: statuses})
	if err != nil {
		t.Fatalf("parse the bench skills: %v", err)
	}
	books.Skills = parsed
	return books
}

// benchRoster is the ten-unit bench: two full teams reaching every element.
func benchRoster(t *testing.T) []battle.Roster {
	t.Helper()
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	// The bench roster is the flat form, so the cast book is never consulted --
	// but ParseRoster will not take a nil one, and refusing it is right: a
	// reference that silently resolved to nothing would be worse.
	roster, err := seed.ParseRoster([]byte(testfixture.Roster), characters)
	if err != nil {
		t.Fatalf("parse the bench roster: %v", err)
	}
	return roster
}

// benchBattle is a battle on the bench, which is what an engine test wants: the
// shipped cast is one character and cannot emit half the events the engine can.
func benchBattle(t *testing.T, seedValue uint64) *battle.Battle {
	t.Helper()
	fight, err := battle.New(benchBooks(t), seedValue, benchRoster(t))
	if err != nil {
		t.Fatalf("seed %d: %v", seedValue, err)
	}
	return fight
}

func TestShippedRosterIsUsable(t *testing.T) {
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load roster: %v", err)
	}
	// Not a count: the shipped roster is whatever the cast currently supports,
	// and it grows as characters are authored. What has to hold is that it is a
	// battle -- both sides present, inside the team limit, and legal to enlist.
	if len(roster) == 0 {
		t.Fatal("the shipped roster is empty, so the game has nothing to play")
	}
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load books: %v", err)
	}
	perSide := map[hex.Side]int{}
	elements := map[string]bool{}
	for _, unit := range roster {
		perSide[unit.Side]++
		elements[unit.Affinity.String()] = true
		if err := books.Chart.ValidateAffinity(unit.Affinity); err != nil {
			t.Errorf("unit %q: %v", unit.ID, err)
		}
		if err := books.Limits.CheckValues(unit.Stats, books.Rules); err != nil {
			t.Errorf("unit %q: %v", unit.ID, err)
		}
		if len(unit.Skills) < 3 {
			t.Errorf("unit %q knows only %d skills, too few to make a turn a decision", unit.ID, len(unit.Skills))
		}
	}
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		// A bound, not a count: the shipped roster is the cast, and it grows as
		// characters are authored. A full team is what the board allows, not
		// what the game currently holds.
		if perSide[side] < 1 || perSide[side] > hex.MaxTeamSize {
			t.Errorf("the %s side has %d units, want between 1 and %d",
				side, perSide[side], hex.MaxTeamSize)
		}
	}
	// Reaching every corner of the chart is a property of the bench, which
	// exists for exactly that. What the shipped roster owes is that the chart
	// decides something at all: a roster on one affinity would never consult it,
	// and consulting it is most of what a battle is doing.
	if len(elements) < 2 {
		t.Errorf("the roster declares %d affinities, so the element chart decides nothing", len(elements))
	}
}

// TestTheShippedRosterIsNotAMirror is the property that makes the roster a
// measuring instrument rather than a scenario.
//
// It used to be the same character three times on each side. A mirror cannot
// measure anything: a change to a number helps both squads by exactly as much,
// so the win rate moves only by noise, which is what stopped razor_leaf's
// piercing value from being judged by anything but its damage table.
//
// The check is against the resolved units rather than the authoring form,
// because that is where the property has to hold. A species and a level resolve
// to a name, a stat line and a kit; two units agreeing on all three are the same
// unit however they were written down.
func TestTheShippedRosterIsNotAMirror(t *testing.T) {
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load roster: %v", err)
	}
	twinned := 0
	for _, ally := range roster {
		if ally.Side != hex.SideAlly {
			continue
		}
		for _, foe := range roster {
			if foe.Side != hex.SideEnemy {
				continue
			}
			if ally.Name != foe.Name || ally.Stats != foe.Stats {
				continue
			}
			twinned++
			t.Errorf("%q and %q are the same unit on both sides: %s at %s",
				ally.ID, foe.ID, ally.Name, ally.Stats)
		}
	}
	if twinned > 0 {
		t.Log("a shared unit is a number that weighs the same on both sides, which is a number nothing can measure")
	}
}

// TestEveryShippedUnitCanReachEveryEnemy is what keeps the roster from stalling.
//
// battle.New only refuses a unit that can reach *nobody*, which is the rule a
// game needs: a short-ranged unit behind the front line is a legitimate design,
// because the squad in front of it is what reaches. The seed roster is held to
// the stricter rule, because it is an instrument and a battle that cannot finish
// measures nothing.
//
// It is not hypothetical. An earlier draft of this roster stood its third unit
// on slot 1,2, which hex.Place puts **four** cells from the enemy's own 1,2 —
// past every range in the cast. Five seeds in four thousand ended with those two
// alive and unable to touch each other, and it was not even a draw: one of them
// kept refreshing a regeneration on itself, so something was always pending and
// the board was never final. Those battles ran the turn limit out.
func TestEveryShippedUnitCanReachEveryEnemy(t *testing.T) {
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load roster: %v", err)
	}
	skills, err := seed.SkillBook()
	if err != nil {
		t.Fatalf("load skills: %v", err)
	}
	longest := func(unit battle.Roster) int {
		out := 0
		for _, id := range unit.Skills {
			carried, err := skills.Lookup(id)
			if err != nil {
				t.Fatalf("unit %q: %v", unit.ID, err)
			}
			if carried.Range > out {
				out = carried.Range
			}
		}
		return out
	}
	for _, unit := range roster {
		reach := longest(unit)
		from := hex.Place(unit.Side, unit.Slot)
		for _, other := range roster {
			if other.Side == unit.Side {
				continue
			}
			distance := from.DistanceTo(hex.Place(other.Side, other.Slot))
			if distance <= reach {
				continue
			}
			t.Errorf("%q at %s reaches %d, but %q stands %d cells away",
				unit.ID, from, reach, other.ID, distance)
		}
	}
}

// TestSeedBattlesFinishFromEverySeed is the end to end check: the whole engine
// assembled from the shipped data plays a battle out without stalling, whatever
// the rolls are.
func TestSeedBattlesFinishFromEverySeed(t *testing.T) {
	const turnLimit = 4000
	longest, winners := 0, map[hex.Side]int{}
	for seedValue := uint64(0); seedValue < 40; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: assemble: %v", seedValue, err)
		}
		fight.Begin()
		turns, err := fight.RunToEnd(turnLimit)
		if err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		if !fight.Finished() {
			t.Fatalf("seed %d did not finish in %d turns", seedValue, turns)
		}
		winner, decided := fight.Winner()
		if !decided {
			t.Errorf("seed %d ended without a winner", seedValue)
		}
		winners[winner]++
		if turns > longest {
			longest = turns
		}
		// Every unit that is not on the winning side must actually be dead.
		for _, unit := range fight.Units() {
			if unit.Side != winner && !unit.Dead {
				t.Errorf("seed %d: %q survived on the losing side", seedValue, unit.ID)
			}
			if unit.Dead && unit.HP != 0 {
				t.Errorf("seed %d: dead unit %q has %d health", seedValue, unit.ID, unit.HP)
			}
		}
	}
	t.Logf("longest battle %d turns, wins %v", longest, winners)
	// Both sides should be able to win. A roster where one side always loses is
	// not a test bed, it is a scripted defeat.
	for _, side := range []hex.Side{hex.SideAlly, hex.SideEnemy} {
		if winners[side] == 0 {
			t.Errorf("the %s side never won across 40 seeds", side)
		}
	}
}

func TestSeedBattleReplaysExactly(t *testing.T) {
	run := func() []battle.Event {
		fight, err := seed.NewBattle(99)
		if err != nil {
			t.Fatalf("assemble: %v", err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("run: %v", err)
		}
		return fight.Drain()
	}
	first, second := run(), run()
	if len(first) != len(second) {
		t.Fatalf("the two runs produced %d and %d events", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("event %d differs:\n%+v\n%+v", i, first[i], second[i])
		}
	}
}

// TestNoEventNamesBothASkillAndATrait is the encoding a renderer reads a reply
// by, asserted where it can actually be broken.
//
// A reply is damage that names a trait instead of a skill, and everything that
// draws one — the terminal client, the replay report, a future graphical client
// — decides which sentence to write by asking which of the two fields is set. If
// an event ever carried both, every one of those readers would pick whichever it
// happened to test first, and they would not all pick the same.
//
// The sweep is the shipped roster rather than the bench, because the shipped
// roster is the one holding a trait that replies.
func TestNoEventNamesBothASkillAndATrait(t *testing.T) {
	for seedValue := uint64(0); seedValue < 40; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		for _, event := range fight.Drain() {
			if event.Skill != "" && event.Passive != "" {
				t.Fatalf("seed %d: %s names the skill %q and the trait %q at once: %+v",
					seedValue, event.Kind, event.Skill, event.Passive, event)
			}
		}
	}
}

// TestTheShippedRosterAnswersItsAttackers is the shipped trait doing its job in
// the shipped game, which is a different question from whether the mechanism
// works: a reply nobody holds is a feature nobody has.
func TestTheShippedRosterAnswersItsAttackers(t *testing.T) {
	answered, poisoned := 0, 0
	for seedValue := uint64(0); seedValue < 40; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		for _, event := range fight.Drain() {
			if event.Passive == "" {
				continue
			}
			switch event.Kind {
			case battle.Damaged:
				answered++
			case battle.StatusApplied:
				poisoned++
			}
		}
	}
	if answered == 0 {
		t.Error("nothing in the shipped roster ever answered an attacker")
	}
	// The poison is the other half of the trait's name, and it is authored at a
	// low enough chance that forty battles is the smallest sweep that reliably
	// sees one. A count rather than a rate: what is being checked is that the
	// path exists, not how often it fires.
	if poisoned == 0 {
		t.Error("no reply in the shipped roster ever landed its status")
	}
}

// TestEveryEventKindIsReachable stops a kind being declared and never emitted,
// which would mean a renderer had a case nobody could test.
//
// Sixty battles on autopilot, and then one played by hand, because autopilot is
// not the whole game and one kind is only reachable off it — see
// aHandPlayedGateCrossing.
func TestEveryEventKindIsReachable(t *testing.T) {
	seen := make(map[battle.Kind]bool, battle.KindCount)
	record := func(events []battle.Event) {
		for _, event := range events {
			seen[event.Kind] = true
		}
	}
	for seedValue := uint64(0); seedValue < 60; seedValue++ {
		fight := benchBattle(t, seedValue)
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		record(fight.Drain())
	}
	record(aHandPlayedGateCrossing(t))
	for kind := 0; kind < battle.KindCount; kind++ {
		if !seen[battle.Kind(kind)] {
			t.Errorf("no battle on the bench ever emitted %s", battle.Kind(kind))
		}
	}
}

// aHandPlayedGateCrossing drives one battle by hand until a gated trait has come
// on and gone off again, and returns what it logged.
//
// It is here rather than folded into the sweep above because of something the
// sweep measured: on autopilot a gated trait is very nearly a one way door. A
// unit that falls below a third of its health is a unit that is losing, and the
// opponent never buffs, never cleanses and never heals anybody — the only
// healing in sixty battles is what a drain returns to its own caster, which is
// worth about a fortieth of a health bar against damage worth a tenth. Across
// four thousand battle-seeds of every arrangement tried, a trait came back off
// once.
//
// So passive_released is reachable, and reachable is what this test is about: a
// player heals, and Suggest does not. Proving it with a hand-played battle says
// exactly that, where widening the sweep until the rare case turned up would
// have been a test that passed for a reason nobody could name.
func aHandPlayedGateCrossing(t *testing.T) []battle.Event {
	t.Helper()
	fight, err := battle.New(benchBooks(t), 3, []battle.Roster{
		{ID: "holder", Side: hex.SideAlly, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "fire"), Stats: benchStats(2000, 700, 300, 120),
			Skills: []string{"strike", "siphon"}, Passives: []string{"blaze"}},
		{ID: "sparring", Side: hex.SideEnemy, Slot: hex.Offset{Col: 2, Row: 1},
			Affinity: mustAffinity(t, "metal"), Stats: benchStats(4800, 500, 300, 100),
			Skills: []string{"strike"}},
	})
	if err != nil {
		t.Fatalf("new battle: %v", err)
	}
	fight.Begin()
	var events []battle.Event
	held, released := false, false
	// A generous bound rather than a tuned one: what the loop is waiting for is a
	// pair of events, and it stops on them.
	for turn := 0; turn < 400 && !released; turn++ {
		prompt, err := fight.Advance()
		if err != nil {
			t.Fatalf("advance: %v", err)
		}
		if prompt == nil {
			break
		}
		if !prompt.Skipped {
			// Trade until the trait is on, then drink the way back up. A player
			// choosing between the two is the whole of what autopilot will not
			// do.
			want := "strike"
			if prompt.Unit == "holder" && held {
				want = "siphon"
			}
			acted := false
			for _, option := range prompt.Options {
				if option.Skill != want || !option.Available() {
					continue
				}
				if err := fight.Act(option.Skill, option.Aims[0]); err != nil {
					t.Fatalf("act %s: %v", option.Skill, err)
				}
				acted = true
				break
			}
			if !acted {
				if err := fight.Pass("waiting"); err != nil {
					t.Fatalf("pass: %v", err)
				}
			}
		}
		for _, event := range fight.Drain() {
			events = append(events, event)
			switch event.Kind {
			case battle.PassiveHeld:
				if event.Passive == "blaze" {
					held = true
				}
			case battle.PassiveReleased:
				released = true
			}
		}
		if fight.Finished() {
			break
		}
	}
	if !held {
		t.Fatal("the hand-played battle never got its holder below the gate")
	}
	if !released {
		t.Fatal("the hand-played battle never healed its holder back over the gate")
	}
	return events
}

func mustAffinity(t *testing.T, id string) element.Affinity {
	t.Helper()
	member, err := element.Parse(id)
	if err != nil {
		t.Fatalf("element %s: %v", id, err)
	}
	affinity, err := element.Single(member)
	if err != nil {
		t.Fatalf("affinity %s: %v", id, err)
	}
	return affinity
}

func benchStats(hp, attack, defense, speed int64) progression.Values {
	return progression.Values{
		progression.HP: hp, progression.Attack: attack, progression.Defense: defense,
		progression.Speed: speed, progression.Accuracy: 0, progression.Dodge: 0,
	}
}

func TestBattleReplayGolden(t *testing.T) {
	got := replayReport(11, 90)
	path := filepath.Join("testdata", "replay.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/seed -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("replay differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}

// replayReport renders a battle from its seed the way a terminal client would,
// which is also the check that the event log alone is enough to draw from.
func replayReport(seedValue uint64, lines int) string {
	var b strings.Builder
	fight, err := seed.NewBattle(seedValue)
	if err != nil {
		fmt.Fprintf(&b, "unavailable: %v\n", err)
		return b.String()
	}
	fmt.Fprintf(&b, "== battle from seed %d ==\n", seedValue)
	fight.Begin()
	turns, err := fight.RunToEnd(4000)
	if err != nil {
		fmt.Fprintf(&b, "run failed: %v\n", err)
		return b.String()
	}
	events := fight.Drain()
	winner, decided := fight.Winner()
	outcome := "no winner"
	if decided {
		outcome = winner.String() + " won"
	}
	fmt.Fprintf(&b, "%d turns, %d events, %s\n\n", turns, len(events), outcome)

	fmt.Fprintf(&b, "the first %d events\n", lines)
	shown := lines
	if shown > len(events) {
		shown = len(events)
	}
	for _, event := range events[:shown] {
		b.WriteString(render(event))
		b.WriteString("\n")
	}
	if len(events) > shown {
		fmt.Fprintf(&b, "... %d more\n", len(events)-shown)
	}

	b.WriteString("\nthe last 12 events\n")
	tail := events
	if len(tail) > 12 {
		tail = tail[len(tail)-12:]
	}
	for _, event := range tail {
		b.WriteString(render(event))
		b.WriteString("\n")
	}

	b.WriteString("\nevent counts\n")
	counts := make([]int, battle.KindCount)
	for _, event := range events {
		if int(event.Kind) < len(counts) {
			counts[event.Kind]++
		}
	}
	for kind := 0; kind < battle.KindCount; kind++ {
		fmt.Fprintf(&b, "  %-16s%6d\n", battle.Kind(kind), counts[kind])
	}

	b.WriteString("\nfinal board\n")
	for _, unit := range fight.Units() {
		state := "alive"
		if unit.Dead {
			state = "dead"
		}
		fmt.Fprintf(&b, "  %-18s%-7s%-6s%6d / %-6d hp   %s\n",
			unit.ID, unit.Side, state, unit.HP, unit.MaxHP(), unit.Affinity)
	}
	return b.String()
}

// render turns one event into a line, using nothing but the event itself. That
// constraint is the point: if a line cannot be drawn from the log alone, the log
// is missing something a renderer would have to reach into the battle for.
func render(event battle.Event) string {
	head := fmt.Sprintf("%8d %-16s", event.At, event.Kind)
	switch event.Kind {
	case battle.Started:
		return head + fmt.Sprintf("%-18s %-6s %s  %d hp", event.Actor, event.Side, event.Cell, event.Amount)
	case battle.TurnBegan:
		return head + fmt.Sprintf("%-18s turn %-3d %d hp", event.Actor, event.Turn, event.Amount)
	case battle.StatusTicked:
		return head + fmt.Sprintf("%-18s %s x%d  %d damage", event.Actor, event.Status, event.Stacks, event.Amount)
	case battle.StatusExpired:
		return head + fmt.Sprintf("%-18s %s wore off", event.Actor, event.Status)
	case battle.SpeedChanged:
		return head + fmt.Sprintf("%-18s speed %d to %d", event.Actor, event.Before, event.Amount)
	case battle.TurnSkipped:
		return head + fmt.Sprintf("%-18s %s", event.Actor, event.Note)
	case battle.SkillUsed:
		return head + fmt.Sprintf("%-18s %s at %s", event.Actor, event.Skill, event.Cell)
	case battle.Amplified:
		return head + fmt.Sprintf("%-18s %s on %s x%d, power %d",
			event.Actor, event.Skill, event.Status, event.Stacks, event.Power)
	case battle.StatusConsumed:
		return head + fmt.Sprintf("%-18s consumed %s x%d off %s, %d forgone",
			event.Actor, event.Status, event.Stacks, event.Target, event.Amount)
	case battle.Missed:
		return head + fmt.Sprintf("%-18s %s missed %s at %d per mille",
			event.Actor, event.Skill, event.Target, event.Chance)
	case battle.Blocked:
		return head + fmt.Sprintf("%-18s %s blocked by %s, %d charges left",
			event.Actor, event.Skill, event.Target, event.Remaining)
	case battle.Damaged:
		// The pierce is on the line only when there is some, so a book where
		// nothing pierces reads exactly as it did before piercing existed. It is
		// on the line at all because this golden is the design record: a damage
		// figure it cannot account for from its own terms is a record of nothing.
		pierced := ""
		if event.Pierce > 0 {
			pierced = fmt.Sprintf(", %d per mille pierced", event.Pierce)
		}
		// A reply carries a trait where a strike carries a skill, and the verb
		// changes with it: "answered" rather than "hit", because the damage
		// landed on somebody else's turn from a unit that was not acting, and a
		// record that read the two the same way would be hiding the one thing
		// about a reply that is worth recording.
		if event.Passive != "" {
			return head + fmt.Sprintf("%-18s %s answered %s for %d, x%d affinity, %d hp left",
				event.Actor, event.Passive, event.Target, event.Amount,
				event.Multiplier, event.Remaining)
		}
		return head + fmt.Sprintf("%-18s %s hit %s for %d, x%d affinity%s, %d hp left",
			event.Actor, event.Skill, event.Target, event.Amount, event.Multiplier,
			pierced, event.Remaining)
	case battle.StatusApplied:
		note := ""
		if event.Note != "" {
			note = "  " + event.Note
		}
		if event.Passive != "" {
			note = "  from " + event.Passive + note
		}
		return head + fmt.Sprintf("%-18s %s x%d on %s, now %d%s",
			event.Actor, event.Status, event.Stacks, event.Target, event.Remaining, note)
	case battle.StatusResisted:
		// The share refused only when there is one, so a book where nothing
		// resists reads exactly as it did before resistances existed — and where
		// something does, the record says whether the roll failed or the target
		// refused it.
		if event.Refused > 0 {
			return head + fmt.Sprintf("%-18s %s resisted %s at %d per mille, %d refused",
				event.Actor, event.Target, event.Status, event.Chance, event.Refused)
		}
		return head + fmt.Sprintf("%-18s %s resisted %s at %d per mille",
			event.Actor, event.Target, event.Status, event.Chance)
	case battle.StatusStripped:
		return head + fmt.Sprintf("%-18s %s stripped %d off %s", event.Actor, event.Skill, event.Stacks, event.Target)
	case battle.PassiveHeld:
		return head + fmt.Sprintf("%-18s %s: %s x%d",
			event.Actor, event.Passive, event.Status, event.Stacks)
	case battle.Died:
		return head + fmt.Sprintf("%-18s fell at %s", event.Actor, event.Cell)
	case battle.Ended:
		// The outcome and, when there is one, the side that won it. A stalemate
		// and a mutual kill are both draws and both name no side, so the
		// outcome is what the golden has to carry.
		if event.Outcome == battle.Victory {
			return head + fmt.Sprintf("%-18s %s to the %s side", "", event.Outcome, event.Side)
		}
		return head + fmt.Sprintf("%-18s %s", "", event.Outcome)
	default:
		return head + fmt.Sprintf("%-18s", event.Actor)
	}
}

func TestStatsResolveThroughStatuses(t *testing.T) {
	fight := benchBattle(t, 5)
	// A unit carrying nothing, found rather than named. ally.bulwark was named
	// here and then given a passive, so it resolved to 458 defence against a base
	// of 400 and this failed — which is the mechanism working, in the wrong test.
	var unit *battle.Unit
	for _, candidate := range fight.Units() {
		if len(candidate.Statuses.Active()) == 0 {
			unit = candidate
			break
		}
	}
	if unit == nil {
		t.Fatal("every unit on the bench carries a status, so nothing here measures an unaffected one")
	}
	if got := fight.Stats(unit); got != unit.Base {
		t.Errorf("an unaffected unit resolved to %s, want its base %s", got, unit.Base)
	}
	books := fight.Books()
	weaken, err := books.Statuses.Lookup("weaken")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	unit.Statuses.Apply(weaken, 0)
	unit.Statuses.Apply(weaken, 0)
	weakened := fight.Stats(unit)
	if weakened[progression.Attack] >= unit.Base[progression.Attack] {
		t.Errorf("two stacks of weaken left attack at %d, base is %d",
			weakened[progression.Attack], unit.Base[progression.Attack])
	}
	if weakened[progression.HP] != unit.Base[progression.HP] {
		t.Errorf("weaken changed health to %d", weakened[progression.HP])
	}
}

// TestLoggedBattlesVerify is the whole loop: fight, write the log, read it back,
// replay the decisions it recorded and check the events match. A log that
// survives that is a record of the battle rather than a story about one.
func TestLoggedBattlesVerify(t *testing.T) {
	for seedValue := uint64(0); seedValue < 12; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		fight.Begin()
		script, _, err := fight.Replay(nil, 4000, fight.Suggest)
		if err != nil {
			t.Fatalf("seed %d: play: %v", seedValue, err)
		}
		if !fight.Finished() {
			t.Fatalf("seed %d did not finish", seedValue)
		}
		events := fight.Drain()

		raw, err := battle.MarshalLog(battle.Log{Seed: seedValue, Choices: script, Events: events})
		if err != nil {
			t.Fatalf("seed %d: marshal: %v", seedValue, err)
		}
		parsed, err := battle.ParseLog(raw)
		if err != nil {
			t.Fatalf("seed %d: parse: %v", seedValue, err)
		}

		rerun, err := seed.NewBattle(parsed.Seed)
		if err != nil {
			t.Fatalf("seed %d: reassemble: %v", seedValue, err)
		}
		rerun.Begin()
		if _, _, err := rerun.Replay(parsed.Choices, 4000, nil); err != nil {
			t.Fatalf("seed %d: replay: %v", seedValue, err)
		}
		got := rerun.Drain()
		if len(got) != len(parsed.Events) {
			t.Fatalf("seed %d: the log holds %d events, replaying produced %d",
				seedValue, len(parsed.Events), len(got))
		}
		for i := range got {
			if got[i] != parsed.Events[i] {
				t.Fatalf("seed %d: event %d differs:\nlogged %+v\nre-ran %+v",
					seedValue, i, parsed.Events[i], got[i])
			}
		}
		if !rerun.Finished() {
			t.Errorf("seed %d: the replay did not reach the end the log recorded", seedValue)
		}
	}
}

// TestUndoRebuildsTheExactPosition is what a client's undo rests on. Dropping the
// last decision and replaying the rest has to land on the same state the battle
// was in before that decision, with nothing deep copied to get there.
func TestUndoRebuildsTheExactPosition(t *testing.T) {
	const seedValue = 4
	fight, err := seed.NewBattle(seedValue)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	fight.Begin()
	// Play a while, keeping the position after each decision.
	script, _, err := fight.Replay(nil, 25, fight.Suggest)
	if err != nil {
		t.Fatalf("play: %v", err)
	}
	if len(script) < 6 {
		t.Fatalf("only %d decisions were taken, too few to undo", len(script))
	}
	snapshot := func(f *battle.Battle) string {
		var b strings.Builder
		for _, unit := range f.Units() {
			fmt.Fprintf(&b, "%s|%d|%v|%v;", unit.ID, unit.HP, unit.Dead, unit.Statuses.Snapshot())
		}
		fmt.Fprintf(&b, "now=%d", f.Queue().Now())
		return b.String()
	}

	shortened := script[:len(script)-1]
	before, err := seed.NewBattle(seedValue)
	if err != nil {
		t.Fatalf("reassemble: %v", err)
	}
	before.Begin()
	if _, _, err := before.Replay(shortened, 4000, nil); err != nil {
		t.Fatalf("replay the shortened script: %v", err)
	}

	// Replaying the shortened script twice must land in the same place, and
	// carrying on from it must reproduce the decision that was undone.
	again, err := seed.NewBattle(seedValue)
	if err != nil {
		t.Fatalf("reassemble: %v", err)
	}
	again.Begin()
	if _, _, err := again.Replay(shortened, 4000, nil); err != nil {
		t.Fatalf("replay again: %v", err)
	}
	if snapshot(before) != snapshot(again) {
		t.Errorf("two replays of the same script landed in different positions:\n%s\n%s",
			snapshot(before), snapshot(again))
	}

	resumed, _, err := before.Replay(script[len(shortened):], 4000, nil)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if len(resumed) != 1 {
		t.Fatalf("%d decisions were resumed, want 1", len(resumed))
	}
	full, err := seed.NewBattle(seedValue)
	if err != nil {
		t.Fatalf("reassemble: %v", err)
	}
	full.Begin()
	if _, _, err := full.Replay(script, 4000, nil); err != nil {
		t.Fatalf("replay the full script: %v", err)
	}
	if snapshot(before) != snapshot(full) {
		t.Errorf("undoing then redoing did not return to the original position:\n%s\n%s",
			snapshot(before), snapshot(full))
	}
}
