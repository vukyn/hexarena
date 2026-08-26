package forge

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// This file is editing a skill that already ships, which is a different job from
// authoring one, and the difference is worth stating because it is the whole
// reason this is not "add, but overwriting".
//
// Adding a skill cannot break anything: nobody carries it yet. Editing one can,
// in two ways.
//
// It can orphan a carrier. A character already carrying a skill may no longer be
// allowed to once its element changes or its restriction narrows, and so may an
// archetype preset whose kit holds it. So an edit re-validates the presets and
// the cast against the edited book before anything is written, and refuses,
// naming who would break. A save that would make cast.ParseBook fail is the
// thing to prevent — discovering it afterwards means a data directory the game
// no longer boots from.
//
// And it changes balance that already exists. A new skill nobody carries moves
// no golden; a power, a strike count, an accuracy or a cooldown on a skill
// shipped units carry moves scenarios.golden and the tables beside it. So a
// successful edit reports the damage before and against the damage after, from
// the same PreviewDamage reference skills.golden's own column is measured from.

// SkillEdit is a partial change to a skill: a field it does not name is left
// exactly as it is.
//
// A pointer per field rather than a string, and that is the trap this type
// exists to close: an answer nobody gave and an answer of zero are different
// things, and a zero value cannot tell them apart. `--cooldown 0` means make the
// cooldown zero; no `--cooldown` at all means leave whatever is there. A
// front-end comparing against "" would silently turn the first into the second,
// and the author would be left with a cooldown they had explicitly cleared and
// which had not moved.
//
// An explicitly empty string is therefore a real answer, and on the three
// allowlists it is the only way to say "clear this": SplitList reads it as no
// entries, and Restriction turns three empty lists into no restrict block at
// all, which is a skill back in the common pool.
type SkillEdit struct {
	// ID is only ever here to be refused. It is carried so that a front-end can
	// pass on what was asked for and get this package's refusal rather than
	// wording one of its own. See SkillDraft.ResolveEdit.
	ID   *string
	Name *string
	// Element and the rest are the fields an edit may name.
	Element  *string
	Target   *string
	Range    *string
	Pattern  *string
	Power    *string
	Strikes  *string
	Accuracy *string
	Cooldown *string
	// SelfApplies, Pierce, Restores and Drains, like every other field here, are nil
	// when the flag was not given and point at an answer when it was — so an
	// empty string clears a list or zeroes a figure, and an absent flag leaves
	// it alone.
	SelfApplies *string
	Pierce      *string
	Restores    *string
	Drains      *string
	Applies     *string

	RestrictElements   *string
	RestrictArchetypes *string
	RestrictCharacters *string
	RestrictSpecies    *string
}

// Draft lays a partial change over a skill as it stands, producing the whole set
// of answers the skill would be authored from.
//
// A partial edit and a filled-in form therefore arrive at the same place — one
// SkillDraft — which is what keeps a flag-driven edit and a form-driven one from
// applying different rules. The answers not named come from SkillAnswers, so
// they are the values the file already holds.
func (e SkillEdit) Draft(current skill.Skill) SkillDraft {
	drafted := SkillAnswers(current)
	for _, field := range []struct {
		into  *string
		given *string
	}{
		{&drafted.ID, e.ID},
		{&drafted.Name, e.Name},
		{&drafted.Element, e.Element},
		{&drafted.Target, e.Target},
		{&drafted.Range, e.Range},
		{&drafted.Pattern, e.Pattern},
		{&drafted.Power, e.Power},
		{&drafted.Strikes, e.Strikes},
		{&drafted.Accuracy, e.Accuracy},
		{&drafted.Cooldown, e.Cooldown},
		{&drafted.Applies, e.Applies},
		{&drafted.SelfApplies, e.SelfApplies},
		{&drafted.Pierce, e.Pierce},
		{&drafted.Restores, e.Restores},
		{&drafted.Drains, e.Drains},
		{&drafted.RestrictElements, e.RestrictElements},
		{&drafted.RestrictArchetypes, e.RestrictArchetypes},
		{&drafted.RestrictCharacters, e.RestrictCharacters},
		{&drafted.RestrictSpecies, e.RestrictSpecies},
	} {
		if field.given != nil {
			*field.into = *field.given
		}
	}
	return drafted
}

// Names reports whether an edit changes anything at all, which is what a
// front-end refusing an empty invocation needs.
func (e SkillEdit) Names() bool {
	empty := SkillEdit{}
	return e != empty
}

// SkillChange is a written edit: the skill before and after, and what each is
// worth.
//
// The two damage figures are the point of returning a value at all rather than
// just an error. A skill is balance, so what an author needs after a write is
// not that it succeeded but what moved — and both figures come from the one
// PreviewDamage reference, so the before is comparable with the after and both
// are comparable with skills.golden's own column.
type SkillChange struct {
	Before, After             skill.Skill
	BeforeDamage, AfterDamage SkillPreview
}

// MovesDamage reports whether the edit changed what the skill is worth.
//
// A front-end uses it to leave the before-and-after line out when there is
// nothing to compare: an edit to a restriction or a targeting side moves no
// damage, and a line saying a number did not change is a line that has to be
// read to learn nothing.
func (c SkillChange) MovesDamage() bool {
	return c.BeforeDamage.PerStrike != c.AfterDamage.PerStrike ||
		c.BeforeDamage.Total != c.AfterDamage.Total
}

// DamageSummary is MovesDamage's figures as the English cmd/hexforge prints.
func (c SkillChange) DamageSummary() string {
	return strconv.FormatInt(c.BeforeDamage.PerStrike, 10) + " → " +
		strconv.FormatInt(c.AfterDamage.PerStrike, 10) + " per strike, " +
		strconv.FormatInt(c.BeforeDamage.Total, 10) + " → " +
		strconv.FormatInt(c.AfterDamage.Total, 10) + " in total"
}

// EditSkill writes a changed skill, having first made sure nothing already in
// the data directory stops carrying it.
//
// The order is the whole design. Replace validates the skill itself exactly as a
// load would. Then the presets and the cast are re-parsed against the edited
// book, because those are the two files that name a skill, and a refusal from
// either is turned into a refusal naming who would break. Only then is anything
// written, through the same temp-file-then-rename SaveSkill uses.
//
// On success the library holds all three re-parsed books, not just the skills.
// An archetype's Demands is derived from its kit's elements, so an edit to a
// skill's element changes a preset without touching archetypes.json, and a
// library still holding the old presets would answer the next question wrongly.
func (l *Library) EditSkill(edited skill.Skill) (SkillChange, error) {
	before, err := l.skills.Lookup(edited.ID)
	if err != nil {
		return SkillChange{}, &UnknownSkillError{ID: edited.ID, Err: err}
	}
	changed, err := l.skills.Replace(l.SkillDeps(), edited)
	if err != nil {
		return SkillChange{}, err
	}
	archetypes, characters, err := l.recheckCarriers(changed, edited.ID)
	if err != nil {
		return SkillChange{}, err
	}
	raw, err := changed.Marshal()
	if err != nil {
		return SkillChange{}, err
	}
	if err := l.replaceFile(skillsFile, raw); err != nil {
		return SkillChange{}, err
	}
	l.skills, l.archetypes, l.characters = changed, archetypes, characters
	after, err := changed.Lookup(edited.ID)
	if err != nil {
		return SkillChange{}, err
	}
	return SkillChange{
		Before: before, After: after,
		BeforeDamage: l.PreviewDamage(before), AfterDamage: l.PreviewDamage(after),
	}, nil
}

// recheckCarriers re-parses the two books that name a skill against an edited
// one, and hands both back so a caller can hold what it just proved loads.
//
// The bytes come off the disk rather than out of the books in hand, because what
// has to keep loading is the directory, not this process's memory: a preset file
// edited by hand since the library was loaded is the file the next load will
// read.
//
// The presets go first because the cast cannot be parsed without them — a
// character names the archetype it was tuned from — and because a preset refused
// for holding an edited skill is a refusal about the skill either way.
func (l *Library) recheckCarriers(edited *skill.Book, id string) (*cast.ArchetypeBook, *cast.Book, error) {
	rawArchetypes, err := os.ReadFile(filepath.Join(l.dir, archetypesFile))
	if err != nil {
		return nil, nil, err
	}
	// The library's own dependency lists with the skill book swapped, rather than
	// two fresh ones written out here. A re-parse missing a book refuses on the
	// missing book instead of on the edit, and the refusal it produces names a
	// preset the author never touched — which is exactly what happened when
	// passives arrived and this restated its deps: every skill edit in the
	// repository began failing with "archetype blighter names passives, which
	// cannot be checked without the passive book".
	archetypeDeps := l.ArchetypeDeps()
	archetypeDeps.Skills = edited
	archetypes, err := cast.ParseArchetypes(rawArchetypes, archetypeDeps)
	if err != nil {
		return nil, nil, l.brokenPreset(edited, id, err)
	}
	rawCast, err := os.ReadFile(filepath.Join(l.dir, castFile))
	if err != nil {
		return nil, nil, err
	}
	castDeps := l.CastDeps()
	castDeps.Archetypes = archetypes
	castDeps.Skills = edited
	characters, err := cast.ParseBook(rawCast, castDeps)
	if err != nil {
		return nil, nil, l.brokenCharacter(edited, id, err)
	}
	return archetypes, characters, nil
}

// brokenPreset and brokenCharacter say which carrier a refused re-parse was
// about.
//
// Neither decides anything. The parser has already said no by the time either
// runs, and nothing here can turn that into a yes — this is the same
// classification checkAffinity does after the element chart has refused, and for
// the same reason: a front-end needs the carrier's id to name it in the author's
// language, and pattern-matching the parser's English for it would be a second
// declaration of the rule behind the refusal.
//
// A refusal neither walk recognises keeps the parser's own words and names
// nobody, rather than being pinned on a carrier that is not at fault. A kit
// demanding a third element is the case that lands here today: it is a rule
// about the kit as a whole rather than about one skill in it, and cast is the
// only place that holds it.
func (l *Library) brokenPreset(edited *skill.Book, id string, refused error) error {
	broken := &SkillEditBreaksError{Carrier: BrokenPreset, Skill: id, Err: refused}
	for _, preset := range l.archetypes.All() {
		kit, err := kitFrom(edited, preset.Skills)
		if err != nil {
			continue
		}
		if reason := CheckPresetKit(preset.ID, kit); reason != nil {
			broken.ID, broken.Err = preset.ID, reason
			return broken
		}
	}
	return broken
}

func (l *Library) brokenCharacter(edited *skill.Book, id string, refused error) error {
	broken := &SkillEditBreaksError{Carrier: BrokenCharacter, Skill: id, Err: refused}
	for _, character := range l.characters.All() {
		kit, err := kitFrom(edited, character.Skills)
		if err != nil {
			continue
		}
		who := Carrier{
			ID: character.ID, Archetype: character.Archetype,
			Affinity: character.Element, HasAffinity: true,
		}
		if reason := CheckKit(who, kit); reason != nil {
			broken.ID, broken.Err = character.ID, reason
			return broken
		}
	}
	return broken
}

// kitFrom resolves a kit against a particular book, which is what checking a
// carrier against an *edited* one needs: Library.LookupKit reads the book the
// library currently holds, and that is the one the edit has not happened in yet.
func kitFrom(book *skill.Book, named []string) ([]skill.Skill, error) {
	out := make([]skill.Skill, 0, len(named))
	for _, id := range named {
		known, err := book.Lookup(id)
		if err != nil {
			return nil, &UnknownSkillError{ID: id, Err: err}
		}
		out = append(out, known)
	}
	return out, nil
}

// CheckPresetKit reports why an archetype preset's kit could not hold one of its
// skills, or nil when it may hold all of them.
//
// It is cast.resolveArchetype's two restriction rules brought forward, in the
// order that function applies them, exactly as CheckSkill brings forward
// resolveCharacter's three. The predicates are skill.Restriction's own, so this
// asks the same questions rather than asking equivalent ones.
//
// The first rule is the one worth remembering: a skill kept for named characters
// cannot sit in a preset's kit at all, because a preset is the starting point for
// every character built from it, so the refusal would land on the author of the
// next character rather than on whoever narrowed the skill.
func CheckPresetKit(archetype string, kit []skill.Skill) error {
	for _, carried := range kit {
		if carried.Restrict.NamesCharacters() {
			return &PresetOwnedSkillError{
				Archetype: archetype, Skill: carried.ID,
				Allowed: append([]string(nil), carried.Restrict.Characters...),
			}
		}
		if err := CheckSkill(Carrier{Archetype: archetype}, carried); err != nil {
			return err
		}
	}
	return nil
}

// EditSkillNoteFacts is what a written edit is worth saying about, in order.
//
// The goldens note is here for the reason it is on a save — a skill is balance —
// and it is unconditional for a reason worth stating: the damage figures are not
// the only thing that reaches a golden. An element decides every matchup in
// scenarios.golden, and an accuracy or a cooldown moves a battle without moving
// a single damage number, so "the damage did not move" is not "nothing moved".
func (l *Library) EditSkillNoteFacts(change SkillChange) []Note {
	return []Note{
		{Kind: NoteEdited, ID: change.After.ID, Path: l.SkillsPath()},
		{Kind: NoteGoldensMove},
		{Kind: NoteRebuild},
	}
}

// EditSkillNotes is EditSkillNoteFacts in the English cmd/hexforge prints.
func (l *Library) EditSkillNotes(change SkillChange) []string {
	return l.noteLines(l.EditSkillNoteFacts(change))
}
