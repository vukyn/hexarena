package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
)

func runSkills(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return runSkillsAdd(args[1:])
		case "edit":
			return runSkillsEdit(args[1:])
		}
	}
	lib, err := loadForListing("skills", args)
	if err != nil {
		return err
	}
	renderSkills(os.Stdout, lib)
	return nil
}

func renderSkills(out io.Writer, lib *forge.Library) {
	skills := lib.Skills().Skills()
	rendered := newTable("id", "element", "tgt", "rng", "shape",
		"power", "hits", "acc", "cd", "who may carry it").rightAlign(3, 5, 6, 7, 8)
	for _, current := range skills {
		rendered.add(current.ID, current.Element.String(), current.Target.String(),
			strconv.Itoa(current.Range), current.Pattern,
			strconv.Itoa(current.Power), strconv.Itoa(current.StrikeCount()),
			strconv.Itoa(current.Accuracy), strconv.Itoa(current.Cooldown),
			forge.WhoMaySummary(current))
	}
	rendered.render(out)
	fmt.Fprintf(out, "\n%d skills; \"anyone\" is a neutral skill with no restriction, which is\n"+
		"the common pool every character can draw from\n", len(skills))
	fmt.Fprintf(out, "shapes: %s\n", strings.Join(lib.PatternNames(), " "))
	fmt.Fprintf(out, "statuses: %s\n", strings.Join(lib.StatusIDs(), " "))
}

// runSkillsAdd authors one skill.
//
// It takes the same shape as `origins add` — an id as the operand and the rest
// as flags — and the same prompting behaviour as `new`: a flag's value is used
// as it stands, a missing answer is asked for, and with nobody answering a field
// takes its default or errors naming the flag that would have supplied it.
//
// Every refusal it shows is internal/forge's. This file decides the order of the
// questions and nothing else, which is what keeps it from disagreeing with the
// full-screen client about what a legal skill is.
func runSkillsAdd(args []string) error {
	set := newFlagSet("skills add")
	dir := dataFlag(set)
	var given forge.SkillDraft
	set.StringVar(&given.ID, "id", "", "the skill's id")
	set.StringVar(&given.Element, "element", "", "the skill's element; neutral is the common pool")
	set.StringVar(&given.Target, "target", "", "who it aims at: enemy, ally or self")
	set.StringVar(&given.Range, "range", "", "how far it reaches, in hexes")
	set.StringVar(&given.Pattern, "pattern", "", "the shape it covers, by name in the pattern book")
	set.StringVar(&given.Power, "power", "", "power per strike, in parts per thousand")
	set.StringVar(&given.Strikes, "strikes", "", "how many times it lands")
	set.StringVar(&given.Accuracy, "accuracy", "", "its own chance to connect, in parts per thousand")
	set.StringVar(&given.Cooldown, "cooldown", "", "the caster's own turns before it can be used again")
	set.StringVar(&given.Applies, "applies", "",
		"statuses it inflicts, comma separated, each status:chance or status:chance:stacks")
	set.StringVar(&given.RestrictElements, "restrict-elements", "",
		"only these elements may carry it; leave empty for anyone")
	set.StringVar(&given.RestrictArchetypes, "restrict-archetypes", "",
		"only these role presets may carry it; leave empty for anyone")
	set.StringVar(&given.RestrictCharacters, "restrict-characters", "",
		"only these characters may carry it; leave empty for anyone")
	confirmed := set.Bool("yes", false, "write without asking for confirmation")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	switch len(operands) {
	case 0:
	case 1:
		if given.ID != "" && given.ID != operands[0] {
			return fmt.Errorf("the id was given twice, as %q and as --id %q", operands[0], given.ID)
		}
		given.ID = operands[0]
	default:
		return fmt.Errorf("usage: hexforge skills add [id] [flags]")
	}

	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	prompt := newPrompter()
	filled, err := fillSkill(given, lib, prompt)
	if err != nil {
		return err
	}
	built, err := filled.Resolve(lib)
	if err != nil {
		return err
	}

	renderSkill(os.Stdout, lib, built)
	if !*confirmed {
		if !prompt.interactive {
			return fmt.Errorf("stdin is not a terminal, so the write cannot be confirmed: pass --yes")
		}
		agreed, err := prompt.confirm("\nwrite " + built.ID + " to " + lib.SkillsPath() + "?")
		if err != nil {
			return err
		}
		if !agreed {
			fmt.Println("nothing written")
			return nil
		}
	}
	if err := lib.SaveSkill(built); err != nil {
		return err
	}
	for _, line := range lib.SaveSkillNotes(built) {
		fmt.Println(line)
	}
	return nil
}

// runSkillsEdit changes one skill that is already in the book.
//
// It takes the same flag names as `skills add`, and the difference between the
// two is entirely in what an absent flag means. On `add` an absent flag is a
// question the wizard asks or a default it takes; here it means leave the field
// alone, and that is why nothing below prompts for anything. An edit is a
// sentence about the fields it names.
//
// The trap this function exists to avoid: an absent flag and a flag set to zero
// are different answers. `--cooldown 0` clears a cooldown and no `--cooldown`
// leaves it, so which flags were given is read from FlagSet.Visit — which walks
// only the flags that were really on the command line — rather than by comparing
// a value against "" or against zero, which cannot tell the two apart. An
// explicitly empty `--restrict-elements ""` is a real answer too, and the only
// way this front-end can take a restriction back off a skill.
func runSkillsEdit(args []string) error {
	set := newFlagSet("skills edit")
	dir := dataFlag(set)
	// The flag values land in strings and the pointers are filled in afterwards
	// from Visit, because the flag package has no way to ask for "unset".
	var given forge.SkillDraft
	set.StringVar(&given.ID, "id", "", "refused: a skill's id cannot be edited")
	set.StringVar(&given.Element, "element", "", "the skill's element; neutral is the common pool")
	set.StringVar(&given.Target, "target", "", "who it aims at: enemy, ally or self")
	set.StringVar(&given.Range, "range", "", "how far it reaches, in hexes")
	set.StringVar(&given.Pattern, "pattern", "", "the shape it covers, by name in the pattern book")
	set.StringVar(&given.Power, "power", "", "power per strike, in parts per thousand")
	set.StringVar(&given.Strikes, "strikes", "", "how many times it lands")
	set.StringVar(&given.Accuracy, "accuracy", "", "its own chance to connect, in parts per thousand")
	set.StringVar(&given.Cooldown, "cooldown", "", "the caster's own turns before it can be used again")
	set.StringVar(&given.Applies, "applies", "",
		"statuses it inflicts, comma separated, each status:chance or status:chance:stacks")
	set.StringVar(&given.RestrictElements, "restrict-elements", "",
		"only these elements may carry it; an empty value clears the list")
	set.StringVar(&given.RestrictArchetypes, "restrict-archetypes", "",
		"only these role presets may carry it; an empty value clears the list")
	set.StringVar(&given.RestrictCharacters, "restrict-characters", "",
		"only these characters may carry it; an empty value clears the list")
	confirmed := set.Bool("yes", false, "write without asking for confirmation")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("usage: hexforge skills edit <id> [flags]")
	}
	id := operands[0]

	edit := editFrom(set, &given)
	if !edit.Names() {
		return fmt.Errorf("nothing to change: name at least one field, "+
			"as in `hexforge skills edit %s --power 1200`", id)
	}

	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	current, err := lib.Skills().Lookup(id)
	if err != nil {
		return err
	}
	built, err := edit.Draft(current).ResolveEdit(lib, id)
	if err != nil {
		return err
	}

	renderSkill(os.Stdout, lib, built)
	if !*confirmed {
		prompt := newPrompter()
		if !prompt.interactive {
			return fmt.Errorf("stdin is not a terminal, so the write cannot be confirmed: pass --yes")
		}
		agreed, err := prompt.confirm("\nchange " + id + " in " + lib.SkillsPath() + "?")
		if err != nil {
			return err
		}
		if !agreed {
			fmt.Println("nothing written")
			return nil
		}
	}
	change, err := lib.EditSkill(built)
	if err != nil {
		return err
	}
	for _, line := range lib.EditSkillNotes(change) {
		fmt.Println(line)
	}
	// The before-and-after is the point of editing through the tool rather than
	// in the file: the figures are what the goldens will show moved.
	if change.MovesDamage() {
		fmt.Printf("damage %s\n", change.DamageSummary())
	}
	return nil
}

// editFrom reads which flags were really given.
//
// FlagSet.Visit walks only those, which is the whole reason this exists rather
// than a run of `if value != ""`: it is what makes `--cooldown 0` a change to
// zero and no `--cooldown` no change at all, and what makes
// `--restrict-elements ""` clear a list instead of being mistaken for silence.
// The map is read by key and never ranged over, so nothing about the order it
// walks in reaches the output.
func editFrom(set *flag.FlagSet, given *forge.SkillDraft) forge.SkillEdit {
	var edit forge.SkillEdit
	fields := map[string]**string{
		"id": &edit.ID, "element": &edit.Element, "target": &edit.Target,
		"range": &edit.Range, "pattern": &edit.Pattern, "power": &edit.Power,
		"strikes": &edit.Strikes, "accuracy": &edit.Accuracy,
		"cooldown": &edit.Cooldown, "applies": &edit.Applies,
		"restrict-elements":   &edit.RestrictElements,
		"restrict-archetypes": &edit.RestrictArchetypes,
		"restrict-characters": &edit.RestrictCharacters,
	}
	values := map[string]*string{
		"id": &given.ID, "element": &given.Element, "target": &given.Target,
		"range": &given.Range, "pattern": &given.Pattern, "power": &given.Power,
		"strikes": &given.Strikes, "accuracy": &given.Accuracy,
		"cooldown": &given.Cooldown, "applies": &given.Applies,
		"restrict-elements":   &given.RestrictElements,
		"restrict-archetypes": &given.RestrictArchetypes,
		"restrict-characters": &given.RestrictCharacters,
	}
	set.Visit(func(flagged *flag.Flag) {
		field, ours := fields[flagged.Name]
		if !ours {
			return
		}
		*field = values[flagged.Name]
	})
	return edit
}

// The defaults a skill takes when nobody says otherwise.
//
// They are the shape of an ordinary single-target attack, and the element among
// them is the one worth spelling out: neutral is the common pool, so a skill
// authored without an opinion about its element is one every character can take.
// Power and accuracy have no default on purpose — both are balance, and a
// default would let an unattended run write a number nobody chose.
const (
	defaultSkillElement  = "neutral"
	defaultSkillTarget   = "enemy"
	defaultSkillPattern  = "single"
	defaultSkillRange    = "1"
	defaultSkillStrikes  = "1"
	defaultSkillCooldown = "0"
)

// fillSkill asks for whatever the flags left out.
func fillSkill(given forge.SkillDraft, lib *forge.Library, prompt *prompter) (forge.SkillDraft, error) {
	filled := given
	ask := func(field *string, q question) error {
		q.given = *field
		answer, err := prompt.answer(q)
		if err != nil {
			return err
		}
		*field = answer
		return nil
	}

	if err := ask(&filled.ID, question{
		flag: "id", prompt: "id", validate: lib.ValidateNewSkillID,
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	if err := ask(&filled.Element, question{
		flag: "element", prompt: "element", preset: defaultSkillElement,
		validate: func(answer string) error {
			_, err := forge.ParseElement(answer)
			return err
		},
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	if err := ask(&filled.Target, question{
		flag: "target", prompt: "target, one of enemy ally self", preset: defaultSkillTarget,
		validate: func(answer string) error {
			_, err := forge.ParseTarget(answer)
			return err
		},
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	if filled.Pattern == "" {
		prompt.note("shapes: %s\n", strings.Join(lib.PatternNames(), " "))
	}
	if err := ask(&filled.Pattern, question{
		flag: "pattern", prompt: "shape", preset: defaultSkillPattern,
		validate: func(answer string) error {
			_, err := lib.LookupPattern(answer)
			return err
		},
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	for _, numeric := range []struct {
		field  *string
		flag   string
		prompt string
		preset string
	}{
		{&filled.Range, "range", "range in hexes, 0 for a self skill", defaultSkillRange},
		{&filled.Power, "power", "power per strike, in parts per thousand", ""},
		{&filled.Strikes, "strikes", "strikes", defaultSkillStrikes},
		{&filled.Accuracy, "accuracy", "accuracy, in parts per thousand", ""},
		{&filled.Cooldown, "cooldown", "cooldown, in the caster's own turns", defaultSkillCooldown},
	} {
		if err := ask(numeric.field, question{
			flag: numeric.flag, prompt: numeric.prompt, preset: numeric.preset,
			validate: func(answer string) error {
				_, err := forge.ParseNumber(answer)
				return err
			},
		}); err != nil {
			return forge.SkillDraft{}, err
		}
	}
	if filled.Applies == "" {
		prompt.note("statuses: %s\n", strings.Join(lib.StatusIDs(), " "))
	}
	if err := ask(&filled.Applies, question{
		flag: "applies", prompt: "statuses inflicted, status:chance[:stacks]", optional: true,
		validate: func(answer string) error {
			_, err := lib.ParseApplications(answer)
			return err
		},
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	// The three allowlists are optional and empty means the common pool, which
	// is why none of them has a default: a default here would quietly narrow
	// every skill authored without an opinion.
	if err := ask(&filled.RestrictElements, question{
		flag: "restrict-elements", prompt: "elements allowed to carry it, empty for anyone",
		optional: true,
		validate: func(answer string) error {
			for _, name := range forge.SplitList(answer) {
				if _, err := forge.ParseElement(name); err != nil {
					return err
				}
			}
			return nil
		},
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	if err := ask(&filled.RestrictArchetypes, question{
		flag: "restrict-archetypes", prompt: "role presets allowed to carry it, empty for anyone",
		optional: true, validate: lib.ValidateRestrictedArchetypes,
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	if err := ask(&filled.RestrictCharacters, question{
		flag: "restrict-characters", prompt: "characters allowed to carry it, empty for anyone",
		optional: true, validate: lib.ValidateRestrictedCharacters,
	}); err != nil {
		return forge.SkillDraft{}, err
	}
	return filled, nil
}

// renderSkill prints a skill and what it is worth, which is what the wizard
// shows before writing.
//
// The damage row is the point of it. A skill is balance, so the question before
// a write is not whether the answers parse but what they do — and the figures
// come from forge.Library.PreviewDamage, which is combat.Rules.Damage against
// the reference pair skills.golden's own table is measured from.
func renderSkill(out io.Writer, lib *forge.Library, built skill.Skill) {
	label := func(name, format string, args ...any) {
		fmt.Fprintf(out, "  %-10s %s\n", name, fmt.Sprintf(format, args...))
	}
	fmt.Fprintf(out, "\n%s — %s, %s\n", built.ID, built.Element, built.Target)
	label("reach", "range %d, %s", built.Range, built.Pattern)
	// A support skill declares no power, and "0 (0%)" says nothing the zero did
	// not, so the reading is dropped rather than printed empty.
	power := strconv.Itoa(built.Power)
	if built.Power != 0 {
		power += " (" + forge.Percent(built.Power) + ")"
	}
	label("power", "%s x%d, accuracy %d (%s), cooldown %d",
		power, built.StrikeCount(),
		built.Accuracy, forge.Percent(built.Accuracy), built.Cooldown)
	if len(built.Applies) > 0 {
		label("inflicts", "%s", forge.DescribeApplications(built.Applies))
	}
	label("carried by", "%s", forge.WhoMaySummary(built))
	preview := lib.PreviewDamage(built)
	label("damage", "%d per strike, %d in total, against %d attack and %d defence",
		preview.PerStrike, preview.Total, preview.Attack, preview.Defense)
	if preview.Amplified > 0 {
		label("", "%d with its condition holding", preview.Amplified)
	}
	label("", "the same reference the skills table is measured from, at level %d",
		progression.LevelCap)
}
