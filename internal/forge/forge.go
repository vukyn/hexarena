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
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
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
	// Species is a comma separated list of what the character is. Empty is a
	// real answer -- "nothing in particular" -- rather than a default taken from
	// somewhere else: an archetype cannot supply it, because how a character
	// fights says nothing about what it is.
	Species string
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

	// Nil rather than an empty slice when nothing was answered, because
	// Character.Species is omitempty: an empty slice writes the same file a nil
	// one does, so keeping it would make what Resolve returns differ from what
	// a reload produces on a field nobody filled in. See
	// TestWrittenCastIsStableAndReloads.
	var species []string
	if strings.TrimSpace(d.Species) != "" {
		species = SplitList(d.Species)
	}
	for _, id := range species {
		if _, known := lib.species.Get(id); !known {
			return cast.Character{}, &UnknownSpeciesError{ID: id}
		}
	}

	// The preset's traits, and there is no answer that overrides them yet — the
	// same arrangement the evolution line has. A trait is one line in cast.json
	// and a preset is where the suggestion belongs, so a wizard question would be
	// a third place to say the same thing before anybody had asked for one.
	passives := archetype.Passives

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
		// The preset's kit becomes a learnset every entry of which is known from
		// level one. An archetype has no level to gate against — it is a
		// suggestion for authoring rather than a placement — so the levels are
		// something the author adds afterwards, in the file, where the rest of
		// the character's shape is edited.
		Skills: cast.Learn(skills), Species: species, Passives: passives,
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

// Carrier is the draft as a kit check wants it: the id, the preset, the element,
// what the character is and where it is from, with an unreadable or unanswered
// element left out rather than guessed at.
//
// Draft.Resolve refuses a bad element on its own account; this is for the
// checks a half-filled form makes as it is typed, where an element that is not
// an element yet is an ordinary state and not a refusal.
func (d Draft) Carrier() Carrier {
	who := Carrier{
		ID: strings.TrimSpace(d.ID), Archetype: strings.TrimSpace(d.Archetype),
		Species: SplitList(d.Species), Origin: strings.TrimSpace(d.Origin),
	}
	if affinity, err := ParseAffinity(d.Element); err == nil {
		who.Affinity, who.HasAffinity = affinity, true
	}
	return who
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
	// Pierced is what the same line absorbs against damage that ignores its
	// defence outright, which comes to exactly its health.
	//
	// It is here because Effective stopped being the whole answer the moment
	// piercing existed: it measures durability against damage that does not
	// pierce, which is what the bound is set against, and a row showing only
	// that figure would be describing the best case as though it were the only
	// one. The two together are the range a stat line really sits in, and the
	// gap between them is how much of its durability an author bought with
	// armour rather than with health.
	Pierced int64
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
		Pierced:   progression.EffectiveHPAgainst(values, l.rules, scale.Base),
	}
}

// Held is the stat line a character actually fights on: its resolved values with
// every permanent status its trait grants already applied.
//
// # Why the budget is not checked against this, and is not meant to be
//
// progression.Limits.CheckValues takes six numbers and nothing else, so the line
// it bounds is the one on paper — which is the line an **author** writes. A
// trait is not in those six: it is named beside the stat line on a placement,
// and its grants are put on at enlistment, after everything that could have
// refused them. So battle.New will reject a base line of 740 defence as over
// budget and then hand the same unit 786 through a trait, in the same call.
//
// That is the intended split rather than a leak. A ceiling and the budget bound
// what may be **authored**; a battle is where a stat is supposed to move, and a
// buff that could not take one past its ceiling would be a buff with a cliff in
// it. What holds the fought line is the saturation: modifier.Set.Stat rescales
// every change against ceiling × headroom, so nothing reaches three times a
// ceiling however much is stacked on it, and nothing reaches the floor beneath
// it either. Whatever a rune turns out to be, it lands in a system that already
// bounds it.
//
// What was missing was not a check but a figure. The fought line is the one a
// battle is decided on, and nothing printed it — so a trait's whole contribution
// was invisible beside the stat line it modifies. This is that figure, and
// hexforge check prints it beside the authored one.
func (l *Library) Held(base progression.Values, traits []string) (progression.Values, error) {
	carried := status.Set{}
	for _, id := range traits {
		held, err := l.passives.Lookup(id)
		if err != nil {
			return base, err
		}
		// A gated trait is off until the gate holds, and the gate reads a
		// health that no character has outside a battle. Counting it here would
		// price blaze as though a Charizard walked in already burning.
		if held.While != nil {
			continue
		}
		for _, grant := range held.Grants {
			kind, err := l.statuses.Lookup(grant.Status)
			if err != nil {
				return base, err
			}
			// Hold is the gate rather than a check here: it refuses a timed
			// status outright, for the reason it exists at all — a trait that
			// granted one would wear off on its holder's turns with nothing to
			// put it back. Repeating the condition here would be a second place
			// for it to be wrong.
			// A nought amount, which is right here and not a shortcut: this
			// function exists to read what a trait does to a STAT LINE, and a
			// guard's pool moves no stat. Handing it the real figure would file a
			// number nothing below this line reads.
			carried.Hold(kind, 0, grant.Stacks)
		}
	}
	return carried.Modifiers().Stats(base, l.limits.Ceilings, l.bounds), nil
}

// Carrier is who a kit is being checked for: the four facts a skill's
// restriction can name.
//
// Each one may be unanswered, and an unanswered fact restricts nothing. That is
// what lets the kit be filled in before the element or after it: whichever the
// author settles first constrains the other, and the second is checked against
// the first as soon as it exists. An empty id and an empty archetype are
// unanswered by being empty; an affinity cannot be, because the zero value is a
// legal neutral affinity, so HasAffinity says whether one was given.
type Carrier struct {
	ID          string
	Archetype   string
	Affinity    element.Affinity
	HasAffinity bool
	// Species is what the character is. An empty list is unanswered here, which
	// is the one place this axis cannot be checked as strictly as the parser
	// checks it: to cast.ParseBook an empty list is a real answer and a lineage
	// skill refuses it, while on a half-filled form it is a question nobody has
	// reached yet. The form takes the second reading, so a lineage skill picked
	// before a species is settled is refused at the write rather than at the
	// keystroke -- and picked after it, at the keystroke.
	Species []string
	// Origin is the work the character was borrowed from, and is unanswered
	// while it is empty for the reason Species is: on a half-filled form it is
	// a question nobody has reached, not an answer of "from nowhere".
	Origin string
}

// CheckSkill reports why a carrier may not carry one skill, or nil when it may.
//
// The element half is skill.WhyCannotCarry's judgement, which cast.ParseBook
// and battle.enlist also call, so this is only bringing the same answer forward:
// a wrong element should cost one line at the moment it is typed rather than the
// whole session at the moment it is saved. The other four are cast.ParseBook's
// alone — the engine has no archetype, no character identity, no species and no
// origin — and this brings those forward too, from the same predicates on
// skill.Restriction.
//
// Every answer comes back as a typed error holding what it is about rather than
// a sentence, so a front-end can word it in the author's language without
// taking the rule apart again.
func CheckSkill(who Carrier, carried skill.Skill) error {
	if who.HasAffinity {
		switch reason := skill.WhyCannotCarry(who.Affinity, carried); reason {
		case skill.CarryWrongElement, skill.CarryElementRestricted:
			return &CarryError{
				Affinity: who.Affinity, Skill: carried.ID, Element: carried.Element,
				Reason: reason, Allowed: carried.Restrict.ElementNames(),
			}
		}
	}
	if who.Archetype != "" && !carried.Restrict.AllowsArchetype(who.Archetype) {
		return &ArchetypeRestrictedError{
			Archetype: who.Archetype, Skill: carried.ID,
			Allowed: append([]string(nil), carried.Restrict.Archetypes...),
		}
	}
	if who.ID != "" && !carried.Restrict.AllowsCharacter(who.ID) {
		return &CharacterRestrictedError{
			Character: who.ID, Skill: carried.ID,
			Allowed: append([]string(nil), carried.Restrict.Characters...),
		}
	}
	if len(who.Species) > 0 && !carried.Restrict.AllowsSpecies(who.Species) {
		return &SpeciesRestrictedError{
			Character: who.ID, Skill: carried.ID,
			Allowed: carried.Restrict.SpeciesNames(),
		}
	}
	if who.Origin != "" && !carried.Restrict.AllowsOrigin(who.Origin) {
		return &OriginRestrictedError{
			Character: who.ID, Skill: carried.ID,
			Allowed: carried.Restrict.OriginNames(),
		}
	}
	return nil
}

// CheckKit reports the first skill in a kit the carrier may not use.
func CheckKit(who Carrier, kit []skill.Skill) error {
	for _, carried := range kit {
		if err := CheckSkill(who, carried); err != nil {
			return err
		}
	}
	return nil
}

// CheckCarry is CheckKit for the element alone, which is what an element answer
// is checked against: the id and the preset are settled elsewhere on the form
// and are refused there.
func CheckCarry(affinity element.Affinity, kit []skill.Skill) error {
	return CheckKit(Carrier{Affinity: affinity, HasAffinity: true}, kit)
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
//
// ⚠️ **A line that forks is not a chain, and drawing it as one is a lie a table
// tells quietly.** Joining every stage with an arrow reads `Eevee@1 → Vaporeon@32
// → Jolteon@32` — three forms one after another, when the last two are
// alternatives at one threshold. Children share a bracket instead:
//
//	Eevee@1 → (Vaporeon@32 | Jolteon@32 → Tempest@48)
//
// A line that does not fork is unchanged, brackets and all: one child is drawn
// as the arrow it always was, so every shipped character reads exactly as before.
// The parentage comes from progression.Line.Parents rather than from the order
// of the file, because that rule lives in one place.
func StageSummary(character cast.Character) string {
	parents, err := character.Stages.Parents()
	if err != nil || len(character.Stages) == 0 {
		// A line the parser would refuse still has to print as something: the
		// flat list is what it says, and the check screen beside it is where the
		// refusal is reported.
		return flatStages(character)
	}
	children := make([][]int, len(character.Stages))
	root := -1
	for i, parent := range parents {
		if parent < 0 {
			root = i
			continue
		}
		children[parent] = append(children[parent], i)
	}
	if root < 0 {
		return flatStages(character)
	}
	return stageBranch(character, children, root)
}

// stageBranch draws one stage and everything growing out of it.
func stageBranch(character cast.Character, children [][]int, at int) string {
	stage := character.Stages[at]
	drawn := fmt.Sprintf("%s@%d", stage.Name, stage.MinLevel)
	grown := children[at]
	switch len(grown) {
	case 0:
		return drawn
	case 1:
		return drawn + " → " + stageBranch(character, children, grown[0])
	}
	arms := make([]string, 0, len(grown))
	for _, child := range grown {
		arms = append(arms, stageBranch(character, children, child))
	}
	return drawn + " → (" + strings.Join(arms, " | ") + ")"
}

// flatStages is the line in file order, which is what there is to say about one
// nothing can make sense of.
func flatStages(character cast.Character) string {
	parts := make([]string, 0, len(character.Stages))
	for _, stage := range StageFacts(character) {
		parts = append(parts, fmt.Sprintf("%s@%d", stage.Name, stage.MinLevel))
	}
	return strings.Join(parts, " → ")
}

// UnlockSummary writes a learnset as the levels its entries come in at, in the
// same shape StageSummary writes an evolution line: `id@level`.
//
// An entry unlocked at level one prints as a bare id. A gate everybody passes is
// not information, and printing `@1` on the common case would put a number on
// every line so that the one line with a real gate stopped standing out.
func UnlockSummary(entries []cast.Unlock) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, unlockLabel(entry.ID, entry, entry.AtLevel > 1))
	}
	return strings.Join(parts, " ")
}

// unlockLabel is one entry as a single token: the id, the level gate while it is
// worth printing, and the forms that may hold it.
//
// One token rather than a phrase, because these are joined by spaces into a row
// an author scans — a bracket with a space in it would read as two entries. The
// forms are printed whenever there are any, at every level, because unlike a
// level gate they never stop being true: a skill kept for the bulb forms is kept
// for them at level 60 as much as at level 1, and a row that dropped the mark
// once the level passed would be saying the grown form has it.
func unlockLabel(id string, entry cast.Unlock, gated bool) string {
	label := id
	if gated {
		label = fmt.Sprintf("%s@%d", id, entry.AtLevel)
	}
	if len(entry.Stages) == 0 {
		return label
	}
	return fmt.Sprintf("%s[%s]", label, strings.Join(entry.Stages, ","))
}

// UnlockSummaryAt is the same list seen from a level: a gate is printed only
// while it is still ahead.
//
// So the row changes as a level is walked — `endurance@16` at level 8 and a bare
// `endurance` at 24 — and the mark reads as "not yet" rather than as a fact about
// the trait. A screen that showed every gate at every level would say the same
// thing at level 1 and at the cap, which is the one thing a level slider is for
// finding out.
func UnlockSummaryAt(entries []cast.Unlock, level int) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, unlockLabel(entry.ID, entry, !entry.Unlocked(level)))
	}
	return strings.Join(parts, " ")
}

// TraitCarrier is one character that learns a trait, and where in its life.
type TraitCarrier struct {
	// Character is the character's id, which is what a listing shows: a name is
	// authored prose and two characters may share one.
	Character string
	// AtLevel is the first level it has the trait at, and Stages the forms that
	// may hold it — the same two gates cast.Unlock declares, carried through so
	// a listing can say "endurance@16" without re-reading the learnset.
	AtLevel int
	Stages  []string
}

// TraitCarriers is every character that learns a trait, in cast order.
//
// A trait has no restriction mechanism — no element, no archetype, no species,
// no character — so "who may carry this" is *everybody* and answers nothing. The
// question worth asking is the other one: who actually **does**. A trait nobody
// learns is not an error, the way a species nobody is is not an error, but it is
// a trait that cannot reach a battle, and a listing is where that shows.
//
// It walks the cast rather than being indexed off the trait, because the
// learnset is the character's fact: a trait is declared in passives.json knowing
// nothing about who takes it, and an index kept the other way round would be a
// second place for the same edge to live.
func (l *Library) TraitCarriers(id string) []TraitCarrier {
	out := make([]TraitCarrier, 0, 4)
	for _, character := range l.characters.All() {
		for _, entry := range character.Passives {
			if entry.ID != id {
				continue
			}
			out = append(out, TraitCarrier{
				Character: character.ID, AtLevel: entry.AtLevel, Stages: entry.Stages,
			})
			break
		}
	}
	return out
}

// TraitCarrierSummary is those carriers as one scannable row, each token the
// character id with whatever gates its entry declares — the same shape
// UnlockSummary gives a learnset, read from the other end.
func TraitCarrierSummary(carriers []TraitCarrier) string {
	parts := make([]string, 0, len(carriers))
	for _, carrier := range carriers {
		parts = append(parts, unlockLabel(carrier.Character,
			cast.Unlock{AtLevel: carrier.AtLevel, Stages: carrier.Stages},
			carrier.AtLevel > 1))
	}
	return strings.Join(parts, " ")
}

// SuggestedImage proposes where a character's art would live, following the id.
// It is only a default: any relative path ending .svg or .png is allowed.
func SuggestedImage(id string) string {
	if id == "" {
		return ""
	}
	folder, name, split := strings.Cut(id, ".")
	if !split {
		return path.Join(assetsDir, folder+".svg")
	}
	return path.Join(assetsDir, folder, name+".svg")
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
