package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
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
	var given draft
	set.StringVar(&given.id, "id", "", "the character's id; one dot may separate the origin from the name")
	set.StringVar(&given.name, "name", "", "the display name")
	set.StringVar(&given.origin, "origin", "", "the id of the work it is borrowed from")
	set.StringVar(&given.archetype, "archetype", "", "the role preset to start from")
	set.StringVar(&given.image, "image", "", "relative path to the art, ending .svg or .png")
	set.StringVar(&given.element, "element", "", "one element, or two separated by a slash")
	set.StringVar(&given.bio, "bio", "", "a line or two about who this is")
	set.StringVar(&given.skills, "skills", "", "comma separated kit; defaults to the archetype's")
	for _, kind := range progression.Kinds() {
		set.StringVar(&given.stats[kind], shortStat(kind),
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
		if given.id != "" && given.id != operands[0] {
			return fmt.Errorf("the id was given twice, as %q and as --id %q", operands[0], given.id)
		}
		given.id = operands[0]
	default:
		return fmt.Errorf("usage: hexforge new [id] [flags]")
	}

	lib, err := load(*dir)
	if err != nil {
		return err
	}
	prompt := newPrompter()
	filled, err := fill(given, lib, prompt)
	if err != nil {
		return err
	}
	character, err := filled.resolve(lib)
	if err != nil {
		return err
	}

	renderCharacter(os.Stdout, lib, character, 1, progression.LevelCap)
	target := filepath.Join(lib.dir, castFile)
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
	updated, err := lib.characters.Append(lib.castDeps(), character)
	if err != nil {
		return err
	}
	if err := lib.writeCast(updated); err != nil {
		return err
	}
	fmt.Printf("wrote %s to %s\n", character.ID, target)
	if !lib.imageExists(character.Image) {
		fmt.Printf("note: %s is not there yet; hexforge check will keep saying so until it is\n",
			lib.imagePath(character.Image))
	}
	fmt.Printf("note: the game boots from the embedded copy, so rebuild before this reaches a battle\n")
	return nil
}

// fill asks for whatever the flags left out, in an order that lets each answer
// supply the next one's default: the archetype has to be settled before its
// curve and its kit can be offered.
func fill(given draft, lib *library, prompt *prompter) (draft, error) {
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

	if err := ask(&filled.id, question{
		flag: "id", prompt: "id",
		validate: func(answer string) error {
			if err := cast.ValidateID(answer); err != nil {
				return err
			}
			if _, clash := lib.characters.Get(answer); clash {
				return fmt.Errorf("character %q is already in the cast", answer)
			}
			return nil
		},
	}); err != nil {
		return draft{}, err
	}

	if err := ask(&filled.name, question{
		flag: "name", prompt: "display name",
		validate: func(answer string) error {
			if strings.TrimSpace(answer) == "" {
				return fmt.Errorf("a character needs a display name")
			}
			return nil
		},
	}); err != nil {
		return draft{}, err
	}

	if filled.origin == "" {
		prompt.note("origins: %s\n", strings.Join(originIDs(lib), " "))
	}
	if err := ask(&filled.origin, question{
		flag: "origin", prompt: "origin",
		validate: func(answer string) error {
			if _, known := lib.origins.Get(answer); !known {
				return fmt.Errorf("unknown origin %q, add it with %q", answer, "hexforge origins add "+answer)
			}
			return nil
		},
	}); err != nil {
		return draft{}, err
	}

	if filled.archetype == "" {
		for _, preset := range lib.archetypes.All() {
			prompt.note("  %-11s %s\n", preset.ID, presetDemand(preset))
		}
	}
	if err := ask(&filled.archetype, question{
		flag: "archetype", prompt: "archetype",
		validate: func(answer string) error {
			if _, known := lib.archetypes.Get(answer); !known {
				return fmt.Errorf("unknown archetype %q, want one of %s",
					answer, strings.Join(lib.archetypes.IDs(), ", "))
			}
			return nil
		},
	}); err != nil {
		return draft{}, err
	}
	// Safe: the answer was validated against the book.
	archetype, _ := lib.archetypes.Get(filled.archetype)

	if err := ask(&filled.image, question{
		flag: "image", prompt: "art path", preset: suggestedImage(filled.id),
		validate: cast.ValidateImagePath,
	}); err != nil {
		return draft{}, err
	}

	// The kit is settled before the element, because the kit is what decides
	// which elements are legal. Asking the other way round means either
	// validating the element against a preset the author is about to replace, or
	// accepting an answer that the write then refuses.
	if err := ask(&filled.skills, question{
		flag: "skills", prompt: "kit", preset: strings.Join(archetype.Skills, ","),
		validate: func(answer string) error {
			named := splitList(answer)
			if len(named) == 0 {
				return fmt.Errorf("a character with no skills would have nothing to do on its turn")
			}
			seen := make(map[string]bool, len(named))
			for _, id := range named {
				if _, err := lib.skills.Lookup(id); err != nil {
					return err
				}
				if seen[id] {
					return fmt.Errorf("%q is named twice", id)
				}
				seen[id] = true
			}
			return nil
		},
	}); err != nil {
		return draft{}, err
	}
	// Safe: every name in the answer was looked up while it was validated.
	kit, err := lookupKit(lib, splitList(filled.skills))
	if err != nil {
		return draft{}, err
	}

	if err := ask(&filled.element, question{
		flag: "element", prompt: "element, one or two separated by a slash" + demandHint(kit),
		validate: func(answer string) error {
			affinity, err := parseAffinity(answer)
			if err != nil {
				return err
			}
			if err := lib.chart.ValidateAffinity(affinity); err != nil {
				return err
			}
			// The same predicate the write goes through, applied early so a
			// wrong answer costs one line instead of the whole session.
			for _, carried := range kit {
				if !skill.CanCarry(affinity, carried) {
					return fmt.Errorf("%s cannot carry %q, which is %s",
						affinity, carried.ID, carried.Element)
				}
			}
			return nil
		},
	}); err != nil {
		return draft{}, err
	}

	for _, kind := range progression.Kinds() {
		if err := ask(&filled.stats[kind], question{
			flag: shortStat(kind), prompt: kind.String() + " curve, base:max",
			preset: formatCurve(archetype.Stats[kind]),
			validate: func(answer string) error {
				curve, err := parseCurve(answer)
				if err != nil {
					return err
				}
				return curve.Validate(kind)
			},
		}); err != nil {
			return draft{}, err
		}
	}

	if err := ask(&filled.bio, question{
		flag: "bio", prompt: "biography", optional: true,
	}); err != nil {
		return draft{}, err
	}
	return filled, nil
}

// lookupKit resolves a list of skill ids against the book.
func lookupKit(lib *library, named []string) ([]skill.Skill, error) {
	out := make([]skill.Skill, 0, len(named))
	for _, id := range named {
		known, err := lib.skills.Lookup(id)
		if err != nil {
			return nil, err
		}
		out = append(out, known)
	}
	return out, nil
}

// demandHint says which elements a kit will insist on, so the prompt answers the
// question the author is about to get wrong. The demand is derived from the kit
// actually chosen, not from the preset, because the two differ the moment
// --skills is used.
func demandHint(kit []skill.Skill) string {
	demanded := skill.Demands(kit)
	if len(demanded) == 0 {
		return " (this kit is all neutral, so any element carries it)"
	}
	names := make([]string, 0, len(demanded))
	for _, member := range demanded {
		names = append(names, member.String())
	}
	return fmt.Sprintf(" (this kit needs %s)", strings.Join(names, " and "))
}

// suggestedImage proposes where a character's art would live, following the id.
// It is only a default: any relative path ending .svg or .png is allowed.
func suggestedImage(id string) string {
	if id == "" {
		return ""
	}
	folder, name, split := strings.Cut(id, ".")
	if !split {
		return path.Join("assets", folder+".svg")
	}
	return path.Join("assets", folder, name+".svg")
}

func originIDs(lib *library) []string {
	origins := lib.origins.All()
	out := make([]string, 0, len(origins))
	for _, origin := range origins {
		out = append(out, origin.ID)
	}
	return out
}

// renderCharacter prints a character resolved at each of the given levels,
// with what it spends of the effective-health budget. It is what the wizard
// shows before writing and what show prints.
func renderCharacter(out io.Writer, lib *library, character cast.Character, levels ...int) {
	origin, known := lib.origins.Get(character.Origin)
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
	label("kit", "%s", strings.Join(character.Skills, " "))
	label("art", "%s", character.Image)
	label("stages", "%s", stageSummary(character))
	if character.Bio != "" {
		label("bio", "%s", character.Bio)
	}
	for _, level := range levels {
		values, stage, err := character.Resolve(level)
		if err != nil {
			label(fmt.Sprintf("level %d", level), "%v", err)
			continue
		}
		effective := progression.EffectiveHP(values, lib.rules)
		label(fmt.Sprintf("level %d", level), "%s", values)
		label("", "stage %q absorbs %d of the %d effective-health budget, %d to spare",
			stage.Name, effective, lib.limits.MaxEffectiveHP, lib.limits.MaxEffectiveHP-effective)
	}
}

// presetDemand describes a preset for the archetype prompt: what its kit is and
// which elements that kit will insist on.
func presetDemand(preset cast.Archetype) string {
	names := preset.DemandNames()
	if len(names) == 0 {
		return fmt.Sprintf("%s (any element)", strings.Join(preset.Skills, " "))
	}
	return fmt.Sprintf("%s (needs %s)", strings.Join(preset.Skills, " "), strings.Join(names, " and "))
}
