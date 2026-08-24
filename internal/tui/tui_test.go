package tui_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

func opening(t *testing.T) (*battle.Battle, map[string]string) {
	t.Helper()
	fight, err := seed.NewBattle(11)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	fight.Begin()
	return fight, tui.Tags(fight.Units())
}

func TestTagsAreStableAndPerSide(t *testing.T) {
	fight, tags := opening(t)
	if got, want := len(tags), len(fight.Units()); got != want {
		t.Fatalf("%d units were tagged, want %d", got, want)
	}
	seen := make(map[string]bool, len(tags))
	for _, unit := range fight.Units() {
		tag := tags[unit.ID]
		if tag == "" {
			t.Errorf("unit %q has no tag", unit.ID)
		}
		if len([]rune(tag)) != 2 {
			t.Errorf("unit %q is tagged %q, which will not fit a hex", unit.ID, tag)
		}
		if seen[tag] {
			t.Errorf("tag %q is used twice", tag)
		}
		seen[tag] = true
		wantPrefix := "A"
		if unit.Side == hex.SideEnemy {
			wantPrefix = "E"
		}
		if !strings.HasPrefix(tag, wantPrefix) {
			t.Errorf("unit %q on the %s side is tagged %q", unit.ID, unit.Side, tag)
		}
	}
	// Tagging twice gives the same answer, so a log rendered later still matches.
	again := tui.Tags(fight.Units())
	for id, tag := range tags {
		if again[id] != tag {
			t.Errorf("unit %q was tagged %q then %q", id, tag, again[id])
		}
	}
}

func TestBoardPlacesEveryLivingUnit(t *testing.T) {
	fight, tags := opening(t)
	board := tui.Board(fight, tags)
	for _, unit := range fight.Units() {
		if !strings.Contains(board, tags[unit.ID]) {
			t.Errorf("unit %q is not on the board", unit.ID)
		}
	}
	lines := strings.Split(board, "\n")
	if len(lines) < 8 {
		t.Errorf("the board is %d lines, too few to be a board", len(lines))
	}
	// A fallen unit leaves its cell, so the board shows who is still standing.
	fallen := fight.Units()[0]
	fallen.Dead = true
	after := tui.Board(fight, tags)
	if strings.Contains(after, tags[fallen.ID]) {
		t.Errorf("the fallen unit %q is still on the board", fallen.ID)
	}
	if strings.Count(after, "\n") != strings.Count(board, "\n") {
		t.Error("the board changed shape when a unit fell")
	}
}

func TestHealthBar(t *testing.T) {
	cases := []struct {
		name             string
		current, max     int64
		wantFull, wantAt string
	}{
		{"full", 100, 100, "##########", "  100/100  "},
		{"half", 50, 100, "#####.....", ""},
		{"empty", 0, 100, "..........", ""},
		// A unit that is alive keeps a mark, so almost dead does not read as dead.
		{"barely alive", 1, 100, "#.........", ""},
		{"negative is treated as empty", -20, 100, "..........", ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := tui.HealthBar(testCase.current, testCase.max)
			if !strings.Contains(got, testCase.wantFull) {
				t.Errorf("HealthBar(%d, %d) = %q, want a bar of %q",
					testCase.current, testCase.max, got, testCase.wantFull)
			}
		})
	}
	if got := tui.HealthBar(0, 0); got != "-" {
		t.Errorf("a unit with no maximum renders as %q, want %q", got, "-")
	}
}

func TestEffectsSummarisesStacksAndDuration(t *testing.T) {
	fight, _ := opening(t)
	unit := fight.Units()[0]
	if got := tui.Effects(unit); got != "-" {
		t.Errorf("a clean unit shows %q, want %q", got, "-")
	}
	poison, err := fight.Books().Statuses.Lookup("poison")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	unit.Statuses.Apply(poison, 120)
	unit.Statuses.Apply(poison, 120)
	got := tui.Effects(unit)
	for _, want := range []string{"poison", "x2", "3t"} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary %q is missing %q", got, want)
		}
	}
}

func TestOrderShowsWhatIsComing(t *testing.T) {
	fight, tags := opening(t)
	got := tui.Order(fight.Queue(), tags, 6)
	if !strings.HasPrefix(got, "next: ") {
		t.Fatalf("the order line is %q", got)
	}
	fields := strings.Fields(strings.TrimPrefix(got, "next: "))
	if len(fields) != 6 {
		t.Errorf("%d turns were shown, want 6", len(fields))
	}
	// The fastest unit leads, and the preview never disturbs the queue.
	before := fight.Queue().Now()
	tui.Order(fight.Queue(), tags, 20)
	if fight.Queue().Now() != before {
		t.Error("rendering the order advanced the battle")
	}
	empty := tui.Order(fight.Queue(), tags, 0)
	if empty != "next: -" {
		t.Errorf("an empty preview renders as %q", empty)
	}
}

// TestMenuShowsUnusableSkillsWithTheirReason keeps a skill from silently
// vanishing from the list. A player deciding what to do needs to know a skill
// exists and is two turns away.
func TestMenuShowsUnusableSkillsWithTheirReason(t *testing.T) {
	fight, tags := opening(t)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	unit, _ := fight.Unit(prompt.Unit)
	menu := tui.Menu(fight, prompt, tags)
	lines := strings.Split(menu, "\n")
	if len(lines) != len(prompt.Options) {
		t.Fatalf("the menu has %d lines for %d options", len(lines), len(prompt.Options))
	}
	unusable := 0
	for i, option := range prompt.Options {
		if !strings.Contains(lines[i], option.Skill) {
			t.Errorf("line %d does not name %q", i, option.Skill)
		}
		if option.Available() {
			continue
		}
		unusable++
		if !strings.Contains(lines[i], option.Reason) {
			t.Errorf("line %d does not give the reason %q", i, option.Reason)
		}
	}
	// The backline unit that opens this battle has a melee skill it cannot use,
	// which is exactly the case the reason exists for.
	if unusable == 0 {
		t.Errorf("unit %q had nothing unusable, so the reason path is untested", unit.ID)
	}
}

func TestAimsNamesWhoIsCaught(t *testing.T) {
	fight, tags := opening(t)
	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	var area battle.Option
	for _, option := range prompt.Options {
		if !option.Available() {
			continue
		}
		declared, err := fight.Books().Skills.Lookup(option.Skill)
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if declared.Pattern != "single" {
			area = option
			break
		}
	}
	if area.Skill == "" {
		t.Skip("the opening unit has no area skill available")
	}
	rendered := tui.Aims(fight, area, tags)
	lines := strings.Split(rendered, "\n")
	if len(lines) != len(area.Aims) {
		t.Fatalf("%d lines for %d cells", len(lines), len(area.Aims))
	}
	caught := 0
	for i, aim := range area.Aims {
		if !strings.Contains(lines[i], aim.String()) {
			t.Errorf("line %d does not name the cell %s", i, aim)
		}
		caught += strings.Count(lines[i], ",") + 1
	}
	// An area skill has to show more than one unit somewhere, or the display is
	// hiding what the shape does.
	if caught <= len(area.Aims) {
		t.Errorf("no aim showed more than one unit caught: %q", rendered)
	}
}

// TestEveryEventKindRenders is the same discipline the engine holds itself to: a
// kind that falls through to the default has no line worth reading, and nobody
// would notice until it happened in a real battle.
func TestEveryEventKindRenders(t *testing.T) {
	tags := map[string]string{"a": "A1", "f": "E1"}
	for kind := 0; kind < battle.KindCount; kind++ {
		event := battle.Event{
			Kind: battle.Kind(kind), At: 5000, Turn: 3,
			Actor: "a", Target: "f", Skill: "ember_lance", Status: "burn",
			Amount: 617, Before: 100, Stacks: 2, Strike: 1, Chance: 900,
			Multiplier: 1500, Power: 1800, Remaining: 2400,
			Cell: hex.Offset{Col: 3, Row: 1}, Side: hex.SideAlly, Note: "ally",
		}
		line := tui.Line(event, tags)
		if strings.TrimSpace(line) == "" {
			t.Errorf("%s renders as nothing", battle.Kind(kind))
		}
		if strings.Contains(line, "kind(") {
			t.Errorf("%s falls through to the default: %q", battle.Kind(kind), line)
		}
		// The default branch prints the tag and the kind name and nothing else.
		// Comparing against that exact shape catches a missing case without
		// flagging a line that legitimately uses the word.
		if line == fmt.Sprintf("  %-3s %s", tags[event.Actor], battle.Kind(kind)) {
			t.Errorf("%s has no case of its own: %q", battle.Kind(kind), line)
		}
	}
}

func TestLineSpellsOutTheAffinity(t *testing.T) {
	tags := map[string]string{"a": "A1", "f": "E1"}
	cases := []struct {
		multiplier int
		want       string
	}{
		{1000, ""},
		{1500, "(weak)"},
		{2250, "(doubly weak!)"},
		{667, "(resisted)"},
		{444, "(doubly resisted)"},
	}
	for _, testCase := range cases {
		line := tui.Line(battle.Event{
			Kind: battle.Damaged, Actor: "a", Target: "f",
			Amount: 500, Multiplier: testCase.multiplier, Remaining: 1000,
		}, tags)
		if testCase.want == "" {
			if strings.Contains(line, "(") {
				t.Errorf("a neutral matchup rendered as %q", line)
			}
			continue
		}
		if !strings.Contains(line, testCase.want) {
			t.Errorf("a multiplier of %d rendered as %q, want %q in it",
				testCase.multiplier, line, testCase.want)
		}
	}
}

func TestLineFallsBackToTheIdWhenUntagged(t *testing.T) {
	line := tui.Line(battle.Event{Kind: battle.Damaged, Actor: "stranger", Target: "other"}, nil)
	if !strings.Contains(line, "stranger") || !strings.Contains(line, "other") {
		t.Errorf("an untagged unit rendered as %q", line)
	}
}

func TestLogEmpty(t *testing.T) {
	if got := tui.Log(nil, nil); got != "" {
		t.Errorf("an empty log rendered as %q", got)
	}
}

func TestOpeningGolden(t *testing.T) {
	fight, tags := opening(t)
	var b strings.Builder
	b.WriteString(tui.Board(fight, tags))
	b.WriteString("\n\n")
	b.WriteString(tui.Roster(fight, tags))
	b.WriteString("\n\n")
	b.WriteString(tui.Order(fight.Queue(), tags, 8))
	b.WriteString("\n\n")
	b.WriteString(tui.Log(fight.Drain(), tags))
	b.WriteString("\n")

	prompt, err := fight.Advance()
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	unit, _ := fight.Unit(prompt.Unit)
	fmt.Fprintf(&b, "\n%s %s, turn %d at %s\n", tags[unit.ID], unit.Name, prompt.Turn, unit.Cell)
	b.WriteString(tui.Menu(fight, prompt, tags))
	b.WriteString("\n")
	for _, option := range prompt.Options {
		if !option.Available() {
			continue
		}
		fmt.Fprintf(&b, "\naim %s at:\n", option.Skill)
		b.WriteString(tui.Aims(fight, option, tags))
		b.WriteString("\n")
	}

	got := b.String()
	path := filepath.Join("testdata", "opening.golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/tui -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("the opening render differs from %s; rerun with -update to accept\n--- got ---\n%s", path, got)
	}
}
