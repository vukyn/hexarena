package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
)

func runOrigins(args []string) error {
	if len(args) > 0 && args[0] == "add" {
		return runOriginsAdd(args[1:])
	}
	lib, err := loadForListing("origins", args)
	if err != nil {
		return err
	}
	renderOrigins(os.Stdout, lib)
	return nil
}

// loadForListing is the whole of a listing subcommand: one flag, no operands.
//
// Rejecting a stray operand matters more than it looks. `hexforge origins --data
// x add ...` would otherwise print the catalog and silently drop the `add`,
// leaving someone convinced they had added a work they had not.
func loadForListing(name string, args []string) (*library, error) {
	set := newFlagSet(name)
	dir := dataFlag(set)
	operands, err := parseArgs(set, args)
	if err != nil {
		return nil, err
	}
	if len(operands) > 0 {
		return nil, fmt.Errorf("hexforge %s takes no arguments, got %v", name, operands)
	}
	return load(*dir)
}

func renderOrigins(out io.Writer, lib *library) {
	origins := lib.origins.All()
	if len(origins) == 0 {
		fmt.Fprintf(out, "no works in the catalog yet; add one with: hexforge origins add <id> --title T --medium anime\n")
		return
	}
	rendered := newTable("id", "medium", "year", "cast", "title").rightAlign(2, 3)
	for _, origin := range origins {
		year := ""
		if origin.Year != 0 {
			year = strconv.Itoa(origin.Year)
		}
		rendered.add(origin.ID, origin.Medium.String(), year,
			strconv.Itoa(len(lib.characters.OfOrigin(origin.ID))), origin.Title)
	}
	rendered.render(out)
	fmt.Fprintf(out, "\n%d works, media: %s\n", len(origins), strings.Join(cast.MediumNames(), " "))
}

func runOriginsAdd(args []string) error {
	set := newFlagSet("origins add")
	dir := dataFlag(set)
	title := set.String("title", "", "the work's title")
	medium := set.String("medium", "", "the kind of work: "+strings.Join(cast.MediumNames(), ", "))
	year := set.Int("year", 0, "the year it came out; leave it out if unknown")
	note := set.String("note", "", "anything worth recording about the work")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("usage: hexforge origins add <id> --title T --medium %s [--year N] [--note ...]",
			strings.Join(cast.MediumNames(), "|"))
	}
	id := operands[0]
	if *title == "" {
		return fmt.Errorf("origin %q needs --title", id)
	}
	if *medium == "" {
		return fmt.Errorf("origin %q needs --medium, one of %s", id, strings.Join(cast.MediumNames(), ", "))
	}
	parsed, err := cast.ParseMedium(*medium)
	if err != nil {
		return err
	}
	lib, err := load(*dir)
	if err != nil {
		return err
	}
	if _, clash := lib.origins.Get(id); clash {
		return fmt.Errorf("origin %q is already in the catalog", id)
	}
	// Append validates exactly as a load would, so a rejected entry never
	// reaches the file.
	updated, err := lib.origins.Append(cast.Origin{
		ID: id, Title: *title, Medium: parsed, Year: *year, Note: *note,
	})
	if err != nil {
		return err
	}
	if err := lib.writeOrigins(updated); err != nil {
		return err
	}
	fmt.Printf("added %s (%s) to %s\n", id, parsed, lib.dir)
	return nil
}

func runArchetypes(args []string) error {
	lib, err := loadForListing("archetypes", args)
	if err != nil {
		return err
	}
	renderArchetypes(os.Stdout, lib)
	return nil
}

func renderArchetypes(out io.Writer, lib *library) {
	archetypes := lib.archetypes.All()
	if len(archetypes) == 0 {
		fmt.Fprintln(out, "no presets declared")
		return
	}
	header := []string{"id", "col"}
	for _, kind := range progression.Kinds() {
		header = append(header, shortStat(kind))
	}
	header = append(header, "needs", "kit")
	rendered := newTable(header...).rightAlign(1)
	for _, archetype := range archetypes {
		row := []string{archetype.ID, strconv.Itoa(archetype.Column)}
		for _, kind := range progression.Kinds() {
			curve := archetype.Stats[kind]
			row = append(row, fmt.Sprintf("%d→%d", curve.Base, curve.Max))
		}
		row = append(row, demandColumn(archetype), strings.Join(archetype.Skills, " "))
		rendered.add(row...)
	}
	rendered.render(out)
	fmt.Fprintf(out, "\ncurves read base at level 1 → max at level %d\n", progression.LevelCap)
	fmt.Fprintf(out, "\"needs\" is the elements the kit demands, derived from the skills: a character\n"+
		"built from a preset must carry every one of them, so a preset needing two can be\n"+
		"carried by exactly that pair.\n")
	for _, archetype := range archetypes {
		capped := archetype.Stats.At(progression.LevelCap)
		fmt.Fprintf(out, "  %-11s %s\n", archetype.ID, archetype.Role)
		fmt.Fprintf(out, "  %-11s at the cap: %s, absorbs %d of the %d budget\n", "",
			capped, progression.EffectiveHP(capped, lib.rules), lib.limits.MaxEffectiveHP)
	}
}

// demandColumn is the elements a preset's kit insists on, or a dash when the kit
// is all neutral and any affinity carries it.
func demandColumn(archetype cast.Archetype) string {
	names := archetype.DemandNames()
	if len(names) == 0 {
		return "any"
	}
	return strings.Join(names, "+")
}

func runCast(args []string) error {
	lib, err := loadForListing("cast", args)
	if err != nil {
		return err
	}
	renderCast(os.Stdout, lib)
	return nil
}

func renderCast(out io.Writer, lib *library) {
	characters := lib.characters.All()
	if len(characters) == 0 {
		fmt.Fprintln(out, "no characters authored yet; create one with: hexforge new")
		return
	}
	rendered := newTable("id", "name", "origin", "archetype", "element", "stages", "image")
	for _, character := range characters {
		rendered.add(character.ID, character.Name, character.Origin, character.Archetype,
			character.Element.String(), stageSummary(character), character.Image)
	}
	rendered.render(out)
	fmt.Fprintf(out, "\n%d characters\n", len(characters))
}

// stageSummary writes an evolution line as the levels its stages take over at,
// which is the one thing a table has room for.
func stageSummary(character cast.Character) string {
	parts := make([]string, 0, len(character.Stages))
	for _, stage := range character.Stages {
		parts = append(parts, fmt.Sprintf("%s@%d", stage.Name, stage.MinLevel))
	}
	return strings.Join(parts, " → ")
}

// shortStat is the three letter column heading for a stat.
func shortStat(kind progression.Kind) string {
	switch kind {
	case progression.HP:
		return "hp"
	case progression.Attack:
		return "atk"
	case progression.Defense:
		return "def"
	case progression.Speed:
		return "spd"
	case progression.Accuracy:
		return "acc"
	case progression.Dodge:
		return "ddg"
	default:
		return kind.String()
	}
}
