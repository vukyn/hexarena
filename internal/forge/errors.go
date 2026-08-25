package forge

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
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

// CarryError is a kit holding a skill the affinity may not use, with the three
// things that explain it: what the character is, which skill, and what that
// skill is.
type CarryError struct {
	Affinity element.Affinity
	Skill    string
	Element  element.Element
}

func (e *CarryError) Error() string {
	return fmt.Sprintf("%s cannot carry %q, which is %s", e.Affinity, e.Skill, e.Element)
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
