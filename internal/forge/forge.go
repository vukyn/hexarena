// Package forge is the authoring logic behind the cast: loading the books an
// author edits, turning a set of answers into a character, and reporting what
// is wrong with a data directory.
//
// It holds no rules of its own. Every check it makes it makes by calling into
// internal/core/cast, internal/core/element, internal/core/skill or
// internal/core/progression, because a check that lived here would be a check
// the game's own load did not make. What it adds is the *sequence*: which
// question is asked before which, what a preset supplies, and what the write
// looks like on disk.
//
// # Why this may read and write the filesystem
//
// internal/core may not touch the filesystem at all — its parsers take []byte
// so that loading the game cannot depend on a working directory — and
// internal/seed only ever reads the copy go:embed baked in, which has no
// directory to be relative to. Authoring is the one job that needs real files:
// it writes cast.json and origins.json back out, and it is the only place
// allowed to ask whether the art a character names is really there.
//
// (This comment spells that directive "go:embed" inside a sentence on purpose.
// A comment line beginning with its real spelling is read by the compiler as a
// directive, and in this repository a stray one would be a real trap.)
//
// # Why it is a package rather than part of cmd/hexforge
//
// Two front-ends author the same cast: cmd/hexforge, which is flags and
// prompts and therefore what a script and a pipe use, and cmd/hexforge-tui,
// which is a full-screen terminal program. Anything either of them restated
// would be a second copy of a rule, and this repository has already been bitten
// twice by exactly that — see "One source for a recorded string" and the
// kit-versus-affinity gap in CLAUDE.md. So the rule, the wording of its
// rejection, and the numbers behind a budget all live here, and a front-end
// only decides where on the screen to put them.
package forge

import (
	"fmt"
	"path"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// Draft is every answer that makes a character, held as the strings a flag, a
// prompt or a text input produced.
//
// Keeping the answers as text until the very end is what lets a flag-only
// invocation, a fully prompted one and a form driven by keystrokes turn into
// the same character. That happens in Resolve, and Resolve is pure: it reads
// the library, writes nothing, and touches neither the terminal nor the
// filesystem, which is what makes it testable without capturing a screen.
type Draft struct {
	ID        string
	Name      string
	Origin    string
	Archetype string
	Image     string
	Element   string
	Bio       string
	// Skills is a comma separated list. Empty means "take the archetype's kit".
	Skills string
	// Stats holds a "base:max" override per stat. An empty entry means "take
	// the archetype's curve".
	Stats [progression.KindCount]string
}

// Resolve turns a draft into a character, or says which answer is wrong.
//
// The last word belongs to the parser: the candidate is appended to a copy of
// the book, which validates it exactly as loading the file would, so nothing
// can be produced here that would not load back.
func (d Draft) Resolve(lib *Library) (cast.Character, error) {
	if err := cast.ValidateID(d.ID); err != nil {
		return cast.Character{}, &FieldRefusedError{Field: FieldID, Value: d.ID, Err: err}
	}
	if _, clash := lib.characters.Get(d.ID); clash {
		return cast.Character{}, &IDTakenError{ID: d.ID}
	}
	if strings.TrimSpace(d.Name) == "" {
		return cast.Character{}, &MissingNameError{ID: d.ID}
	}
	if _, known := lib.origins.Get(d.Origin); !known {
		return cast.Character{}, &UnknownOriginError{ID: d.Origin}
	}
	archetype, known := lib.archetypes.Get(d.Archetype)
	if !known {
		return cast.Character{}, &UnknownArchetypeError{ID: d.Archetype, Known: lib.archetypes.IDs()}
	}
	if err := cast.ValidateImagePath(d.Image); err != nil {
		return cast.Character{}, &FieldRefusedError{Field: FieldImage, Value: d.Image, Err: err}
	}
	affinity, err := ParseAffinity(d.Element)
	if err != nil {
		return cast.Character{}, err
	}
	if err := lib.checkAffinity(affinity); err != nil {
		return cast.Character{}, err
	}

	table := archetype.Stats
	for _, kind := range progression.Kinds() {
		override := strings.TrimSpace(d.Stats[kind])
		if override == "" {
			continue
		}
		curve, err := ParseCurve(override)
		if err != nil {
			return cast.Character{}, &StatFieldError{Kind: kind, Err: err}
		}
		if err := checkCurve(kind, curve); err != nil {
			return cast.Character{}, err
		}
		table[kind] = curve
	}

	skills := archetype.Skills
	if strings.TrimSpace(d.Skills) != "" {
		skills = SplitList(d.Skills)
	}

	// A character created here has one stage, named after the character. A
	// second stage is an evolution line, and authoring one is an edit to
	// cast.json rather than a wizard question: the whole point of a stage is
	// the curve behind it, and answering twelve numbers twice at a prompt is
	// worse than editing the file.
	character := cast.Character{
		ID: d.ID, Name: strings.TrimSpace(d.Name),
		Origin: d.Origin, Archetype: d.Archetype,
		Image: d.Image, Element: affinity, Bio: strings.TrimSpace(d.Bio),
		Stages: progression.Line{{Name: strings.TrimSpace(d.Name), MinLevel: 1, Stats: table}},
		Skills: skills,
	}
	if _, err := lib.characters.Append(lib.CastDeps(), character); err != nil {
		return cast.Character{}, err
	}
	return character, nil
}

// Table is the stat curve a draft would produce, taking the archetype's for
// every stat the draft does not override.
//
// It exists so a front-end can show the budget a half-finished draft is
// spending without resolving a character it does not yet have a name for.
// Resolve builds the same table; both call this so an incomplete form and a
// finished write cannot disagree about a number.
func (d Draft) Table(lib *Library) (progression.Table, error) {
	var table progression.Table
	if archetype, known := lib.archetypes.Get(d.Archetype); known {
		table = archetype.Stats
	}
	for _, kind := range progression.Kinds() {
		override := strings.TrimSpace(d.Stats[kind])
		if override == "" {
			continue
		}
		curve, err := ParseCurve(override)
		if err != nil {
			return table, &StatFieldError{Kind: kind, Err: err}
		}
		if err := checkCurve(kind, curve); err != nil {
			return table, err
		}
		table[kind] = curve
	}
	return table, nil
}

// KitNames is the kit a draft would be written with: the named skills, or the
// archetype's when none were named.
func (d Draft) KitNames(lib *Library) []string {
	if strings.TrimSpace(d.Skills) != "" {
		return SplitList(d.Skills)
	}
	if archetype, known := lib.archetypes.Get(d.Archetype); known {
		return archetype.Skills
	}
	return nil
}

// Budget is what a stat line spends of the joint health-and-defence bound.
//
// The two multiply rather than add, so this is the limit an author walks into
// without noticing, and it is worth a number on screen rather than a rejection
// at the end. Headroom is negative exactly when the line is over budget.
type Budget struct {
	Effective int64
	Max       int64
	Headroom  int64
}

// Over reports whether the line breaks the bound.
func (b Budget) Over() bool { return b.Headroom < 0 }

// Budget measures a resolved stat line against the shipped limit.
func (l *Library) Budget(values progression.Values) Budget {
	effective := progression.EffectiveHP(values, l.rules)
	return Budget{
		Effective: effective,
		Max:       l.limits.MaxEffectiveHP,
		Headroom:  l.limits.MaxEffectiveHP - effective,
	}
}

// CheckCarry reports the first skill in a kit that an affinity may not use.
//
// skill.CanCarry is the single declaration of the rule and cast.ParseBook
// applies it at the write, so this is only bringing the same answer forward:
// a wrong element should cost one line at the moment it is typed rather than
// the whole session at the moment it is saved. The answer comes back as a
// *CarryError holding the affinity, the skill and the skill's element, so a
// front-end can word it in the author's language without taking the rule apart
// again.
func CheckCarry(affinity element.Affinity, kit []skill.Skill) error {
	for _, carried := range kit {
		if !skill.CanCarry(affinity, carried) {
			return &CarryError{Affinity: affinity, Skill: carried.ID, Element: carried.Element}
		}
	}
	return nil
}

// KitDemands is the distinct non-neutral elements a kit insists on.
//
// The demand is derived from the kit actually chosen, not from the preset,
// because the two differ the moment the kit is edited. skill.Demands is the
// derivation; this is where a front-end reaches it, so that "what does this kit
// need" has one answer for the prompt, the form and the check.
func KitDemands(kit []skill.Skill) []element.Element { return skill.Demands(kit) }

// DemandSummary is KitDemands as the English sentence cmd/hexforge prints.
func DemandSummary(kit []skill.Skill) string {
	demanded := KitDemands(kit)
	if len(demanded) == 0 {
		return "this kit is all neutral, so any element carries it"
	}
	names := make([]string, 0, len(demanded))
	for _, member := range demanded {
		names = append(names, member.String())
	}
	return "this kit needs " + strings.Join(names, " and ")
}

// Preset is a role preset as a chooser wants it: the kit it supplies and the
// elements that kit will insist on, both as the ids they are written with.
type Preset struct {
	Skills  []string
	Demands []string
}

// PresetFacts reads a preset without wording anything about it.
func PresetFacts(preset cast.Archetype) Preset {
	return Preset{Skills: preset.Skills, Demands: preset.DemandNames()}
}

// PresetSummary is PresetFacts as the English line cmd/hexforge prints.
func PresetSummary(preset cast.Archetype) string {
	facts := PresetFacts(preset)
	if len(facts.Demands) == 0 {
		return fmt.Sprintf("%s (any element)", strings.Join(facts.Skills, " "))
	}
	return fmt.Sprintf("%s (needs %s)",
		strings.Join(facts.Skills, " "), strings.Join(facts.Demands, " and "))
}

// Stage is one step of an evolution line: what it is called and the level it
// takes over at.
type Stage struct {
	Name     string
	MinLevel int
}

// StageFacts is an evolution line as the pairs behind it.
func StageFacts(character cast.Character) []Stage {
	out := make([]Stage, 0, len(character.Stages))
	for _, stage := range character.Stages {
		out = append(out, Stage{Name: stage.Name, MinLevel: stage.MinLevel})
	}
	return out
}

// StageSummary writes an evolution line as the levels its stages take over at,
// which is the one thing a table cell has room for.
func StageSummary(character cast.Character) string {
	stages := StageFacts(character)
	parts := make([]string, 0, len(stages))
	for _, stage := range stages {
		parts = append(parts, fmt.Sprintf("%s@%d", stage.Name, stage.MinLevel))
	}
	return strings.Join(parts, " → ")
}

// SuggestedImage proposes where a character's art would live, following the id.
// It is only a default: any relative path ending .svg or .png is allowed.
func SuggestedImage(id string) string {
	if id == "" {
		return ""
	}
	folder, name, split := strings.Cut(id, ".")
	if !split {
		return path.Join("assets", folder+".svg")
	}
	return path.Join("assets", folder, name+".svg")
}

// ShortStat is the three letter label for a stat: a column heading in a table,
// a flag name on the command line, and a row label in a form.
//
// These six labels are not translated, in any front-end, and that is a
// decision rather than an omission. They are the flag names cmd/hexforge takes
// (--hp, --atk) and the keys the data files are written with, so an author
// types them either way; translating the form's row label would leave a person
// reading "phòng thủ" on screen and needing "def" to act on it. They are also
// what the fixed-width columns were measured for — every one is three
// characters or fewer, which no translation of "defence" is.
func ShortStat(kind progression.Kind) string {
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
