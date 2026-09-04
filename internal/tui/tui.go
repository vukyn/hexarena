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
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
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
		// A permanent status has no countdown to print, and printing its zero
		// would read as "about to run out" for the one thing that never does.
		if entry.Permanent {
			part += " (always)"
		} else {
			part += fmt.Sprintf(" (%dt)", entry.Remaining)
		}
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
			// ⚠️ **The engine's English sentence on purpose, not an oversight.**
			// screen.OptionRefusal says the same four facts in the reader's own
			// language, off battle.Block and the counts beside it — and it needs a
			// Lang to do it. This function is handed an event and a book and
			// nothing else, which is the property that lets a replay be drawn with
			// no library at all, so there is no language here to ask. Every other
			// column on this line is a bare id or a number for the same reason.
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
		// A condition reads a status, or how hurt the target is, or both, so the
		// clause is assembled rather than formatted in one go: the old single
		// line printed the status field unconditionally and would render a
		// health-only condition as "needs  x0", which reads as a bug in the
		// skill rather than as a skill this line cannot describe.
		clauses := make([]string, 0, 2)
		if declared.Requires.ReadsStatus() {
			clauses = append(clauses, fmt.Sprintf("%s x%d",
				declared.Requires.Status, declared.Requires.MinStacks))
		}
		if declared.Requires.ReadsHealth() {
			clauses = append(clauses, fmt.Sprintf("health <=%d%%", declared.Requires.BelowHealth/10))
		}
		parts = append(parts, fmt.Sprintf("%s %s for +%d",
			verb, strings.Join(clauses, " and "), declared.Requires.BonusPower))
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
//
// glosses is data id -> display name, for the skill, status and trait ids the
// line prints — i18n.Lang.LogGlosses is what builds one. It is a caller-supplied
// map for exactly the reason tags and Summary's names are: the event lines are
// built from the event alone, so this package may not be handed books, a battle
// or a language, and a name is a fact about the reader rather than about what
// happened. **A nil or empty map reproduces the line byte for byte**, which is
// both what English is (a data id is shown as the data writes it) and what a
// replay drawn without books is, and it is the property the goldens hold.
//
// Every occurrence is glossed rather than the first mention: a log is read in
// pieces, scrolled and frozen at a frame, so "the first mention" is a row a
// reader may never have on screen.
func Line(event battle.Event, tags, glosses map[string]string) string {
	tag := func(id string) string {
		if id == "" {
			return ""
		}
		if label, ok := tags[id]; ok {
			return label
		}
		return id
	}
	// gloss is the id with its name beside it, and the bare id when there is
	// none. The bracket comes from i18n rather than being spelled here, so the log
	// and every screen that names a data id agree on the punctuation.
	gloss := func(id string) string {
		if id == "" {
			return ""
		}
		return i18n.GlossBracket(id, glosses[id])
	}
	head := fmt.Sprintf("  %-3s", tag(event.Actor))
	switch event.Kind {
	case battle.Started:
		name := event.Name
		if name == "" {
			name = event.Actor
		}
		return fmt.Sprintf("  %-3s %s enters at %s on the %s side with %d health, %s",
			tag(event.Actor), name, event.Cell, event.Side, event.Amount, event.Note)
	case battle.TurnBegan:
		return fmt.Sprintf("\n  %-3s turn %d", tag(event.Actor), event.Turn)
	case battle.StatusTicked:
		return head + fmt.Sprintf(" takes %d from %s x%d", event.Amount, gloss(event.Status), event.Stacks)
	case battle.Healed:
		// A regeneration names itself; a drain or a restoring skill has no
		// status to name and says only how much came back.
		if event.Status != "" {
			return head + fmt.Sprintf(" heals %d from %s x%d%s", event.Amount,
				gloss(event.Status), event.Stacks, reducedNote(event.Reduced))
		}
		// A drain says the share it took, because the amount alone cannot be
		// reproduced from the skill any more: a trait may drain as well, so the
		// number on screen is not the skill's own figure applied to the damage.
		if event.Drained > 0 {
			return head + fmt.Sprintf(" drains %d, %d%% of what it dealt, %d hp left%s",
				event.Amount, event.Drained/10, event.Remaining, reducedNote(event.Reduced))
		}
		return head + fmt.Sprintf(" heals %d, %d hp left%s", event.Amount, event.Remaining,
			reducedNote(event.Reduced))
	case battle.StatusExpired:
		return head + fmt.Sprintf(" %s wears off", gloss(event.Status))
	case battle.SpeedChanged:
		return head + fmt.Sprintf(" speed %d to %d", event.Before, event.Amount)
	case battle.TurnSkipped:
		return head + fmt.Sprintf(" loses the turn (%s)", event.Note)
	case battle.SkillUsed:
		return head + fmt.Sprintf(" uses %s at %s%s", gloss(event.Skill), event.Cell,
			gradientNote(event.Gradient))
	case battle.Amplified:
		return head + fmt.Sprintf("  %s amplified by %s x%d, power x%s",
			gloss(event.Skill), gloss(event.Status), event.Stacks, multiple(event.Power))
	case battle.Spread:
		return head + fmt.Sprintf("  %s jumps off %s carrying %s",
			gloss(event.Skill), tag(event.Target), gloss(event.Status))
	case battle.StatusConsumed:
		// The stacks left where a consume took only some. Nought is both "took
		// the lot" and "there were none left", which are the same fact from the
		// reader's side, so the clause is dropped rather than printing a zero
		// that says nothing.
		if event.Remaining > 0 {
			return head + fmt.Sprintf("  consumes %s x%d off %s, giving up %d, %d left",
				gloss(event.Status), event.Stacks, tag(event.Target), event.Amount, event.Remaining)
		}
		return head + fmt.Sprintf("  consumes %s x%d off %s, giving up %d",
			gloss(event.Status), event.Stacks, tag(event.Target), event.Amount)
	case battle.Missed:
		return head + fmt.Sprintf("  misses %s (%d%%)", tag(event.Target), event.Chance/10)
	case battle.Blocked:
		return head + fmt.Sprintf("  is blocked by %s, %d charges left", tag(event.Target), event.Remaining)
	case battle.Paid:
		// Beside the guard's line because both are health moving for a reason a
		// strike does not explain, and worded so it cannot be read as damage:
		// this is the caster handing something over, not somebody taking it.
		return head + fmt.Sprintf(" pays %d for %s, %d hp left",
			event.Amount, gloss(event.Skill), event.Remaining)
	case battle.Absorbed:
		// A line of its own beside the block above, and the two words are chosen
		// to be unmistakable: a blocked strike is stopped and this one is eaten.
		// The figure after it is what the barrier has left rather than a charge
		// count, which is the whole difference between the two guards said in the
		// unit each is measured in.
		return head + fmt.Sprintf("  %d soaked by %s, %d left in the barrier",
			event.Amount, tag(event.Target), event.Remaining)
	case battle.Damaged:
		// A reply is damage, and the only thing that separates it from a strike
		// in the log is that a trait rather than a skill is named — so it is the
		// same case, worded so a reader can tell that this happened on somebody
		// else's turn. Reading it as damage-with-no-skill would work and would
		// be a rule nobody wrote down.
		if event.Passive != "" {
			return head + fmt.Sprintf("  answers %s with %s for %d%s, %d left",
				tag(event.Target), gloss(event.Passive), event.Amount,
				affinityNote(event.Multiplier), event.Remaining)
		}
		return head + fmt.Sprintf("  hits %s for %d%s%s%s, %d left",
			tag(event.Target), event.Amount, affinityNote(event.Multiplier),
			pierceNote(event.Pierce), criticalNote(event.Critical), event.Remaining)
	case battle.StatusApplied:
		note := ""
		if event.Note != "" {
			note = ", " + event.Note
		}
		// The trait is named where there is one, for the same reason the damage
		// above names it: a status arriving on the attacker's own turn, from the
		// unit it just hit, has nothing else in the log to account for it.
		source := ""
		if event.Passive != "" {
			source = " (" + gloss(event.Passive) + ")"
		}
		// A vulnerability shows on the line where the status lands, because that
		// is the case it exists to cause — the refusal arm below never runs for a
		// target that made itself easier to hit and still got lucky.
		if event.Refused < 0 {
			source += fmt.Sprintf(" (%d%% invited)", -event.Refused/10)
		}
		return head + fmt.Sprintf("  %s x%d on %s, now %d%s%s",
			gloss(event.Status), event.Stacks, tag(event.Target), event.Remaining, note, source)
	case battle.StatusResisted:
		// Two different things end an application, and the kind is called
		// status_resisted for both: the roll failed, or the target's traits
		// refused it. Saying which is the whole reason the event carries the
		// share refused — a reader given only "resists" cannot tell a piece of
		// luck from a property of the unit.
		switch {
		case event.Refused >= 1000:
			return head + fmt.Sprintf("  %s is immune to %s", tag(event.Target), gloss(event.Status))
		case event.Refused > 0:
			return head + fmt.Sprintf("  %s shrugs off %s (%d%% chance, %d%% refused)",
				tag(event.Target), gloss(event.Status), event.Chance/10, event.Refused/10)
		case event.Refused < 0:
			// A vulnerability, which is a refusal of a negative share. Without
			// this line the roll below prints a chance higher than the skill's
			// own and nothing on screen says why — and explaining its own figures
			// is the whole job of this renderer.
			return head + fmt.Sprintf("  %s is wide open to %s (%d%% chance, %d%% invited)",
				tag(event.Target), gloss(event.Status), event.Chance/10, -event.Refused/10)
		default:
			return head + fmt.Sprintf("  %s resists %s (%d%%)",
				tag(event.Target), gloss(event.Status), event.Chance/10)
		}
	case battle.StatusStripped:
		return head + fmt.Sprintf("  strips %d off %s", event.Stacks, tag(event.Target))
	case battle.PassiveHeld:
		// A trait and the permanent status it put on, in one line, because
		// either half alone leaves the reader guessing: the trait's name says
		// nothing about what it does, and the status appearing on its own has
		// nothing to account for it.
		return head + fmt.Sprintf("  holds %s: %s x%d", gloss(event.Passive), gloss(event.Status), event.Stacks)
	case battle.PassiveReleased:
		// The same line the other way round. A gated trait letting go takes a
		// visible number down with it, so it reads beside the heal that caused
		// it rather than being left to the reader to infer from the damage.
		return head + fmt.Sprintf("  lets go of %s: %s x%d", gloss(event.Passive), gloss(event.Status), event.Stacks)
	case battle.Died:
		return head + fmt.Sprintf(" falls at %s", event.Cell)
	case battle.Summoned:
		// The new unit rather than the caster, and its health with it: this is
		// the line that introduces somebody, so it carries what a started line
		// carries — a reader meeting a name for the first time needs the same
		// facts whether the roster placed it or a skill did.
		return head + fmt.Sprintf("  %s calls up %s at %s, %d hp",
			gloss(event.Skill), event.Target, event.Cell, event.Amount)
	case battle.Left:
		// Not "falls". A copy running out of turns is not a unit being beaten,
		// and the note says which of the two reasons it was.
		return head + fmt.Sprintf(" leaves at %s (%s)", event.Cell, event.Note)
	case battle.Ended:
		// Every ending is drawn from the outcome rather than from the winner,
		// because three of the four have no winner to name and one of them —
		// a stalemate — leaves units standing on both sides. Drawing that as
		// "nobody is left" would be the log telling the reader something false.
		switch event.Outcome {
		case battle.Victory:
			return fmt.Sprintf("\n  the %s side wins", event.Side)
		case battle.Annihilation:
			return "\n  a draw: nobody is left standing"
		case battle.Stalemate:
			return "\n  a draw: nobody can reach anyone"
		default:
			return "\n  the battle ends"
		}
	default:
		return head + " " + event.Kind.String()
	}
}

// multiple spells a parts-per-thousand power as the multiplier it actually is,
// because the raw figure is the only number on the line a reader has to divide
// by a thousand before it means anything. A power of 3500 reads as x3.5 and a
// power of 1000 as x1; trailing zeroes are trimmed so the common figures stay
// short, and no number is rounded, because a thousandth is the smallest a power
// can be and the log must stay reproducible against the rules.
//
// ASCII x rather than a multiplication sign: an ambiguous-width glyph is
// measured as one cell and drawn as two in enough terminals to overlap the
// column beside it.
func multiple(permille int) string {
	sign := ""
	if permille < 0 {
		sign, permille = "-", -permille
	}
	whole, fraction := permille/scale.Base, permille%scale.Base
	if fraction == 0 {
		return fmt.Sprintf("%s%d", sign, whole)
	}
	return fmt.Sprintf("%s%d.%s", sign, whole,
		strings.TrimRight(fmt.Sprintf("%03d", fraction), "0"))
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

// pierceNote says how much of the target's armour a hit went through, and says
// nothing at all when it went through none.
//
// It is on the line because a reader who cannot see it cannot account for the
// damage: the same attacker, power and multiplier against the same defender
// produce a different figure, and a log a reader cannot reproduce is the log
// lying. The share rather than the defence it left, because the share is the
// skill's own property and the defence is the defender's.
// gradientNote is what a hurt caster added to its own skill, and it is worded as
// a share rather than a power because the share is the part a reader cannot
// work out: the power on the event is the book's figure, and the strike that
// follows is larger than it with nothing else to say why.
func gradientNote(gradient int) string {
	if gradient <= 0 {
		return ""
	}
	return fmt.Sprintf(" (hurt, +%d%%)", gradient/10)
}

// reducedNote is what the healed unit's own statuses took off the number in
// front of it, and says nothing at all when they took nothing.
//
// It is on the line for the reason pierceNote and gradientNote are, and it is the
// least optional of the three: a heal of 244 off a skill the book prints as 900
// leaves a reader with no figure on the screen or in the data that agrees with
// it. The share rather than the amount lost, because the share is the property of
// the statuses on the unit and the amount is a consequence of whatever was aimed
// at it.
func reducedNote(reduced int) string {
	if reduced <= 0 {
		return ""
	}
	return fmt.Sprintf(" (healing cut %d%%)", reduced/10)
}

func pierceNote(pierce int) string {
	if pierce <= 0 {
		return ""
	}
	if pierce >= 1000 {
		return " (straight through the armour)"
	}
	return fmt.Sprintf(" (through %d%% of the armour)", pierce/10)
}

// criticalNote marks a strike that landed critically. Like pierceNote it names
// no figure: what a critical is worth is one constant the rules hold, and this
// renderer reads events rather than the rules.
func criticalNote(critical bool) string {
	if !critical {
		return ""
	}
	return " (critical)"
}

// Log renders a run of events. glosses is Line's, and a nil one renders exactly
// what this function rendered before there was a third parameter.
func Log(events []battle.Event, tags, glosses map[string]string) string {
	if len(events) == 0 {
		return ""
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, Line(event, tags, glosses))
	}
	return strings.Join(lines, "\n")
}

// Tally is what one unit did and had done to it, counted from the event log.
type Tally struct {
	ID     string
	Name   string
	Side   hex.Side
	Turns  int
	Dealt  int64
	Taken  int64
	Ticked int64
	Healed int64
	Hits   int
	Misses int
	Walled int
	Kills  int
	Fell   bool
}

// Tallies reads a battle's whole log and counts what each unit did.
//
// It is built from the events alone, with no access to the battle, which is the
// same constraint the event lines are held to. Anything a summary needs that the
// log cannot supply is something the log is missing.
func Tallies(events []battle.Event) []Tally {
	order := make([]string, 0, 10)
	byID := make(map[string]*Tally, 10)
	// A kill is credited to whoever last hurt the unit that fell, which the log
	// carries even though no event says "killed by".
	lastHarm := make(map[string]string, 10)

	touch := func(id string) *Tally {
		if id == "" {
			return nil
		}
		if existing, ok := byID[id]; ok {
			return existing
		}
		byID[id] = &Tally{ID: id, Name: id}
		order = append(order, id)
		return byID[id]
	}

	for _, event := range events {
		actor, target := touch(event.Actor), touch(event.Target)
		switch event.Kind {
		case battle.Started:
			actor.Side = event.Side
		case battle.TurnBegan:
			actor.Turns++
		case battle.StatusTicked:
			actor.Ticked += event.Amount
			actor.Taken += event.Amount
		case battle.Healed:
			actor.Healed += event.Amount
		case battle.Damaged:
			actor.Dealt += event.Amount
			actor.Hits++
			if target != nil {
				target.Taken += event.Amount
				lastHarm[target.ID] = actor.ID
			}
		case battle.Missed:
			actor.Misses++
		case battle.Blocked:
			if target != nil {
				target.Walled++
			}
		case battle.Died:
			actor.Fell = true
			if killer, known := lastHarm[actor.ID]; known && killer != actor.ID {
				if credited := byID[killer]; credited != nil {
					credited.Kills++
				}
			}
		}
	}

	out := make([]Tally, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out
}

// Summary renders the tallies as a table.
func Summary(events []battle.Event, tags map[string]string, names map[string]string) string {
	tallies := Tallies(events)
	if len(tallies) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("tag  unit                 turns    dealt    taken   effects   hits   miss  walled  kills\n")
	for _, tally := range tallies {
		name := names[tally.ID]
		if name == "" {
			name = tally.Name
		}
		if tally.Fell {
			name += " (fell)"
		}
		fmt.Fprintf(&b, "%-5s%-21s%6d%9d%9d%10d%7d%7d%8d%7d\n",
			tags[tally.ID], name, tally.Turns, tally.Dealt, tally.Taken,
			tally.Ticked, tally.Hits, tally.Misses, tally.Walled, tally.Kills)
	}
	return trimLines(b.String())
}

// Names maps unit ids to their display names, for a summary rendered from a log
// that carries ids rather than names.
func Names(units []*battle.Unit) map[string]string {
	out := make(map[string]string, len(units))
	for _, unit := range units {
		out[unit.ID] = unit.Name
	}
	return out
}

// NamesFromLog reads the display names out of a log's opening records, so a
// saved battle renders with the names it was fought under.
func NamesFromLog(events []battle.Event) map[string]string {
	out := make(map[string]string, 10)
	for _, event := range events {
		if event.Kind == battle.Started && event.Actor != "" && event.Name != "" {
			out[event.Actor] = event.Name
		}
	}
	return out
}

// TagsFromLog assigns tags using only the log's opening records, so a saved
// battle can be rendered without the roster it was fought with. It is the same
// constraint the event lines hold to, applied to the labels.
func TagsFromLog(events []battle.Event) map[string]string {
	counts := map[hex.Side]int{}
	letters := map[hex.Side]string{hex.SideAlly: "A", hex.SideEnemy: "E"}
	out := make(map[string]string, 10)
	for _, event := range events {
		if event.Kind != battle.Started || event.Actor == "" {
			continue
		}
		if _, already := out[event.Actor]; already {
			continue
		}
		counts[event.Side]++
		out[event.Actor] = fmt.Sprintf("%s%d", letters[event.Side], counts[event.Side])
	}
	return out
}
