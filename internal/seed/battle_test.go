package seed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

func TestShippedRosterIsUsable(t *testing.T) {
	roster, err := seed.Roster()
	if err != nil {
		t.Fatalf("load roster: %v", err)
	}
	if got, want := len(roster), 2*hex.MaxTeamSize; got != want {
		t.Fatalf("the roster holds %d units, want two full teams of %d", got, want)
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
		if perSide[side] != hex.MaxTeamSize {
			t.Errorf("the %s side has %d units, want %d", side, perSide[side], hex.MaxTeamSize)
		}
	}
	// A roster where both sides share one affinity would never exercise the
	// element chart, which is most of what a battle is deciding.
	if len(elements) < 8 {
		t.Errorf("the roster uses %d distinct affinities, want a spread across the chart", len(elements))
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

// TestEveryEventKindIsReachable stops a kind being declared and never emitted,
// which would mean a renderer had a case nobody could test.
func TestEveryEventKindIsReachable(t *testing.T) {
	seen := make(map[battle.Kind]bool, battle.KindCount)
	for seedValue := uint64(0); seedValue < 60; seedValue++ {
		fight, err := seed.NewBattle(seedValue)
		if err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		fight.Begin()
		if _, err := fight.RunToEnd(4000); err != nil {
			t.Fatalf("seed %d: %v", seedValue, err)
		}
		for _, event := range fight.Drain() {
			seen[event.Kind] = true
		}
	}
	for kind := 0; kind < battle.KindCount; kind++ {
		if !seen[battle.Kind(kind)] {
			t.Errorf("no shipped battle ever emitted %s", battle.Kind(kind))
		}
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
		return head + fmt.Sprintf("%-18s %s hit %s for %d, x%d affinity, %d hp left",
			event.Actor, event.Skill, event.Target, event.Amount, event.Multiplier, event.Remaining)
	case battle.StatusApplied:
		note := ""
		if event.Note != "" {
			note = "  " + event.Note
		}
		return head + fmt.Sprintf("%-18s %s x%d on %s, now %d%s",
			event.Actor, event.Status, event.Stacks, event.Target, event.Remaining, note)
	case battle.StatusResisted:
		return head + fmt.Sprintf("%-18s %s resisted %s at %d per mille",
			event.Actor, event.Target, event.Status, event.Chance)
	case battle.StatusStripped:
		return head + fmt.Sprintf("%-18s %s stripped %d off %s", event.Actor, event.Skill, event.Stacks, event.Target)
	case battle.Died:
		return head + fmt.Sprintf("%-18s fell at %s", event.Actor, event.Cell)
	case battle.Ended:
		return head + fmt.Sprintf("%-18s %s", "", event.Note)
	default:
		return head + fmt.Sprintf("%-18s", event.Actor)
	}
}

func TestStatsResolveThroughStatuses(t *testing.T) {
	fight, err := seed.NewBattle(5)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	unit, ok := fight.Unit("ally.bulwark")
	if !ok {
		t.Fatal("the roster has no ally.bulwark")
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
