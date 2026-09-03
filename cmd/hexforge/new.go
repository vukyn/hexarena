package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// question is one field of a draft and everything needed to fill it: the flag
// that prefills it, the wording of the prompt, the default Enter accepts, and
// the check an answer has to pass.
type question struct {
	// flag is the flag that prefills this field. It is carried so that a
	// non-interactive run can name the flag that is missing instead of hanging
	// on a prompt nobody can answer.
	flag   string
	prompt string
	// given is the flag's value. A non-empty one means the answer is already
	// in, and the wizard asks only for what is still missing.
	given string
	// preset is the default shown at the prompt, which Enter accepts. It is
	// where an archetype's suggestion arrives.
	preset   string
	optional bool
	validate func(string) error
}

// prompter asks the questions a draft still has holes for.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
	// interactive is false when stdin is not a terminal. Then there is nobody
	// to ask, and a missing answer is an error naming its flag rather than a
	// prompt into a pipe.
	interactive bool
}

func newPrompter() *prompter {
	return &prompter{in: bufio.NewReader(os.Stdin), out: os.Stdout, interactive: stdinIsTerminal()}
}

// stdinIsTerminal reports whether there is plausibly a person on the other end.
//
// It is only a first guess, and it has to be. os.Stdin.Stat cannot tell a
// terminal from /dev/null — both are character devices — so a run with stdin
// closed or redirected from /dev/null looks interactive here and is not. EOF on
// the first read is the authoritative signal, and answer acts on it.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// answer fills one field.
//
// A flag's value is validated and used as it stands: a bad flag is a mistake to
// report, not a question to re-ask. A prompted answer is validated as it is
// entered and asked again on a failure, because failing at the end of a wizard
// throws away every answer that was right.
func (p *prompter) answer(q question) (string, error) {
	if q.given != "" {
		if q.validate != nil {
			if err := q.validate(q.given); err != nil {
				return "", fmt.Errorf("--%s: %w", q.flag, err)
			}
		}
		return q.given, nil
	}
	if !p.interactive {
		return p.unattended(q)
	}
	for {
		if q.preset != "" {
			fmt.Fprintf(p.out, "%s [%s]: ", q.prompt, q.preset)
		} else if q.optional {
			fmt.Fprintf(p.out, "%s (optional): ", q.prompt)
		} else {
			fmt.Fprintf(p.out, "%s: ", q.prompt)
		}
		line, err := p.in.ReadString('\n')
		if err != nil && strings.TrimSpace(line) == "" {
			// Nobody is answering. Reading again would ask every remaining
			// question into a closed pipe and fail on the first one that had a
			// perfectly good default, which is the worst of both behaviours, so
			// the rest of the session is unattended from here.
			p.interactive = false
			return p.unattended(q)
		}
		answer := strings.TrimSpace(line)
		if answer == "" {
			answer = q.preset
		}
		if answer == "" {
			if q.optional {
				return "", nil
			}
			fmt.Fprintf(p.out, "  --%s has no default, so it needs an answer\n", q.flag)
			continue
		}
		if q.validate != nil {
			if err := q.validate(answer); err != nil {
				fmt.Fprintf(p.out, "  %v\n", err)
				continue
			}
		}
		return answer, nil
	}
}

// unattended fills a field with nobody to ask.
//
// A field with a default takes it — a preset-supplied value is not missing, and
// the wizard's whole promise is that it asks only for what is. A field with no
// default is an error naming the flag that would have supplied it, which is the
// one thing a script can act on.
func (p *prompter) unattended(q question) (string, error) {
	if q.preset == "" {
		if q.optional {
			return "", nil
		}
		return "", fmt.Errorf(
			"nothing is answering prompts (stdin is not a terminal, or it has ended) and --%s was not given",
			q.flag)
	}
	if q.validate != nil {
		if err := q.validate(q.preset); err != nil {
			return "", fmt.Errorf("the default for --%s does not pass: %w", q.flag, err)
		}
	}
	return q.preset, nil
}

// confirm asks a yes or no question, defaulting to no.
func (p *prompter) confirm(question string) (bool, error) {
	fmt.Fprintf(p.out, "%s [y/N]: ", question)
	line, err := p.in.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return false, fmt.Errorf("reading the confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (p *prompter) note(format string, args ...any) {
	if p.interactive {
		fmt.Fprintf(p.out, format, args...)
	}
}

func runNew(args []string) error {
	set := newFlagSet("new")
	dir := dataFlag(set)
	var given forge.Draft
	set.StringVar(&given.ID, "id", "", "the character's id; one dot may separate the origin from the name")
	set.StringVar(&given.Name, "name", "", "the display name")
	set.StringVar(&given.Origin, "origin", "", "the id of the work it is borrowed from")
	set.StringVar(&given.Archetype, "archetype", "", "the role preset to start from")
	set.StringVar(&given.Image, "image", "", "relative path to the art, ending .svg or .png")
	set.StringVar(&given.Element, "element", "", "one element, or two separated by a slash")
	set.StringVar(&given.Bio, "bio", "", "a line or two about who this is")
	set.StringVar(&given.Skills, "skills", "", "comma separated kit; defaults to the archetype's")
	set.StringVar(&given.Species, "species", "",
		"comma separated kinds of creature it is; empty is nothing in particular")
	for _, kind := range progression.Kinds() {
		set.StringVar(&given.Stats[kind], forge.ShortStat(kind),
			"", fmt.Sprintf("override the %s curve, written base:max", kind))
	}
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
		return fmt.Errorf("usage: hexforge new [id] [flags]")
	}

	lib, err := forge.Load(*dir)
	if err != nil {
		return err
	}
	prompt := newPrompter()
	filled, err := fill(given, lib, prompt)
	if err != nil {
		return err
	}
	character, err := filled.Resolve(lib)
	if err != nil {
		return err
	}

	renderCharacter(os.Stdout, lib, character, 1, progression.LevelCap)
	target := lib.CastPath()
	if !*confirmed {
		if !prompt.interactive {
			return fmt.Errorf("stdin is not a terminal, so the write cannot be confirmed: pass --yes")
		}
		agreed, err := prompt.confirm("\nwrite " + character.ID + " to " + target + "?")
		if err != nil {
			return err
		}
		if !agreed {
			fmt.Println("nothing written")
			return nil
		}
	}
	if err := lib.SaveCharacter(character); err != nil {
		return err
	}
	for _, line := range lib.SaveNotes(character) {
		fmt.Println(line)
	}
	return nil
}

// fill asks for whatever the flags left out, in an order that lets each answer
// supply the next one's default: the archetype has to be settled before its
// curve and its kit can be offered.
func fill(given forge.Draft, lib *forge.Library, prompt *prompter) (forge.Draft, error) {
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
		flag: "id", prompt: "id", validate: lib.ValidateNewID,
	}); err != nil {
		return forge.Draft{}, err
	}

	if err := ask(&filled.Name, question{
		flag: "name", prompt: "display name", validate: forge.ValidateName,
	}); err != nil {
		return forge.Draft{}, err
	}

	if filled.Origin == "" {
		prompt.note("origins: %s\n", strings.Join(lib.OriginIDs(), " "))
	}
	if err := ask(&filled.Origin, question{
		flag: "origin", prompt: "origin", validate: lib.ValidateOrigin,
	}); err != nil {
		return forge.Draft{}, err
	}

	if filled.Archetype == "" {
		for _, preset := range lib.Archetypes().All() {
			prompt.note("  %-11s %s\n", preset.ID, forge.PresetSummary(preset))
		}
	}
	if err := ask(&filled.Archetype, question{
		flag: "archetype", prompt: "archetype", validate: lib.ValidateArchetype,
	}); err != nil {
		return forge.Draft{}, err
	}
	// Safe: the answer was validated against the book.
	archetype, _ := lib.Archetypes().Get(filled.Archetype)

	if err := ask(&filled.Image, question{
		flag: "image", prompt: "art path", preset: forge.SuggestedImage(filled.ID),
		validate: cast.ValidateImagePath,
	}); err != nil {
		return forge.Draft{}, err
	}

	// What it is comes before the kit, because a skill kept for a lineage is one
	// of the things the kit check reads -- see forge.Carrier. It is optional, and
	// an empty answer is the ordinary character rather than a hole: most of a
	// cast is nothing in particular.
	if err := ask(&filled.Species, question{
		flag: "species", prompt: "what it is, comma separated, empty for nothing in particular",
		optional: true, validate: lib.ValidateSpeciesList,
	}); err != nil {
		return forge.Draft{}, err
	}

	// The kit is settled before the element, because the kit is what decides
	// which elements are legal. Asking the other way round means either
	// validating the element against a preset the author is about to replace, or
	// accepting an answer that the write then refuses.
	//
	// The kit's own check is against everything already answered — the id and
	// the preset, which a skill's restriction may name — and not against the
	// element, which has not been asked yet. forge.Carrier is what says an
	// unanswered fact restricts nothing.
	if err := ask(&filled.Skills, question{
		flag: "skills", prompt: "kit", preset: strings.Join(archetype.Skills, ","),
		validate: func(answer string) error {
			return lib.ValidateKitFor(answer, filled.Carrier())
		},
	}); err != nil {
		return forge.Draft{}, err
	}
	// Safe: every name in the answer was looked up while it was validated.
	kit, err := lib.LookupKit(forge.SplitList(filled.Skills))
	if err != nil {
		return forge.Draft{}, err
	}

	if err := ask(&filled.Element, question{
		flag:   "element",
		prompt: "element, one or two separated by a slash (" + forge.DemandSummary(kit) + ")",
		validate: func(answer string) error {
			return lib.ValidateElement(answer, kit)
		},
	}); err != nil {
		return forge.Draft{}, err
	}

	for _, kind := range progression.Kinds() {
		if err := ask(&filled.Stats[kind], question{
			flag: forge.ShortStat(kind), prompt: kind.String() + " curve, base:max",
			preset: forge.FormatCurve(archetype.Stats[kind]),
			validate: func(answer string) error {
				return forge.ValidateCurve(kind, answer)
			},
		}); err != nil {
			return forge.Draft{}, err
		}
	}

	if err := ask(&filled.Bio, question{
		flag: "bio", prompt: "biography", optional: true,
	}); err != nil {
		return forge.Draft{}, err
	}
	return filled, nil
}

// renderCharacter prints a character resolved at each of the given levels,
// with what it spends of the effective-health budget. It is what the wizard
// shows before writing and what show prints.
func renderCharacter(out io.Writer, lib *forge.Library, character cast.Character, levels ...int) {
	origin, known := lib.Origins().Get(character.Origin)
	title := character.Origin
	if known {
		title = fmt.Sprintf("%s (%s, %s)", origin.Title, origin.Medium, character.Origin)
	}
	label := func(name, format string, args ...any) {
		fmt.Fprintf(out, "  %-10s %s\n", name, fmt.Sprintf(format, args...))
	}
	fmt.Fprintf(out, "\n%s — %s\n", character.ID, character.Name)
	label("from", "%s", title)
	label("tuned from", "%s", character.Archetype)
	label("element", "%s", character.Element)
	// The learnset, with its gates, through the same summary the traits use:
	label("kit", "%s", forge.UnlockSummary(character.Skills))
	// Only when there are any, on the same terms as the traits line below: most
	// of a cast is nothing in particular, and a row saying so on every one of
	// them says nothing.
	if len(character.Species) > 0 {
		label("species", "%s", strings.Join(character.Species, " "))
	}
	// Only when there are any. A "traits: none" line on every character in a
	// cast that holds none is a row that says nothing on every screen it is on.
	if len(character.Passives) > 0 {
		label("traits", "%s", forge.UnlockSummary(character.Passives))
	}
	label("art", "%s", character.Image)
	if art := character.Art(); len(art) > 1 {
		// Only when a form has its own. Printing "1 picture" for the ordinary
		// character would be noise on every line of every listing.
		for _, entry := range art[1:] {
			label("", "%s for %s", entry.Image, entry.Stage)
		}
	}
	label("stages", "%s", forge.StageSummary(character))
	if character.Bio != "" {
		label("bio", "%s", character.Bio)
	}
	for _, level := range levels {
		// A row per arm, because a line that forks has no single furthest and
		// Resolve refuses to pick one — which this used to print as the row's
		// own value, so `hexforge show pokemon.poliwag` ended on a refusal
		// where the stats belong. The screens answered the same refusal with a
		// chooser; a one-shot print has nowhere to hold a choice, so it prints
		// both. FurthestAt returns exactly one stage on a line that does not
		// fork, so nothing about an ordinary character changes.
		arms, err := character.FurthestAt(level)
		if err != nil {
			label(fmt.Sprintf("level %d", level), "%v", err)
			continue
		}
		for _, arm := range arms {
			values, stage, err := character.Resolve(level, arm.Name)
			if err != nil {
				label(fmt.Sprintf("level %d", level), "%v", err)
				continue
			}
			budget := lib.Budget(values)
			label(fmt.Sprintf("level %d", level), "%s", values)
			label("", "stage %q shows %s and absorbs %d of the %d effective-health budget, %d to spare",
				stage.Name, character.StageArt(stage),
				budget.Effective, budget.Max, budget.Headroom)
		}
	}
}
