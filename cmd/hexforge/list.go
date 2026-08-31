package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
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
func loadForListing(name string, args []string) (*forge.Library, error) {
	set := newFlagSet(name)
	dir := dataFlag(set)
	operands, err := parseArgs(set, args)
	if err != nil {
		return nil, err
	}
	if len(operands) > 0 {
		return nil, fmt.Errorf("hexforge %s takes no arguments, got %v", name, operands)
	}
	return forge.Load(*dir)
}

func renderOrigins(out io.Writer, lib *forge.Library) {
	origins := lib.Origins().All()
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
			strconv.Itoa(len(lib.Characters().OfOrigin(origin.ID))), origin.Title)
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
	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	// SaveOrigin validates exactly as a load would, so a rejected entry never
	// reaches the file.
	if err := lib.SaveOrigin(cast.Origin{
		ID: id, Title: *title, Medium: parsed, Year: *year, Note: *note,
	}); err != nil {
		return err
	}
	fmt.Printf("added %s (%s) to %s\n", id, parsed, lib.Dir())
	return nil
}

func runSpecies(args []string) error {
	if len(args) > 0 && args[0] == "add" {
		return runSpeciesAdd(args[1:])
	}
	lib, err := loadForListing("species", args)
	if err != nil {
		return err
	}
	renderSpecies(os.Stdout, lib)
	return nil
}

// renderSpecies lists what a unit can be, with how many characters are each.
//
// The count is the column worth having: a species nobody is is not an error --
// a catalog may be written before the cast that fills it -- but a skill kept for
// one nobody is cannot be carried by anybody, and this is where that shows.
func renderSpecies(out io.Writer, lib *forge.Library) {
	species := lib.Species().All()
	if len(species) == 0 {
		fmt.Fprintf(out, "nothing in the catalog yet; add one with: hexforge species add <id> --name N\n")
		return
	}
	rendered := newTable("id", "cast", "name", "note").rightAlign(1)
	for _, kind := range species {
		rendered.add(kind.ID,
			strconv.Itoa(len(lib.Characters().OfSpecies(kind.ID))), kind.Name, kind.Note)
	}
	rendered.render(out)
	fmt.Fprintf(out, "\n%d kinds\n", len(species))
}

func runSpeciesAdd(args []string) error {
	set := newFlagSet("species add")
	dir := dataFlag(set)
	name := set.String("name", "", "the word shown beside the id")
	note := set.String("note", "", "where the line is drawn, which is a judgement worth recording")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("usage: hexforge species add <id> --name N [--note ...]")
	}
	id := operands[0]
	if *name == "" {
		return fmt.Errorf("species %q needs --name", id)
	}
	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	// SaveSpecies validates exactly as a load would, so a rejected entry never
	// reaches the file.
	if err := lib.SaveSpecies(cast.Species{ID: id, Name: *name, Note: *note}); err != nil {
		return err
	}
	fmt.Printf("added %s (%s) to %s\n", id, *name, lib.Dir())
	return nil
}

func runStatuses(args []string) error {
	lib, err := loadForListing("statuses", args)
	if err != nil {
		return err
	}
	renderStatuses(os.Stdout, lib)
	return nil
}

// renderStatuses lists the timed effects, grouped by category, with what each
// one does spelled out under it.
//
// It is the listing that was missing, and the gap it leaves is not an author's
// but a reader's: every other listing names a status somewhere in its rows --
// a trait grants one, a skill inflicts one, a cleanse strips a whole category of
// them -- and none of them says what one is. Until this existed the answer was in
// statuses.json.
//
// Grouped rather than tabulated flat, because the grouping is the half a table
// cannot carry: a skill that strips a stat_debuff and a dot is unreadable to
// somebody who cannot see which statuses those two words cover.
//
// The sentences are i18n.Lang.DescribeStatus, in English here as the rest of this
// binary is, so the two front-ends and the battle prompt cannot come to describe
// one status three ways.
func renderStatuses(out io.Writer, lib *forge.Library) {
	groups := lib.Statuses().Grouped()
	if len(groups) == 0 {
		fmt.Fprintln(out, "no statuses declared")
		return
	}
	counted := 0
	for _, group := range groups {
		fmt.Fprintf(out, "\n%s -- %s\n", group.Category, i18n.En.StatusCategory(group.Category.String()))
		for _, kind := range group.Kinds {
			counted++
			fmt.Fprintf(out, "  %s\n", kind.ID)
			for _, line := range strings.Split(i18n.En.DescribeStatus(kind), "\n") {
				fmt.Fprintf(out, "    %s\n", line)
			}
		}
	}
	fmt.Fprintf(out, "\n%d statuses in %d groups\n", counted, len(groups))
	// Said once rather than under every status: it is true of all of them, and a
	// warning repeated fifteen times is a warning nobody finishes reading.
	fmt.Fprintf(out, "%s\n", i18n.En.Text(i18n.BlurbStatusCaveat))
}

func runArchetypes(args []string) error {
	lib, err := loadForListing("archetypes", args)
	if err != nil {
		return err
	}
	renderArchetypes(os.Stdout, lib)
	return nil
}

func runPassives(args []string) error {
	lib, err := loadForListing("passives", args)
	if err != nil {
		return err
	}
	renderPassives(os.Stdout, lib)
	return nil
}

// renderPassives lists the declared traits and what each puts on its holder.
//
// The statuses are named rather than their effect summarised: what a status does
// is the status book's own table, and restating a modifier here would be a second
// declaration of a number that lives somewhere else.
func renderPassives(out io.Writer, lib *forge.Library) {
	passives := lib.Passives().All()
	if len(passives) == 0 {
		fmt.Fprintln(out, "no passives declared")
		return
	}
	rendered := newTable("id", "name", "grants", "adds", "answers", "drains", "resists", "amplifies", "while")
	for _, held := range passives {
		grants := make([]string, 0, len(held.Grants))
		for _, grant := range held.Grants {
			if grant.Stacks > 1 {
				grants = append(grants, fmt.Sprintf("%s x%d", grant.Status, grant.Stacks))
				continue
			}
			grants = append(grants, grant.Status)
		}
		// A full thousand reads as "immune" rather than as a number: it is the
		// one value that is a fact about the holder rather than a share, and a
		// trait whose whole purpose is an immunity should not make a reader
		// divide by ten to find that out.
		resists := make([]string, 0, len(held.Resists))
		for _, resist := range held.Resists {
			if resist.Amount >= scale.Base {
				resists = append(resists, resist.Status+" (immune)")
				continue
			}
			// A negative share is a vulnerability, and it says so in the word
			// rather than by a minus sign in a column headed "resists" — the
			// reader would have to notice the sign to avoid reading it exactly
			// backwards.
			if resist.Amount < 0 {
				resists = append(resists,
					fmt.Sprintf("%s +%s taken", resist.Status, forge.Percent(-resist.Amount)))
				continue
			}
			resists = append(resists,
				fmt.Sprintf("%s %s", resist.Status, forge.Percent(resist.Amount)))
		}
		// The riders read the way a skill's own applications do, through the same
		// helper, so the two lists cannot come out described differently.
		adds := ""
		if len(held.Applies) > 0 {
			adds = forge.DescribeApplications(held.Applies)
		}
		// Both shares in one cell, each named, because a trait may carry either
		// alone and a bare percentage could be read as the wrong half: "poison
		// +30% effect" is a stronger tick and "+20% chance" is a poison that
		// lands more often, and they are worth different things in play.
		amplifies := make([]string, 0, len(held.Amplifies))
		for _, raise := range held.Amplifies {
			shares := make([]string, 0, 2)
			if raise.Effect > 0 {
				shares = append(shares, forge.Percent(raise.Effect)+" effect")
			}
			if raise.Chance > 0 {
				shares = append(shares, forge.Percent(raise.Chance)+" chance")
			}
			amplifies = append(amplifies,
				fmt.Sprintf("%s +%s", raise.Status, strings.Join(shares, " +")))
		}
		// The reply, in **one** cell. DescribePassive writes one sentence for the
		// whole of it deliberately -- what a reader wants is what attacking this
		// unit costs -- and a damage column filed away from a status column would
		// leave them adding it up across the table.
		//
		// Two of the six jobs a trait can hold had no column at all until this:
		// blood_thirst printed a row blank after its name and venom_blood's reply
		// was nowhere, so the listing reported less than the parser accepts.
		answers := ""
		if held.Replies.Answers() {
			parts := make([]string, 0, 2)
			if held.Replies.Power > 0 {
				// The stat the reply is priced against, not the word "attack".
				// It was that word while every reply was priced off attack, and
				// a listing that says attack while the engine reads defence is a
				// listing an author would tune against and be wrong.
				parts = append(parts, forge.Percent(held.Replies.Power)+
					" "+held.Replies.Scaling.Stat.String())
			}
			if len(held.Replies.Applies) > 0 {
				parts = append(parts, forge.DescribeApplications(held.Replies.Applies))
			}
			answers = strings.Join(parts, " + ")
		}
		// The share a trait takes back off its own damage, which is the other job
		// that rendered nowhere.
		drains := ""
		if held.Drains > 0 {
			drains = forge.Percent(held.Drains)
		}
		// A gate is only worth a column when there is one, and the share is read
		// as a percentage for the same reason every other permille is.
		gate := ""
		if held.While != nil {
			gate = "under " + forge.Percent(held.While.BelowHealth) + " health"
		}
		rendered.add(held.ID, held.Name, strings.Join(grants, ", "),
			adds, answers, drains, strings.Join(resists, ", "),
			strings.Join(amplifies, ", "), gate)
	}
	rendered.render(out)
	fmt.Fprint(out, "\na trait is in force from the moment its holder is enlisted, and nothing takes it off:\n"+
		"every status it grants is declared permanent, so it has no duration and no cleanse reaches it.\n"+
		"a resistance takes its share off an incoming application's chance, so a full share never lands;\n"+
		"a negative share is a vulnerability and puts its share back on, so the status lands more often.\n"+
		"what a trait adds rides on its holder's damaging skills only, through the same roll as their own.\n"+
		"an amplifier raises what its holder inflicts: the effect is the tick frozen on the stack, the chance is the roll.\n"+
		"an answer is what attacking the holder costs, and it lands after every strike of the attack rather than the first.\n"+
		"a drain is a share of the damage its holder deals, added to whatever the skill drains on its own.\n")
}

func renderArchetypes(out io.Writer, lib *forge.Library) {
	archetypes := lib.Archetypes().All()
	if len(archetypes) == 0 {
		fmt.Fprintln(out, "no presets declared")
		return
	}
	header := []string{"id", "col"}
	for _, kind := range progression.Kinds() {
		header = append(header, forge.ShortStat(kind))
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
		budget := lib.Budget(capped)
		fmt.Fprintf(out, "  %-11s %s\n", archetype.ID, archetype.Role)
		fmt.Fprintf(out, "  %-11s at the cap: %s, absorbs %d of the %d budget\n", "",
			capped, budget.Effective, budget.Max)
	}
}

// demandColumn is the elements a preset's kit insists on, or "any" when the kit
// is all neutral and any affinity carries it.
func demandColumn(archetype cast.Archetype) string {
	names := archetype.DemandNames()
	if len(names) == 0 {
		return "any"
	}
	return strings.Join(names, "+")
}

// runBuilds lists the late-game catalogue: which four skills and which trait a
// character is *for*.
//
// It is the last of the books to get a listing, and until it had one the only
// way to read the catalogue was to open builds.json — which is the one file here
// whose whole purpose is to be read by somebody choosing between directions.
func runBuilds(args []string) error {
	lib, err := loadForListing("builds", args)
	if err != nil {
		return err
	}
	renderBuilds(os.Stdout, lib)
	return nil
}

// renderBuilds draws the catalogue grouped the way it is declared: a character's
// directions are adjacent in the file, and a listing that sorted would separate
// the two things a reader is choosing between.
//
// The form is a column of its own rather than folded into the character, because
// on a line that FORKS it is part of the loadout — `poliwag.chorus` is Politoed's
// and the two beside it are Poliwrath's, and nothing else on the row says so.
func renderBuilds(out io.Writer, lib *forge.Library) {
	builds := lib.Builds()
	if len(builds) == 0 {
		fmt.Fprintln(out, "no builds authored yet; a character with fewer than two has none")
		return
	}
	rendered := newTable("id", "name", "character", "form", "trait", "kit", "intent")
	for _, build := range builds {
		rendered.add(build.ID, build.Name, build.Character, build.Stage,
			strings.Join(build.Passives, " "), strings.Join(build.Skills, " "), build.Intent)
	}
	rendered.render(out)

	// How many characters have a choice, because that is what the catalogue is
	// for: a character with one build has a kit rather than a decision, and one
	// with none is the honest case.
	chosen := 0
	for _, character := range lib.Characters().All() {
		if len(lib.BuildsOf(character.ID)) > 0 {
			chosen++
		}
	}
	fmt.Fprintf(out, "\n%d builds across %d of the %d characters authored\n",
		len(builds), chosen, len(lib.Characters().All()))
}

func runCast(args []string) error {
	lib, err := loadForListing("cast", args)
	if err != nil {
		return err
	}
	renderCast(os.Stdout, lib)
	return nil
}

func renderCast(out io.Writer, lib *forge.Library) {
	characters := lib.Characters().All()
	if len(characters) == 0 {
		fmt.Fprintln(out, "no characters authored yet; create one with: hexforge new")
		return
	}
	rendered := newTable("id", "name", "origin", "archetype", "element", "stages", "image")
	for _, character := range characters {
		rendered.add(character.ID, character.Name, character.Origin, character.Archetype,
			character.Element.String(), forge.StageSummary(character), character.Image)
	}
	rendered.render(out)
	fmt.Fprintf(out, "\n%d characters\n", len(characters))
}
