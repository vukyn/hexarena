// Package tui renders a battle for a terminal.
//
// Everything here is a pure function returning a string. Nothing reads input,
// writes output or holds state, which is what makes the rendering testable and
// keeps the client a renderer rather than a second copy of the rules.
//
// The event lines are built from the event alone, never from the battle. That
// constraint is deliberate: if a line cannot be drawn from the log, the log is
// missing something, and a renderer that reaches into the engine to fill the gap
// is one that will drift from it.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vukyn/hexarena/internal/core/atb"
	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// Tags assigns each unit a two character label, stable in roster order, so the
// board can name a cell in the space a hex has.
func Tags(units []*battle.Unit) map[string]string {
	counts := map[hex.Side]int{}
	letters := map[hex.Side]string{hex.SideAlly: "A", hex.SideEnemy: "E"}
	out := make(map[string]string, len(units))
	for _, unit := range units {
		counts[unit.Side]++
		out[unit.ID] = fmt.Sprintf("%s%d", letters[unit.Side], counts[unit.Side])
	}
	return out
}

// Board draws the battlefield, labelling each occupied cell with its unit's tag.
func Board(fight *battle.Battle, tags map[string]string) string {
	occupied := make(map[hex.Offset]string, len(fight.Units()))
	for _, unit := range fight.Units() {
		if unit.Dead {
			continue
		}
		occupied[unit.Cell] = tags[unit.ID]
	}
	header := " c0  c1  c2  c3  c4  c5\n BK  MD  FR  FR  MD  BK\n"
	return header + hex.Render(func(cell hex.Offset) string { return occupied[cell] })
}

// Roster lists every unit with its health, tempo and active effects.
func Roster(fight *battle.Battle, tags map[string]string) string {
	var b strings.Builder
	b.WriteString("tag  unit                 hp                        spd   effects\n")
	for _, unit := range fight.Units() {
		state := HealthBar(unit.HP, unit.MaxHP())
		if unit.Dead {
			state = "fallen"
		}
		stats := fight.Stats(unit)
		fmt.Fprintf(&b, "%-5s%-21s%-26s%4d   %s\n",
			tags[unit.ID], unit.Name, state,
			stats[progression.Speed], Effects(unit))
	}
	return trimLines(b.String())
}

// HealthBar draws a health figure as a bar and a count.
func HealthBar(current, max int64) string {
	const width = 10
	if max <= 0 {
		return "-"
	}
	if current < 0 {
		current = 0
	}
	filled := int(current * width / max)
	// A unit that is alive at all keeps one mark, so an almost dead unit does
	// not read as a dead one.
	if filled == 0 && current > 0 {
		filled = 1
	}
	return fmt.Sprintf("[%s%s] %5d/%-5d",
		strings.Repeat("#", filled), strings.Repeat(".", width-filled), current, max)
}

// Effects summarises a unit's timed effects.
func Effects(unit *battle.Unit) string {
	snapshot := unit.Statuses.Snapshot()
	if len(snapshot) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(snapshot))
	for _, entry := range snapshot {
		part := entry.ID
		if entry.Stacks > 1 {
			part += fmt.Sprintf(" x%d", entry.Stacks)
		}
		part += fmt.Sprintf(" (%dt)", entry.Remaining)
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

// Order shows the units due to act next.
func Order(queue *atb.Queue, tags map[string]string, count int) string {
	turns := queue.Preview(count)
	if len(turns) == 0 {
		return "next: -"
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		label := tags[turn.ID]
		if label == "" {
			label = turn.ID
		}
		parts = append(parts, label)
	}
	return "next: " + strings.Join(parts, " ")
}

// Menu lists what the acting unit may do. Unavailable options are shown with
// their reason rather than hidden, because a player deciding what to do needs to
// know a skill exists and is two turns away.
func Menu(fight *battle.Battle, prompt *battle.Prompt, tags map[string]string) string {
	var b strings.Builder
	books := fight.Books()
	for index, option := range prompt.Options {
		declared, err := books.Skills.Lookup(option.Skill)
		if err != nil {
			fmt.Fprintf(&b, " %d) %-16s unknown\n", index+1, option.Skill)
			continue
		}
		label := fmt.Sprintf(" %d)", index+1)
		if !option.Available() {
			label = "  -"
		}
		fmt.Fprintf(&b, "%s %-16s%-10s rng %-3d %-13s pow %-6d acc %-5d",
			label, declared.ID, declared.Element, declared.Range,
			declared.Pattern, declared.TotalPower(), declared.Accuracy)
		if declared.StrikeCount() > 1 {
			fmt.Fprintf(&b, " x%d", declared.StrikeCount())
		}
		if declared.Cooldown > 0 {
			fmt.Fprintf(&b, " cd%d", declared.Cooldown)
		}
		if extra := Extras(declared); extra != "" {
			fmt.Fprintf(&b, "  %s", extra)
		}
		if !option.Available() {
			fmt.Fprintf(&b, "  <%s>", option.Reason)
		}
		b.WriteString("\n")
	}
	_ = tags
	return trimLines(b.String())
}

// trimLines drops the padding a column layout leaves at the end of a line, so
// what is printed has no invisible tail.
func trimLines(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

// Extras summarises what a skill does beyond damage.
func Extras(declared skill.Skill) string {
	parts := make([]string, 0, 4)
	for _, application := range declared.Applies {
		parts = append(parts, fmt.Sprintf("%s %d%%", application.Status, application.Chance/10))
	}
	for _, application := range declared.SelfApplies {
		parts = append(parts, fmt.Sprintf("self %s x%d", application.Status, application.Stacks))
	}
	if declared.Requires != nil {
		verb := "needs"
		if declared.Requires.Consume {
			verb = "eats"
		}
		parts = append(parts, fmt.Sprintf("%s %s x%d for +%d",
			verb, declared.Requires.Status, declared.Requires.MinStacks, declared.Requires.BonusPower))
	}
	if declared.Strips != nil {
		names := make([]string, 0, len(declared.Strips.Categories))
		for _, category := range declared.Strips.Categories {
			names = append(names, category.String())
		}
		sort.Strings(names)
		parts = append(parts, fmt.Sprintf("strips %s x%d", strings.Join(names, "/"), declared.Strips.Stacks))
	}
	return strings.Join(parts, ", ")
}

// Aims lists the cells a chosen skill may be pointed at, with who is standing
// there and what the shape would catch.
func Aims(fight *battle.Battle, option battle.Option, tags map[string]string) string {
	var b strings.Builder
	books := fight.Books()
	declared, err := books.Skills.Lookup(option.Skill)
	if err != nil {
		return "unknown skill"
	}
	shape, err := books.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return "unknown shape"
	}
	occupied := make(map[hex.Offset]*battle.Unit, len(fight.Units()))
	for _, unit := range fight.Units() {
		if !unit.Dead {
			occupied[unit.Cell] = unit
		}
	}
	for index, aim := range option.Aims {
		caught := make([]string, 0, shape.MaxTargets())
		for _, cell := range shape.Targets(aim) {
			unit, standing := occupied[cell]
			if !standing {
				continue
			}
			caught = append(caught, fmt.Sprintf("%s %s", tags[unit.ID], unit.Name))
		}
		fmt.Fprintf(&b, " %d) %-6s%s\n", index+1, aim, strings.Join(caught, ", "))
	}
	return trimLines(b.String())
}

// Line renders one event, using nothing but the event.
func Line(event battle.Event, tags map[string]string) string {
	tag := func(id string) string {
		if id == "" {
			return ""
		}
		if label, ok := tags[id]; ok {
			return label
		}
		return id
	}
	head := fmt.Sprintf("  %-3s", tag(event.Actor))
	switch event.Kind {
	case battle.Started:
		return fmt.Sprintf("  %-3s enters at %s on the %s side with %d health",
			tag(event.Actor), event.Cell, event.Side, event.Amount)
	case battle.TurnBegan:
		return fmt.Sprintf("\n  %-3s turn %d", tag(event.Actor), event.Turn)
	case battle.StatusTicked:
		return head + fmt.Sprintf(" takes %d from %s x%d", event.Amount, event.Status, event.Stacks)
	case battle.StatusExpired:
		return head + fmt.Sprintf(" %s wears off", event.Status)
	case battle.SpeedChanged:
		return head + fmt.Sprintf(" speed %d to %d", event.Before, event.Amount)
	case battle.TurnSkipped:
		return head + fmt.Sprintf(" loses the turn (%s)", event.Note)
	case battle.SkillUsed:
		return head + fmt.Sprintf(" uses %s at %s", event.Skill, event.Cell)
	case battle.Amplified:
		return head + fmt.Sprintf("  %s is amplified by %s x%d, power %d",
			event.Skill, event.Status, event.Stacks, event.Power)
	case battle.StatusConsumed:
		return head + fmt.Sprintf("  consumes %s x%d off %s, giving up %d",
			event.Status, event.Stacks, tag(event.Target), event.Amount)
	case battle.Missed:
		return head + fmt.Sprintf("  misses %s (%d%%)", tag(event.Target), event.Chance/10)
	case battle.Blocked:
		return head + fmt.Sprintf("  is blocked by %s, %d charges left", tag(event.Target), event.Remaining)
	case battle.Damaged:
		return head + fmt.Sprintf("  hits %s for %d%s, %d left",
			tag(event.Target), event.Amount, affinityNote(event.Multiplier), event.Remaining)
	case battle.StatusApplied:
		note := ""
		if event.Note != "" {
			note = ", " + event.Note
		}
		return head + fmt.Sprintf("  %s x%d on %s, now %d%s",
			event.Status, event.Stacks, tag(event.Target), event.Remaining, note)
	case battle.StatusResisted:
		return head + fmt.Sprintf("  %s resists %s (%d%%)", tag(event.Target), event.Status, event.Chance/10)
	case battle.StatusStripped:
		return head + fmt.Sprintf("  strips %d off %s", event.Stacks, tag(event.Target))
	case battle.Died:
		return head + fmt.Sprintf(" falls at %s", event.Cell)
	case battle.Ended:
		return fmt.Sprintf("\n  the %s side wins", event.Note)
	default:
		return head + " " + event.Kind.String()
	}
}

// affinityNote spells out an elemental multiplier, and says nothing when the
// matchup is neutral.
func affinityNote(multiplier int) string {
	switch {
	case multiplier == 0 || multiplier == 1000:
		return ""
	case multiplier >= 2000:
		return " (doubly weak!)"
	case multiplier > 1000:
		return " (weak)"
	case multiplier <= 500:
		return " (doubly resisted)"
	default:
		return " (resisted)"
	}
}

// Log renders a run of events.
func Log(events []battle.Event, tags map[string]string) string {
	if len(events) == 0 {
		return ""
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, Line(event, tags))
	}
	return strings.Join(lines, "\n")
}
