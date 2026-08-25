package forge

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// The refusals this package makes, as types rather than as sentences.
//
// # Why a type and not an fmt.Errorf
//
// Two front-ends now show the same refusal in different languages:
// cmd/hexforge is English because it is what a script reads, and
// cmd/hexforge-tui speaks whichever language the author picked. A sentence
// cannot be translated after the fact without parsing it back apart, so the
// *fact* — which id was taken, which skill an affinity cannot carry — is what
// travels, and the wording is chosen where it is drawn.
//
// Every Error method below writes exactly the English cmd/hexforge printed
// before these types existed, so the command line's output is unchanged and
// cannot drift from the facts: there is only one place left that knows the id
// was taken. internal/i18n turns the same values into Vietnamese.
//
// The ones that wrap an error from internal/core delegate their Error to it
// rather than restating it. A core message belongs to core; repeating it here
// would be a second declaration of a rule this package is not allowed to hold.

// IDTakenError is an id that is already in the cast.
type IDTakenError struct{ ID string }

func (e *IDTakenError) Error() string {
	return fmt.Sprintf("character %q is already in the cast", e.ID)
}

// MissingNameError is a character with nothing to be called. The id is empty
// when the check was made before an id was settled, which is the difference
// between the two wordings the command line has always used.
type MissingNameError struct{ ID string }

func (e *MissingNameError) Error() string {
	if e.ID == "" {
		return "a character needs a display name"
	}
	return fmt.Sprintf("character %q needs a display name", e.ID)
}

// UnknownOriginError is a work that is not in the catalog. It carries the
// command that would add it, because a refusal an author cannot act on is only
// half a refusal.
type UnknownOriginError struct{ ID string }

// AddCommand is what a person would type to catalog the missing work.
func (e *UnknownOriginError) AddCommand() string { return "hexforge origins add " + e.ID }

func (e *UnknownOriginError) Error() string {
	return fmt.Sprintf("unknown origin %q, add it with %q", e.ID, e.AddCommand())
}

// UnknownArchetypeError is a preset that does not exist, with the ones that do.
type UnknownArchetypeError struct {
	ID    string
	Known []string
}

func (e *UnknownArchetypeError) Error() string {
	return fmt.Sprintf("unknown archetype %q, want one of %s", e.ID, strings.Join(e.Known, ", "))
}

// OriginTakenError is a work whose id is already catalogued.
type OriginTakenError struct{ ID string }

func (e *OriginTakenError) Error() string {
	return fmt.Sprintf("origin %q is already in the catalog", e.ID)
}

// UnknownCharacterError is a name that is not in the cast. It is raised by an
// allowlist naming somebody who does not exist, which is the same mistake as an
// empty allowlist: nobody satisfies it.
type UnknownCharacterError struct{ ID string }

func (e *UnknownCharacterError) Error() string {
	return fmt.Sprintf("no character %q in the cast", e.ID)
}

// DuplicateEntryError is one name listed twice in an allowlist.
type DuplicateEntryError struct{ Value string }

func (e *DuplicateEntryError) Error() string {
	return fmt.Sprintf("%q is named twice", e.Value)
}

// SkillTakenError is an id that is already in the skill book.
type SkillTakenError struct{ ID string }

func (e *SkillTakenError) Error() string {
	return fmt.Sprintf("skill %q is already in the book", e.ID)
}

// MissingSkillIDError is a skill with nothing to be called.
type MissingSkillIDError struct{}

func (e *MissingSkillIDError) Error() string { return "a skill needs an id" }

// SkillRenameError is an edit that would change a skill's id.
//
// It is a refusal rather than a rename because a rename is a different
// operation: the id is named by every kit that carries the skill, by every
// preset's kit, and by every restrict.characters list that keeps it for
// somebody, so moving the declaration alone would leave a book that does not
// load. The refusal says so rather than saying no, because an author who typed a
// new id wants to know what would have to happen, not only that it did not.
type SkillRenameError struct{ From, To string }

func (e *SkillRenameError) Error() string {
	return fmt.Sprintf("a skill's id cannot be edited, so %q cannot become %q; "+
		"renaming one has to change every kit and every restriction that names it, "+
		"which is a separate operation", e.From, e.To)
}

// PresetOwnedSkillError is an archetype preset whose kit holds a skill kept for
// named characters.
//
// The judgement is cast.resolveArchetype's, brought forward by CheckPresetKit.
// It is a refusal the engine can never make, for the same reason
// ArchetypeRestrictedError is: a roster entry carries no archetype.
type PresetOwnedSkillError struct {
	Archetype string
	Skill     string
	Allowed   []string
}

// The sentence names the skill and its owners and leaves the preset out, even
// though the preset is carried in a field. Whoever shows this has the preset's
// id already — it is what they asked about — and a refusal that named it here as
// well would say it twice in one line.
func (e *PresetOwnedSkillError) Error() string {
	return fmt.Sprintf("%q belongs to %s, and a preset is shared by every character built from it",
		e.Skill, strings.Join(e.Allowed, " or "))
}

// BrokenCarrier is which kind of carrier an edit would break.
//
// The two are not interchangeable in a refusal: a character is one unit an
// author can retune, and a preset is the starting point for every character
// built from it, so the same edit is a smaller problem in the first case than in
// the second.
type BrokenCarrier int

const (
	// BrokenCharacter is an authored character that would stop being able to
	// carry the skill.
	BrokenCharacter BrokenCarrier = iota
	// BrokenPreset is an archetype preset whose kit would stop being legal.
	BrokenPreset
)

// SkillEditBreaksError is an edit that would leave something already authored
// unable to carry the skill.
//
// Err is the reason, and it is either one of this package's own typed refusals —
// the classified case, where a carrier walk found who breaks and why — or the
// parser's own words, for a refusal about a kit as a whole rather than about one
// skill in it. ID is empty in that second case, and a front-end then says what
// the parser said rather than blaming a carrier that may not be at fault.
type SkillEditBreaksError struct {
	Carrier BrokenCarrier
	// ID is the character or preset that would break, empty when the refusal
	// belongs to no single one.
	ID    string
	Skill string
	Err   error
}

func (e *SkillEditBreaksError) Error() string {
	if e.ID == "" {
		return fmt.Sprintf("editing %q would stop the books loading: %v", e.Skill, e.Err)
	}
	if e.Carrier == BrokenPreset {
		return fmt.Sprintf("editing %q would leave the %s preset unable to carry it: %v",
			e.Skill, e.ID, e.Err)
	}
	return fmt.Sprintf("editing %q would leave %s unable to carry it: %v",
		e.Skill, e.ID, e.Err)
}

func (e *SkillEditBreaksError) Unwrap() error { return e.Err }

// UnknownPatternError is a shape the pattern book does not hold. The judgement
// is pattern.Book.Lookup's; this only carries the name alongside it.
type UnknownPatternError struct {
	Name string
	Err  error
}

func (e *UnknownPatternError) Error() string { return e.Err.Error() }
func (e *UnknownPatternError) Unwrap() error { return e.Err }

// UnknownTargetError is a targeting side that is not one, on the same terms.
type UnknownTargetError struct {
	Name string
	Err  error
}

func (e *UnknownTargetError) Error() string { return e.Err.Error() }
func (e *UnknownTargetError) Unwrap() error { return e.Err }

// UnknownStatusError is a status the book does not hold, on the same terms.
type UnknownStatusError struct {
	ID  string
	Err error
}

func (e *UnknownStatusError) Error() string { return e.Err.Error() }
func (e *UnknownStatusError) Unwrap() error { return e.Err }

// NumberError is an answer that was meant to be a number and is not.
//
// It carries no bounds, deliberately: what a legal power or a legal cooldown is
// belongs to skill.ParseBook, and a second opinion here would be a second
// declaration of it. This is only the difference between "that is not a number"
// and "that number will not do".
type NumberError struct {
	Raw string
	Err error
}

func (e *NumberError) Error() string {
	return fmt.Sprintf("%q is not a number", e.Raw)
}

func (e *NumberError) Unwrap() error { return e.Err }

// ApplicationShapeError is an inflicted-status entry that is not a status and a
// chance.
type ApplicationShapeError struct{ Raw string }

func (e *ApplicationShapeError) Error() string {
	return fmt.Sprintf("%q is not a status and a chance, want status:chance or status:chance:stacks", e.Raw)
}

// EmptyKitError is a character that would have nothing to do on its turn.
type EmptyKitError struct{}

func (e *EmptyKitError) Error() string {
	return "a character with no skills would have nothing to do on its turn"
}

// DuplicateSkillError is one skill named twice in a kit.
type DuplicateSkillError struct{ ID string }

func (e *DuplicateSkillError) Error() string { return fmt.Sprintf("%q is named twice", e.ID) }

// UnknownSkillError is a kit naming a skill the book does not hold. The
// judgement is skill.Book.Lookup's; this only carries the id alongside it.
type UnknownSkillError struct {
	ID  string
	Err error
}

func (e *UnknownSkillError) Error() string { return e.Err.Error() }
func (e *UnknownSkillError) Unwrap() error { return e.Err }

// UnknownElementError is a name that is not an element. element.Parse decides;
// this carries what was typed so a form can say so in its own language.
type UnknownElementError struct {
	Name string
	Err  error
}

func (e *UnknownElementError) Error() string { return e.Err.Error() }
func (e *UnknownElementError) Unwrap() error { return e.Err }

// MissingElementError is an empty answer where an affinity was wanted.
type MissingElementError struct{}

func (e *MissingElementError) Error() string { return "no element given" }

// AffinityCountError is an answer listing more elements than an affinity holds.
type AffinityCountError struct {
	Raw   string
	Count int
}

func (e *AffinityCountError) Error() string {
	return fmt.Sprintf("%q lists %d elements, want one or two separated by a slash", e.Raw, e.Count)
}

// AffinityReason is why the chart refused a pairing, classified only so a
// front-end can pick a sentence for it.
//
// The refusal itself is element.Chart.ValidateAffinity's, and this is never
// consulted to decide one: the chart has already said no by the time a reason
// is filled in. An outcome the chart grows later that this does not recognise
// arrives as AffinityReasonUnclassified, and a front-end falls back to the
// chart's own words rather than inventing a wrong explanation.
type AffinityReason int

const (
	AffinityReasonUnclassified AffinityReason = iota
	// AffinityReasonUndeclared is an affinity holding an element the chart has
	// never heard of.
	AffinityReasonUndeclared
	// AffinityReasonCounters is a pairing of two elements that already counter
	// each other.
	AffinityReasonCounters
)

// AffinityRefusedError is an affinity the element chart will not allow.
type AffinityRefusedError struct {
	Affinity element.Affinity
	Reason   AffinityReason
	Err      error
}

func (e *AffinityRefusedError) Error() string { return e.Err.Error() }
func (e *AffinityRefusedError) Unwrap() error { return e.Err }

// CarryError is a kit holding a skill the affinity may not use, with the four
// things that explain it: what the character is, which skill, what that skill
// is, and which of the two element rules refused it.
//
// Reason is skill.WhyCannotCarry's own answer, carried rather than recomputed.
// The two need different advice — a wrong element is fixed by taking the
// skill's element, and a restricted one cannot be, because the skill's element
// is already shared — so a front-end that had only the affinity and the skill
// would have to work out which had happened, which is the rule declared twice.
type CarryError struct {
	Affinity element.Affinity
	Skill    string
	Element  element.Element
	Reason   skill.CarryRefusal
	// Allowed is the skill's element allowlist, on CarryElementRestricted.
	Allowed []string
}

func (e *CarryError) Error() string {
	if e.Reason == skill.CarryElementRestricted {
		return fmt.Sprintf("%s cannot carry %q, which only %s may carry",
			e.Affinity, e.Skill, strings.Join(e.Allowed, " or "))
	}
	return fmt.Sprintf("%s cannot carry %q, which is %s", e.Affinity, e.Skill, e.Element)
}

// ArchetypeRestrictedError is a kit holding a skill the preset it was tuned
// from may not use.
//
// It is a refusal the engine can never make: a roster entry carries no
// archetype, so this rule lives entirely where a character is authored.
type ArchetypeRestrictedError struct {
	Archetype string
	Skill     string
	Allowed   []string
}

func (e *ArchetypeRestrictedError) Error() string {
	return fmt.Sprintf("%q cannot carry %q, which only the %s archetype may carry",
		e.Archetype, e.Skill, strings.Join(e.Allowed, " or "))
}

// CharacterRestrictedError is a kit holding a skill that belongs to somebody
// else, on the same terms as ArchetypeRestrictedError.
type CharacterRestrictedError struct {
	Character string
	Skill     string
	Allowed   []string
}

func (e *CharacterRestrictedError) Error() string {
	return fmt.Sprintf("%q cannot carry %q, which only %s may carry",
		e.Character, e.Skill, strings.Join(e.Allowed, " or "))
}

// CurveShapeError is an answer that is not a "base:max" pair at all.
type CurveShapeError struct{ Raw string }

func (e *CurveShapeError) Error() string {
	return fmt.Sprintf("%q is not a curve, want base:max", e.Raw)
}

// CurveHalf names which side of a curve would not read as a number.
type CurveHalf int

const (
	CurveBase CurveHalf = iota
	CurveMax
)

func (h CurveHalf) String() string {
	if h == CurveMax {
		return "max"
	}
	return "base"
}

// CurveNumberError is a "base:max" pair with an unreadable half.
type CurveNumberError struct {
	Raw  string
	Half CurveHalf
	Err  error
}

func (e *CurveNumberError) Error() string {
	return fmt.Sprintf("%q has an unreadable %s: %v", e.Raw, e.Half, e.Err)
}

func (e *CurveNumberError) Unwrap() error { return e.Err }

// CurveReason is why progression refused a curve, classified the same way and
// with the same caveat as AffinityReason: the refusal is the engine's, and an
// unrecognised one keeps the engine's own words.
type CurveReason int

const (
	CurveReasonUnclassified CurveReason = iota
	// CurveReasonNotPositive is a curve starting at zero or below.
	CurveReasonNotPositive
	// CurveReasonShrinks is a curve whose maximum is under its base, which
	// would make a stat fall as the character levels.
	CurveReasonShrinks
)

// CurveRefusedError is a readable curve the stat it is for will not accept.
type CurveRefusedError struct {
	Kind   progression.Kind
	Curve  progression.Curve
	Reason CurveReason
	Err    error
}

func (e *CurveRefusedError) Error() string { return e.Err.Error() }
func (e *CurveRefusedError) Unwrap() error { return e.Err }

// StatFieldError names which stat a curve refusal belongs to, which a form
// column already shows and a flag-driven run does not.
type StatFieldError struct {
	Kind progression.Kind
	Err  error
}

func (e *StatFieldError) Error() string { return fmt.Sprintf("%s: %v", e.Kind, e.Err) }
func (e *StatFieldError) Unwrap() error { return e.Err }

// Field names an answer whose shape internal/core/cast judges rather than this
// package.
type Field int

const (
	FieldID Field = iota
	FieldImage
)

// FieldRefusedError is a value internal/core/cast rejected.
//
// Those messages stay in English wherever they are shown. They describe the
// shape of a data file — a dot too many, a backslash, an extension that is
// neither .svg nor .png — and reproducing that judgement in a second language
// would mean restating a rule this package does not own. A front-end puts a
// lead-in in the author's language in front of the parser's own words.
type FieldRefusedError struct {
	Field Field
	Value string
	Err   error
}

func (e *FieldRefusedError) Error() string { return e.Err.Error() }
func (e *FieldRefusedError) Unwrap() error { return e.Err }

// YearError is a work's year that will not read as a number.
type YearError struct {
	Raw string
	Err error
}

func (e *YearError) Error() string {
	return fmt.Sprintf("the year %q is not a number; leave it empty if it is unknown", e.Raw)
}

func (e *YearError) Unwrap() error { return e.Err }
