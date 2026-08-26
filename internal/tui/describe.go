package tui

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/i18n"
)

// Describe is what a skill does, in sentences, for somebody deciding whether to
// use it rather than somebody tuning it.
//
// # Why it is derived rather than authored
//
// Every number in it comes out of the skill itself. An authored description is
// free to drift the moment a value moves — a line reading "doubles" survives a
// bonus dropping from 1000 to 700 with nothing to catch it — and this engine
// already refused that trade once, in Archetype.Demands, which is derived from a
// kit precisely so it cannot describe a kit it no longer has. The cost is a
// uniform voice, which is the right thing to pay: a wrong number is worse than a
// flat sentence.
//
// The tuning numbers stay on the menu line above it. This says what happens.
//
// # Why the shares rather than the damage
//
// Power is a share of the caster's scaling stat, so "100% of attack" is true
// wherever it is read and a damage figure is only true for one caster against
// one target. A player comparing two skills is comparing the shares; the figure
// they would actually deal changes with every buff between now and the strike.
//
// # Why Vietnamese here and English everywhere else on the screen
//
// The rest of this package is English and staying that way until the whole
// battle screen is translated at once — see the note in Menu. This block is the
// one piece a player reads to *decide*, so it is worth being legible before the
// rest catches up, and the mixed screen is a stated cost rather than an
// oversight. It borrows i18n's tables rather than growing a second set of
// Vietnamese names: two vocabularies for one status is how they drift apart.
func Describe(declared skill.Skill, shapes *pattern.Book) string {
	lines := make([]string, 0, 4)
	if opening := describeOpening(declared, shapes); opening != "" {
		lines = append(lines, opening)
	}
	lines = append(lines, describeExtras(declared)...)
	if condition := describeCondition(declared); condition != "" {
		lines = append(lines, condition)
	}
	lines = append(lines, describeCosts(declared))
	return strings.Join(lines, "\n")
}

// describeOpening is the damage sentence, or the aim alone when a skill deals
// none: a skill with no power is not a weaker attack, it is a different kind of
// action, and opening with "0% of attack" would file it as the first.
func describeOpening(declared skill.Skill, shapes *pattern.Book) string {
	aim := describeAim(declared, shapes)
	if declared.Power <= 0 {
		// A skill that deals no damage still has to say what it reaches. Without
		// this a two-cell powder read exactly like a single-target one, because
		// every sentence after this one talks about "the target" and none of
		// them count how many there are.
		if declared.Target == skill.Self {
			return ""
		}
		return fmt.Sprintf("Nhắm %s.", aim)
	}
	stat := describeStat(declared.Scaling.Stat)
	var damage string
	if declared.StrikeCount() > 1 {
		damage = fmt.Sprintf("%d nhát, mỗi nhát %d%% %s (tổng %d%%)",
			declared.StrikeCount(), percent(declared.Power), stat,
			percent(declared.TotalPower()))
	} else {
		damage = fmt.Sprintf("%d%% %s", percent(declared.Power), stat)
	}
	sentence := fmt.Sprintf("Đánh %s, %s", aim, damage)
	if declared.Pierce > 0 {
		sentence += fmt.Sprintf(", xuyên %d%% giáp", percent(declared.Pierce))
	}
	return sentence + "."
}

// describeAim is who the skill reaches: the side it aims at, and how many cells
// its shape covers when that is more than one.
func describeAim(declared skill.Skill, shapes *pattern.Book) string {
	side := map[skill.Side]string{
		skill.Enemy: "đối phương", skill.Ally: "đồng đội",
		skill.Self: "bản thân", skill.All: "cả hai bên",
	}[declared.Target]
	cells := 1
	if shapes != nil {
		if shape, err := shapes.Lookup(declared.Pattern); err == nil {
			cells = shape.MaxTargets()
		}
	}
	if declared.Target == skill.Self || cells <= 1 {
		return "1 mục tiêu " + side
	}
	return fmt.Sprintf("%d ô %s", cells, side)
}

// describeExtras is everything a skill does that is not damage, one sentence
// each, in the order the fields are declared in.
func describeExtras(declared skill.Skill) []string {
	out := make([]string, 0, 4)
	if declared.Restores > 0 {
		out = append(out, fmt.Sprintf("Hồi máu bằng %d%% %s.",
			percent(declared.Restores), describeStat(declared.Scaling.Stat)))
	}
	if declared.Drains > 0 {
		out = append(out, fmt.Sprintf("Hút lại %d%% sát thương gây ra.", percent(declared.Drains)))
	}
	for _, application := range declared.Applies {
		out = append(out, fmt.Sprintf("Gây %s cho mục tiêu, %d%% khả năng%s.",
			glossed(application.Status), percent(application.Chance), stacksOf(application.Stacks)))
	}
	// One sentence for all of them rather than one each: a skill granting two
	// buffs grants them together, on the same turn, and two sentences read as
	// two separate things happening.
	if len(declared.SelfApplies) > 0 {
		names := make([]string, 0, len(declared.SelfApplies))
		for _, application := range declared.SelfApplies {
			names = append(names, glossed(application.Status)+stacksOf(application.Stacks))
		}
		out = append(out, fmt.Sprintf("Tự nhận %s.", strings.Join(names, " và ")))
	}
	if declared.Strips != nil {
		names := make([]string, 0, len(declared.Strips.Categories))
		for _, category := range declared.Strips.Categories {
			names = append(names, glossed(category.String()))
		}
		out = append(out, fmt.Sprintf("Gỡ %d lớp %s.",
			declared.Strips.Stacks, strings.Join(names, " và ")))
	}
	return out
}

// describeCondition is the amplifier, written as what the target must be rather
// than as a bonus figure: a player picking a skill is asking when it is worth
// using, and "+1000 power" answers a different question.
func describeCondition(declared skill.Skill) string {
	if declared.Requires == nil {
		return ""
	}
	clauses := make([]string, 0, 2)
	if declared.Requires.ReadsStatus() {
		clauses = append(clauses, fmt.Sprintf("đang dính %s%s",
			glossed(declared.Requires.Status), stacksOf(declared.Requires.MinStacks)))
	}
	if declared.Requires.ReadsHealth() {
		clauses = append(clauses, fmt.Sprintf("còn <=%d%% máu", percent(declared.Requires.BelowHealth)))
	}
	amplified := declared.PowerAgainst(declared.Requires.Satisfying())
	sentence := fmt.Sprintf("Mục tiêu %s: %d%% %s",
		strings.Join(clauses, " và "), percent(amplified*declared.StrikeCount()),
		describeStat(declared.Scaling.Stat))
	if declared.Requires.Consume {
		sentence += fmt.Sprintf(", và tiêu mất %s", glossed(declared.Requires.Status))
	}
	return sentence + "."
}

// describeCosts is the line every skill has: how far it reaches, how often it
// connects, and how long it is gone for.
func describeCosts(declared skill.Skill) string {
	parts := make([]string, 0, 3)
	if declared.Target == skill.Self {
		// A self-targeted skill has no range to state, and saying nothing at all
		// would leave the line opening with its cooldown as though the aim were
		// obvious. It is the one aim a reader cannot infer from the sentences
		// above, because those talk about what the caster receives.
		parts = append(parts, "bản thân")
	} else {
		parts = append(parts, fmt.Sprintf("tầm %d", declared.Range))
	}
	if declared.Power > 0 || len(declared.Applies) > 0 {
		parts = append(parts, fmt.Sprintf("%d%% trúng", percent(declared.Accuracy)))
	}
	if declared.Cooldown > 0 {
		parts = append(parts, fmt.Sprintf("hồi %d lượt", declared.Cooldown))
	} else {
		parts = append(parts, "dùng mọi lượt")
	}
	line := strings.Join(parts, " · ") + "."
	return strings.ToUpper(line[:1]) + line[1:]
}

// DescribePassive is what a trait does. A trait is not chosen, so this answers
// "what is this unit carrying" rather than "should I use it".
func DescribePassive(held passive.Passive) string {
	lines := make([]string, 0, 3)
	for _, grant := range held.Grants {
		lines = append(lines, fmt.Sprintf("Luôn mang %s%s.", glossed(grant.Status), stacksOf(grant.Stacks)))
	}
	for _, application := range held.Applies {
		lines = append(lines, fmt.Sprintf("Đòn của nó gây thêm %s, %d%% khả năng%s.",
			glossed(application.Status), percent(application.Chance), stacksOf(application.Stacks)))
	}
	for _, resistance := range held.Resists {
		if resistance.Amount >= scale.Base {
			lines = append(lines, fmt.Sprintf("Miễn hoàn toàn %s.", glossed(resistance.Status)))
			continue
		}
		lines = append(lines, fmt.Sprintf("Giảm %d%% khả năng dính %s.",
			percent(resistance.Amount), glossed(resistance.Status)))
	}
	if held.While != nil {
		lines = append(lines, fmt.Sprintf("Chỉ có hiệu lực khi còn <=%d%% máu.",
			percent(held.While.BelowHealth)))
	}
	return strings.Join(lines, "\n")
}

// percent turns a share in parts per thousand into whole percent. Truncation is
// deliberate: a share that is not a whole percent is a tuning detail, and the
// menu line above carries the exact figure for anybody who wants it.
func percent(permille int) int { return permille / (scale.Base / 100) }

// glossed is a data id under its Vietnamese name, falling back to the id.
// A miss is a bare id rather than a blank, the same answer every other listing
// gives, because a name nobody wrote is better read than guessed at.
func glossed(id string) string {
	if name := i18n.Vi.Gloss(id); name != "" {
		return name
	}
	return id
}

// describeStat is the stat a skill scales off. Nearly every skill scales off
// attack, so the name is looked up rather than assumed — a skill that scales off
// defence and said "attack" would read as a typo in the data.
func describeStat(stat progression.Kind) string {
	switch stat {
	case progression.Attack:
		return "công"
	case progression.Defense:
		return "thủ"
	case progression.Speed:
		return "tốc"
	case progression.Accuracy:
		return "chính xác"
	case progression.Dodge:
		return "né"
	default:
		return stat.String()
	}
}

// stacksOf is the stack count when it is worth saying. One stack is the unstated
// default everywhere else in the data, so writing "x1" here would make the
// common case look like a special one.
func stacksOf(stacks int) string {
	if stacks <= 1 {
		return ""
	}
	return fmt.Sprintf(" x%d", stacks)
}

// Detail is a description as a block a prompt can print: the skill's name, then
// its sentences, indented under a rule.
//
// It is here rather than at the call site because the framing is part of the
// reading. The menu above it is a table of figures, and a paragraph dropped into
// one without a break reads as another row that has lost its columns.
func Detail(declared skill.Skill, shapes *pattern.Book) string {
	title := declared.ID
	if name := i18n.Vi.SkillName(declared); name != "" && name != declared.ID {
		title = fmt.Sprintf("%s · %s", declared.ID, name)
	}
	return block(title, Describe(declared, shapes))
}

// DetailPassives is the traits a unit is carrying, as one block. A unit with
// none says so rather than printing an empty frame, because a blank answer to a
// question the player asked reads as the tool failing to answer it.
func DetailPassives(name string, held []passive.Passive) string {
	if len(held) == 0 {
		return block(name, "Không mang nội tại nào.")
	}
	parts := make([]string, 0, len(held))
	for _, one := range held {
		heading := one.ID
		if one.Name != "" {
			heading = fmt.Sprintf("%s · %s", one.ID, one.Name)
		}
		parts = append(parts, heading+"\n"+indent(DescribePassive(one), "  "))
	}
	return block(name, strings.Join(parts, "\n"))
}

func block(title, body string) string {
	var b strings.Builder
	b.WriteString("  " + title + "\n")
	b.WriteString("  " + strings.Repeat("-", 46) + "\n")
	b.WriteString(indent(body, "  "))
	return b.String()
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
