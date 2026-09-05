// Package passive declares what a character *has* rather than what it uses.
//
// Every skill is spent on a turn. A passive is not: it is in force from the
// moment a unit is enlisted, and nothing the unit or its opponents do turns it
// off. That is the whole distinction, and it is why a passive is declared here
// rather than as a skill with a flag on it — a skill book full of entries that
// can never be chosen is a book that lies about what a unit can do.
//
// Nothing here is a new rule. A passive grants statuses, and a status already
// knows how to change a stat: the terms belong to the status, the saturation
// belongs to modifier.Set, and both are inherited rather than reimplemented. A
// passive that composed with temporary buffs instead of saturating alongside
// them would be the one place in this game where stacking explodes, and reusing
// the status is what makes that unwritable.
//
// It computes nothing and reads no filesystem. Turning a declaration into an
// effect is the battle layer's job, because only the battle knows which unit is
// holding it.
package passive

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

// Grant is a status a passive puts on its holder, and how many stacks of it.
//
// There is no chance here and there is not going to be one. A passive is a fact
// about a unit rather than an event that happened to it, so a grant that
// sometimes failed would make two units built from the same character different
// in a way nothing on screen could explain.
type Grant struct {
	Status string
	Stacks int
	// Power is how big the granted status is, in parts per thousand of the stat
	// Scaling names, and it is read only by a grant whose status carries a
	// quantity — which today is an absorbing guard and nothing else.
	//
	// # Why a grant needed a number at all
	//
	// Every grant before this one was a *switch*: `toughened` is a status whose
	// modifiers say what it does, so granting it needs nothing but its name.
	// A barrier is not a switch — it is a pool, and how deep the pool is is the
	// whole of what the trait says. Set.Hold applied a nought amount for exactly
	// this reason ("a permanent status can be neither a damage-over-time nor a
	// regeneration, so there is nothing here to snapshot"), and a barrier granted
	// that way would have been a guard that stops nothing at all.
	//
	// ⚠️ It is read against the holder's **base** line rather than its buffed
	// one, and that is a determinism rule rather than a preference. Grants go on
	// at enlistment, one trait after another, so a grant reading buffed stats
	// would depend on which traits happened to be applied first — a unit with
	// `endurance` and a defence-scaled barrier would get a bigger barrier in one
	// declaration order than another, and nothing on either trait would say so.
	Power int
	// Scaling is which of the holder's stats Power is a share of. Absent is
	// refused where Power is required, because the zero value is health and a
	// guard scaled off health is a different design question.
	Scaling progression.Kind
}

// Passive is one declared trait.
type Passive struct {
	ID string
	// Name is the authored display name, and this package knows nothing about it
	// beyond that it is text — the same arrangement skill.Name has, and for the
	// same reason: a trait and the name it is called by are authored in one
	// sitting, and a separate translations file is a second thing to keep in
	// step. Absent is a real answer; a passive with no name renders as its id.
	Name string
	// Flavour is the authored clause a description opens with, and it is the one
	// thing about a trait that is written rather than derived.
	//
	// It sits beside Name for the same reason skill.Flavour does: both are the
	// words a trait is read in, and a name alone was never enough. A trait's
	// sentences are one per declared field — "always carries cứng đòn", "refuses
	// bỏng outright" — and they say what it does without ever saying what it is,
	// so a reader meets the mechanism with nothing to hang it on. The clause is
	// where "máu nó là nọc" is allowed to be said, because nothing derives that
	// from a resistance and a reply.
	//
	// It is safe for exactly one reason: it may carry **no figure**, which
	// ParseBook enforces. A clause with no number in it cannot be made wrong by
	// changing a number, which is what lets prose sit beside derived sentences
	// without the drift that Archetype.Demands was made derived to avoid.
	//
	// ⚠️ And it may name **no body**, which is a rule the seed tests hold rather
	// than this package. A skill free for anybody to carry may not say "mai" and
	// a restricted one may, because its restriction guarantees the body — a trait
	// has **no restriction mechanism at all**, so for a trait the ban is
	// unconditional. Nothing here can be authored its way out of.
	//
	// Absent is a real answer, the way it is for a skill: a trait with no clause
	// opens on its derived sentences, which is what every one of them read like
	// before this existed.
	Flavour string
	// Grants are the statuses the holder carries while the trait is in force,
	// which is the whole battle unless While says otherwise.
	//
	// Every one of them must be a permanent status, which ParseBook enforces.
	// That is not about how long the trait lasts — a gated one comes and goes —
	// but about who decides: a permanent status is one nothing in the game can
	// dispel, so the only thing that can take a grant off is the gate that put it
	// on. A timed status here would wear off on the holder's own turns with
	// nothing to put it back, which is a trait that quietly stopped being true.
	Grants []Grant
	// Applies are the statuses the holder adds to what its own damaging skills
	// inflict, on top of whatever those skills declare.
	//
	// skill.Application is reused rather than copied: "this inflicts that, at a
	// fixed chance" is a shape that already exists, and a second one beside it is
	// how the two drift apart. So a trait contributes to the list a skill's own
	// applications are drawn from, which is a change in battle rather than a new
	// rule.
	//
	// It rides on a skill that **deals damage** and on no other. A support skill
	// aimed at an ally would otherwise carry a hostile rider to the ally:
	// resolveAgainst deliberately never asks which side a target is on, so
	// "already hurting whatever it hits" is the available way to say "hostile",
	// and it is the honest one — a damaging skill aimed across the midline is
	// already an attack on whoever is standing there.
	Applies []skill.Application
	// Renews are the statuses the holder puts back on itself at the start of
	// every turn of its own.
	//
	// ⚠️ **It is the answer to the rule above rather than a hole in it.** Grants
	// take permanent statuses only, because a timed one would wear off on the
	// holder's own turns with nothing to put it back. This field is the nothing
	// to put it back: a trait that renews says so once and the turn does the
	// rest, so a timed status is exactly what belongs here and a permanent one is
	// refused as a grant written in the wrong place.
	//
	// They land on the holder, which is why a permanent status here would be a
	// grant and not a renewal, and they land after the tick that spends
	// durations -- ahead of it the buff would lose its first turn to the very
	// tick that was meant to leave it standing.
	Renews []skill.Application
	// Replies is what the trait costs an attacker, or nil when it answers
	// nothing.
	Replies *Reply
	// While gates the trait, or nil when it is always in force.
	//
	// It gates the *whole* trait rather than one field of it: the grants, the
	// resistances and the applications all come and go together. A trait wanting
	// one gated half and one ungated half is two traits, and saying so is what
	// keeps a gate from being a per-field flag nobody can read off the data.
	While *Condition
	// Resists is the share of an incoming application's chance the holder
	// refuses, per status.
	//
	// It is the one thing a passive does that had no home already: nothing in the
	// engine could refuse a status before this. A stat change reuses
	// status.Kind.Modifiers, so a trait grants and the existing code does the
	// rest; there was no equivalent field for "this unit says no", and the roll
	// that decides an application belonged entirely to the skill making it.
	Resists []Resistance
	// Drains is the share of the damage its holder deals that comes back as
	// health, in parts per thousand, and nought is "does not drain".
	//
	// # Why a trait may hold one when a skill already can
	//
	// A skill's drain belongs to the skill: leech_seed takes six hundred because
	// that is what leech_seed is, and a unit carrying it drains only on the turns
	// it casts it. A trait's belongs to the unit, so everything it does drains —
	// which is the difference between a skill that heals and a build that
	// sustains, and there was no way to write the second.
	//
	// It is a **share** rather than a granted status, and that is what makes it
	// gateable. A grant is applied once at enlistment and cannot be taken back,
	// so While is refused on one; a share is read fresh on every strike, so a
	// gate on this works exactly as written. That is the whole of why "drains
	// harder when badly hurt" is writable here and "hits harder when badly hurt"
	// is not.
	Drains int
	// Converts is the share of every blow its holder throws that meets no defence
	// at all, in parts per thousand.
	//
	// # Why it is here rather than on a skill
	//
	// skill.Pierce already lets a *skill* ignore armour, and this is the other
	// half of the same sentence: piercing belongs to razor_leaf because that is
	// what razor_leaf is, so every carrier gets the same ratio. Converting
	// belongs to whoever is swinging — "part of everything I throw goes straight
	// through" is a fact about a unit and applies to its whole kit — and there
	// was no way to write it. It is Drains' neighbour for exactly that reason:
	// that field exists because a skill's drain belongs to the skill and a
	// trait's belongs to the unit.
	//
	// ⚠️ **It is not a bigger pierce.** Piercing lowers the divisor, so a heavy
	// enough armour still eats most of what is left; converting splits the blow
	// and the converted share is divided by nothing, so it arrives whole however
	// deep the armour is. One is the answer to armour in general and this is the
	// answer to a wall — which is what makes it the damage dealer's tool against
	// a tank rather than a second helping of the same thing.
	//
	// Read fresh on every strike like Drains, so a gate on it works as written.
	Converts int
	// Spares is the share of a skill's health cost its holder does not pay, in
	// parts per thousand.
	//
	// # Why a share rather than a flag
	//
	// "Pays nothing" is the whole of what the first trait wanting this does, and a
	// bool would have said it — but a cost is a quantity and the interesting
	// designs are the partial ones: a trait that halves the price is a different
	// unit from one that removes it, and neither is expressible if the field can
	// only be on. Resists is the same shape for the same reason, and the base
	// means immunity in both.
	//
	// # What it does NOT spare
	//
	// Only skill.Cost — the health a caster hands over up front, whether or not
	// anything lands. It is not damage resistance, not a heal, and not a refusal
	// of a status; a trait that should blunt a poison tick says so with Resists.
	// The separation matters because the two read at opposite ends of a turn: a
	// cost is paid before a strike is rolled and a tick lands on somebody else's.
	//
	// ⚠️ **It has to be read where the RATING prices a cost as well as where the
	// battle charges it.** Battle.spendHealth is what a caster pays and
	// pricing.spentHealth is what Suggest thinks it will pay; a trait applied to
	// only the first makes a unit that never casts the skill it holds the trait
	// for, because the rating still sees the full price and declines.
	Spares int
	// Amplifies is what the holder is better at inflicting, which is the one
	// field here that reads the *other* unit's side of an application: every
	// other job a trait has is about its holder, and this one is about what its
	// holder does to somebody else.
	Amplifies []Amplification
}

// Reply is what a trait costs whoever attacked its holder.
//
// It is the one thing a passive does that fires on somebody else's turn, from a
// unit that is not acting. `Applies` is the mirror of it and is not it: that one
// rides on the holder's own attack, at a moment the battle is already resolving
// a skill the holder chose.
//
// # What it is not allowed to be
//
// It is not a second damage path. The battle prices it through `combat.Rules`
// like everything else and writes it to the log as damage, so a replay reads it
// as events and `--verify` re-runs it from the seed. Anything that could only be
// seen by reading `*Battle` would be a rule the renderer and the verifier could
// not agree on.
//
// # The two numbers, and what is deliberately missing
//
// Power is a share of the holder's attack, in parts per thousand, exactly as a
// skill's is — so a reply is read against the same scale as the attacks around
// it, and nought is the honest way to write a trait that answers with a status
// and no damage.
//
// There is no element and no accuracy, and both absences are the same decision:
// a reply is not an attack the holder chose to make. The elemental chart prices
// what one creature threw at another, and a trait reading it would make a fire
// creature's blood weak to water for a reason written nowhere on the trait; an
// accuracy roll asks whether contact was made, and contact is the thing that has
// already happened. So a reply is neutral, and it lands.
type Reply struct {
	// Power is the share of the stat below that the reply deals, in parts per
	// thousand. Nought is a reply that only applies statuses.
	Power int
	// Scaling is which of the holder's stats the reply is priced against, and
	// whether it reads the base line or the modified one. Attack unless the
	// trait says otherwise, which is what every damaging thing in this game
	// scales off by default.
	//
	// It is authored because a reply is the one damaging thing whose owner is
	// not choosing to use it. A trait that answers whoever hit it belongs to a
	// unit built to be hit, and such a unit is armoured rather than sharp -- so
	// pricing every reply off attack made thorns worth least to exactly the
	// character thorns are for. Blastoise carries 640 defence and 460 attack;
	// the same share off the wrong stat is a third less.
	Scaling skill.Scaling
	// Applies are the statuses the reply puts on the attacker. Timed only, for
	// the same reason a rider is: a permanent one would be an effect on somebody
	// else that nothing in the game could ever take off.
	Applies []skill.Application
}

// Answers reports whether the reply does anything at all. A trait that answers
// with no damage and no status would be a rule with no effect, which ParseBook
// refuses rather than accepting and skipping.
func (r *Reply) Answers() bool {
	return r != nil && (r.Power > 0 || len(r.Applies) > 0)
}

// Resistance is how much of one status's application chance its holder refuses,
// in parts per thousand.
//
// # Why a status rather than a category
//
// A category would read as the tidier choice — Cleanse takes categories, and for
// good reason — but it cannot say the thing that is actually wanted. "Immune to
// poison" as a category is "immune to every damage-over-time", which hands the
// holder immunity to burn as well; there is no way to be exact. An id can always
// name a class by listing it, while a category can never name one member of one.
//
// So the cost of ids is verbosity on a trait meaning to cover a whole class, and
// the cost of categories is being unable to express the case that motivated the
// feature. A category resistance can be added beside this later if a class is
// ever wanted; adding it the other way round would mean shipping something wrong
// first.
//
// # Why a ratio, and what a full thousand means
//
// A ratio because refusing outright is a hard cap on a continuous quantity, and
// this engine has chosen against that everywhere: buffs saturate, piercing is a
// share, dodge approaches a floor it never reaches. A declared full thousand is
// then the explicit way to write true immunity — exactly as a skill declaring
// full accuracy is the explicit way to write an effect that must land. The
// absolute is available to whoever authors it and never falls out of stacking.
//
// # A negative share is a vulnerability
//
// Amount may be negative, and then the status lands **more** easily on the
// holder. That is a whole feature reusing the arithmetic rather than a second
// field: the chance is multiplied by what the resistance lets through, so a
// share of -300 lets 1300 through and a trait that says "no guard at all" costs
// what it says. Nought is refused in both directions — a share of nothing is a
// status the trait does not touch, and the way to write that is to leave the
// entry out.
//
// It inherits the harmful-only rule for free, and that is the right answer
// rather than a happy accident: refusing a status the holder's own side puts on
// it is nonsense, and so is *inviting* one. A vulnerability is about being easier
// to hurt.
//
// ⚠️ It composes with a resistance to the same status through the same multiply,
// so the two do not cancel to nothing — a -500 and a +500 leave 1500*500/1000 =
// 750, three quarters, not the full chance. Anybody expecting them to annihilate
// is reading it as addition, which is what shares do and chances do not.
type Resistance struct {
	Status string
	// Amount is the share refused, in parts per thousand. Negative is a
	// vulnerability; the full base is declared immunity; nought is refused.
	Amount int
}

// Amplification is a status the trait's holder is better at inflicting.
//
// # Why two shares rather than two features with one share each
//
// "Makes my poison better" is two different things and they land in two
// different places: a **stronger tick**, which folds into the one multiplication
// battle freezes on the stack, and a **better chance**, which lands at the site a
// resistance already bites. They read differently in play — a tick is worth more
// the longer a stack lives, a chance the more often the skill is cast — so a
// trait must be able to want either alone, and either share may be left out.
//
// One entry per status rather than two lists, because two lists would let the
// same status be named in both and there would be nothing to say which entry was
// the real one. Leaving a share at nought is how a trait says it does not touch
// that half.
//
// # Why this one is not restricted to harmful statuses
//
// Resistance is, and the asymmetry is the point rather than an oversight:
// refusing a status the holder's own side puts on it is nonsense, while making
// one *better* is exactly as sensible for a shield as for a poison. What is
// refused instead is narrower and more useful — an Effect on a status with no
// tick to raise, which is a trait that cannot do the thing it says.
//
// # Why the shares compose by multiplying, and what that costs
//
// The same arithmetic a resistance composes with, read in the other direction:
// a chance is multiplied by everything raising it and everything lowering it, so
// the order they are applied in cannot matter and neither side has to know the
// other exists. The cost is that resistances stacking *diminish* for free while
// amplifiers stacking *compound* — two shares of three hundred are 69 percent
// rather than 60. The guard is the per-trait bound plus the fact that a unit
// holds a handful of traits and cannot restack them the way it can restack a
// buff; if that ever stops being true, scale.Saturate is where this goes.
type Amplification struct {
	Status string
	// Effect is the share added to a damage-over-time tick, in parts per
	// thousand. Only a ticking status has one.
	Effect int
	// Chance is the share added to the application's chance, in parts per
	// thousand.
	Chance int
}

// Condition is when a trait is in force.
//
// # Why this is not skill.Condition
//
// Reusing the existing vocabulary was the plan, and it does not fit. A skill's
// condition asks what the *target* is carrying — a status, a stack count, and
// whether to spend it — because it exists to pay a skill off for arriving after
// a debuff. What a trait wants to ask is about its *holder*, and the question it
// wants most is one no status can answer: how hurt am I. Bending
// skill.Condition to carry a health share would give one type two unrelated
// jobs, and the note that said to reuse it also said this term was the work.
//
// One term today, and it is a share rather than a number of points: a threshold
// in points would mean a different fraction of the bar at every level, so a
// trait authored for a level-eight unit would be permanently on at sixty.
type Condition struct {
	// BelowHealth is the share of its maximum health the holder must be at or
	// under, in parts per thousand. A third is 333.
	BelowHealth int
}

// Holds reports whether the condition is met by a unit at the given health.
//
// At or under, not strictly under: a threshold of a third means a third counts.
// Maximum health of nought answers no rather than dividing by it — a unit with
// no maximum is not a unit that is hurt.
func (c *Condition) Holds(health, maximum int64) bool {
	if c == nil {
		return true
	}
	return scale.AtOrBelowShare(health, maximum, c.BelowHealth)
}

// StatusIDs is the statuses the passive grants, in declaration order. It is what
// a listing and a refusal want, and it saves every caller writing the same loop.
func (p Passive) StatusIDs() []string {
	out := make([]string, 0, len(p.Grants))
	for _, grant := range p.Grants {
		out = append(out, grant.Status)
	}
	return out
}

// Deps are the books a passive's declarations are checked against. Validating
// here rather than at use is the whole point: a passive naming a status that
// does not exist is a data error, and a data error should stop the load.
type Deps struct {
	Statuses *status.Book
}

// Book is the declared passives, in the order they were written.
type Book struct {
	passives []Passive
	byID     map[string]Passive
}

type grantFile struct {
	Status  string `json:"status"`
	Stacks  int    `json:"stacks"`
	Power   int    `json:"power,omitempty"`
	Scaling string `json:"scaling,omitempty"`
}

// passiveFile is the shape a passive is written in, and therefore the shape it
// is read in: Marshal builds one of these, so the writer cannot describe a field
// the parser does not read.
type passiveFile struct {
	ID string `json:"id"`
	// Written only when there is one, so a book that names none round-trips to
	// the bytes it was authored as.
	Name string `json:"name,omitempty"`
	// Flavour sits beside the name and is omitted when absent, so a book that
	// declares neither round-trips to the bytes it was authored as.
	Flavour   string              `json:"flavour,omitempty"`
	Grants    []grantFile         `json:"grants"`
	Applies   []applicationFile   `json:"applies,omitempty"`
	Renews    []applicationFile   `json:"renews,omitempty"`
	Replies   *replyFile          `json:"replies,omitempty"`
	While     *conditionFile      `json:"while,omitempty"`
	Resists   []resistanceFile    `json:"resists,omitempty"`
	Drains    int                 `json:"drains,omitempty"`
	Converts  int                 `json:"converts,omitempty"`
	Spares    int                 `json:"spares,omitempty"`
	Amplifies []amplificationFile `json:"amplifies,omitempty"`
}

type applicationFile struct {
	Status string `json:"status"`
	Chance int    `json:"chance"`
	Stacks int    `json:"stacks,omitempty"`
}

type scalingFile struct {
	Stat   string `json:"stat"`
	Source string `json:"source,omitempty"`
}

type replyFile struct {
	Power   int               `json:"power,omitempty"`
	Scaling *scalingFile      `json:"scaling,omitempty"`
	Applies []applicationFile `json:"applies,omitempty"`
}

type conditionFile struct {
	BelowHealth int `json:"below_health"`
}

type resistanceFile struct {
	Status string `json:"status"`
	Amount int    `json:"amount"`
}

type amplificationFile struct {
	Status string `json:"status"`
	Effect int    `json:"effect,omitempty"`
	Chance int    `json:"chance,omitempty"`
}

type bookFile struct {
	Passives []passiveFile `json:"passives"`
}

// ParseBook reads a passive declaration. It never touches the filesystem; the
// caller supplies the bytes.
func ParseBook(raw []byte, deps Deps) (*Book, error) {
	if deps.Statuses == nil {
		return nil, fmt.Errorf("a passive book needs the status book to check against")
	}
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode passive book: %w", err)
	}
	book := &Book{byID: make(map[string]Passive, len(file.Passives))}
	for _, declared := range file.Passives {
		resolved, err := resolve(declared, deps)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[resolved.ID]; clash {
			return nil, fmt.Errorf("passive %q is declared twice", resolved.ID)
		}
		book.byID[resolved.ID] = resolved
		book.passives = append(book.passives, resolved)
	}
	return book, nil
}

func resolve(declared passiveFile, deps Deps) (Passive, error) {
	if declared.ID == "" {
		return Passive{}, fmt.Errorf("a passive needs an id")
	}
	fail := func(format string, args ...any) (Passive, error) {
		return Passive{}, fmt.Errorf("passive %q: "+format,
			append([]any{declared.ID}, args...)...)
	}
	if len(declared.Grants) == 0 && len(declared.Resists) == 0 &&
		len(declared.Applies) == 0 && len(declared.Renews) == 0 && declared.Replies == nil &&
		declared.Drains == 0 && declared.Converts == 0 && declared.Spares == 0 &&
		len(declared.Amplifies) == 0 {
		return fail("grants nothing, renews nothing, resists nothing, adds nothing, answers nothing, drains nothing, converts nothing, spares nothing and amplifies nothing, so holding it would change nothing")
	}
	// The one authored clause, and the one rule that makes it safe. A figure in
	// it is refused rather than trusted, because every number in a description is
	// derived and a clause reading "gấp đôi" would outlive the amount that made
	// it true. Digits rather than a percent sign, since "40" and "gấp 2" are the
	// same mistake wearing different clothes. Said the same way skill.resolve
	// says it, because it is the same rule and a second wording of one rule is
	// how the two come to disagree about what counts.
	flavour := strings.TrimSpace(declared.Flavour)
	if index := strings.IndexFunc(flavour, unicode.IsDigit); index >= 0 {
		return fail("has a flavour clause carrying the figure %q; every number in a description is derived, and an authored one would outlive what it describes",
			flavour[index:index+1])
	}

	grants := make([]Grant, 0, len(declared.Grants))
	for _, grant := range declared.Grants {
		kind, err := deps.Statuses.Lookup(grant.Status)
		if err != nil {
			return fail("%w", err)
		}
		if !kind.Permanent {
			return fail("grants %q, which is timed: it would wear off on the holder's own turns and a passive is granted only once",
				kind.ID)
		}
		// An unstated stack count is one, the way a skill's unstated strike count
		// is one.
		stacks := max(grant.Stacks, 1)
		if stacks > kind.MaxStacks {
			return fail("grants %d stacks of %q, which caps at %d", stacks, kind.ID, kind.MaxStacks)
		}
		if slices.ContainsFunc(grants, func(seen Grant) bool { return seen.Status == kind.ID }) {
			return fail("grants %q twice; say the stack count instead", kind.ID)
		}
		// A quantity is required exactly where the status is one, and refused
		// everywhere else. A barrier without a depth stops nothing and would
		// still parse; a power on a switch would be a figure nothing reads, which
		// is the shape every unread field in this repository has been refused in.
		carries := kind.Category == status.Absorb
		switch {
		case carries && grant.Power < 1:
			return fail("grants %q without a power: a guard that holds a pool needs a depth, or it stops nothing",
				kind.ID)
		case !carries && grant.Power != 0:
			return fail("grants %q with a power of %d, which only a guard holding a pool reads",
				kind.ID, grant.Power)
		}
		scaling := progression.Kind(0)
		if carries {
			// ⚠️ A gated trait may not grant one, and this is the rule that
			// permanent-absorb was let through for. hold and release run the
			// grant again every time the gate reopens, so a barrier behind
			// `below_health` would come back full each time its holder crossed
			// the line — a wall with no cost, refilled by being hit. An ungated
			// grant runs once at enlistment, which is exactly "puts a barrier up
			// when the battle starts".
			if declared.While != nil {
				return fail("grants %q behind a gate: a pool is refilled every time a gate reopens, so a guard may only be granted by a trait that is always in force",
					kind.ID)
			}
			// Through skill.ParseScaling rather than a second reading of the
			// same question, which is the sentence that function is exported
			// under — and it brings the health refusal with it, so a barrier
			// cannot be scaled off the bar it is protecting.
			//
			// The base line rather than the current one, for the reason on
			// Grant.Power: grants go on one after another at enlistment, so
			// reading a buffed stat would make the answer depend on which trait
			// was applied first.
			parsed, err := skill.ParseScaling(grant.Scaling, "base")
			if err != nil {
				return fail("grants %q, which %w", kind.ID, err)
			}
			scaling = parsed.Stat
		}
		grants = append(grants, Grant{
			Status: kind.ID, Stacks: stacks, Power: grant.Power, Scaling: scaling,
		})
	}

	resists := make([]Resistance, 0, len(declared.Resists))
	for _, resist := range declared.Resists {
		kind, err := deps.Statuses.Lookup(resist.Status)
		if err != nil {
			return fail("%w", err)
		}
		// Only what can be done *to* the holder. A trait resisting a buff would
		// be refusing its own side's help, and a trait resisting a shield or a
		// regeneration the same — Harmful is the existing split and this is the
		// second thing to lean on it, so the two cannot disagree about which
		// categories are an attack.
		if !kind.Category.Harmful() {
			return fail("resists %q, which is a %s: there is nothing to refuse in a status the holder's own side puts on it",
				kind.ID, kind.Category)
		}
		// A negative share is a **vulnerability**: the status lands more easily
		// on the holder rather than less. It reuses the whole composition rather
		// than adding a field, because it is the same multiplication read in the
		// other direction — see Resistance.Amount.
		//
		// Nought is refused in both directions. A share of nothing is a trait
		// naming a status it does not touch, and the way to say that is to leave
		// the entry out; writing it down would be a line in the description
		// promising an effect of zero.
		if resist.Amount < -scale.Base || resist.Amount > scale.Base || resist.Amount == 0 {
			return fail("resists %q by %d, want a share in parts per thousand — negative to be more easily afflicted, and not nought",
				kind.ID, resist.Amount)
		}
		if slices.ContainsFunc(resists, func(seen Resistance) bool { return seen.Status == kind.ID }) {
			return fail("resists %q twice; say one share instead", kind.ID)
		}
		resists = append(resists, Resistance{Status: kind.ID, Amount: resist.Amount})
	}

	amplifies := make([]Amplification, 0, len(declared.Amplifies))
	for _, raise := range declared.Amplifies {
		kind, err := deps.Statuses.Lookup(raise.Status)
		if err != nil {
			return fail("%w", err)
		}
		if raise.Effect == 0 && raise.Chance == 0 {
			return fail("amplifies %q by nothing; say a share of its effect, of its chance, or of both",
				kind.ID)
		}
		// A damage-over-time's tick is the only effect there is to raise, and
		// refusing everything else is what keeps a trait from claiming a job it
		// cannot do: "makes my mire hit harder" has no number behind it because a
		// mire does not tick, and accepting the field would leave an author
		// waiting for a change that never arrives.
		//
		// A regeneration is refused too, and the reason changed under it. It used
		// to be that battle.inflict computed a tick only for a Dot, so a regen
		// froze nought and accepting the share would have promised an author a
		// multiplication of zero. That bug is fixed: a regen now freezes a real
		// amount and heals from it every turn, so the share would multiply
		// something.
		//
		// It stays refused on the other ground, which was always the stronger of
		// the two. The share is described to a player as "its poison ticks 30%
		// harder" — words of harm, in both languages — and a share that heals
		// under that sentence is a description that lies, which is the one thing
		// every derived description in this engine exists to prevent. Lifting it
		// is therefore a wording change first and a one-line condition second,
		// and worth doing only when a trait actually wants it.
		if raise.Effect != 0 && kind.Category != status.Dot {
			return fail("amplifies the effect of %q, which is a %s: only a damage-over-time has a tick this could raise",
				kind.ID, kind.Category)
		}
		for _, share := range []struct {
			name   string
			amount int
		}{{"effect", raise.Effect}, {"chance", raise.Chance}} {
			if share.amount == 0 {
				continue
			}
			if share.amount < 1 || share.amount > scale.Base {
				return fail("amplifies the %s of %q by %d, want a share in parts per thousand",
					share.name, kind.ID, share.amount)
			}
		}
		if slices.ContainsFunc(amplifies, func(seen Amplification) bool {
			return seen.Status == kind.ID
		}) {
			return fail("amplifies %q twice; say one entry with both shares instead", kind.ID)
		}
		amplifies = append(amplifies,
			Amplification{Status: kind.ID, Effect: raise.Effect, Chance: raise.Chance})
	}

	renews, err := readApplications(declared.Renews, deps, "renews")
	if err != nil {
		return fail("%w", err)
	}
	applies, err := readApplications(declared.Applies, deps, "adds")
	if err != nil {
		return fail("%w", err)
	}

	var replies *Reply
	if declared.Replies != nil {
		answers, err := readApplications(declared.Replies.Applies, deps, "answers with")
		if err != nil {
			return fail("%w", err)
		}
		scaling := skill.DefaultScaling()
		if declared.Replies.Scaling != nil {
			var err error
			scaling, err = skill.ParseScaling(
				declared.Replies.Scaling.Stat, declared.Replies.Scaling.Source)
			if err != nil {
				return fail("answers with a share that %w", err)
			}
		}
		// A reply that deals no damage has no stat to be priced against, so
		// naming one is a clause the author will wait forever to see take effect.
		if declared.Replies.Power == 0 && declared.Replies.Scaling != nil {
			return fail("answers with no damage but names a stat to scale it off")
		}
		if declared.Replies.Power < 0 {
			return fail("answers for %d power, and a reply cannot heal what attacked it",
				declared.Replies.Power)
		}
		replies = &Reply{Power: declared.Replies.Power, Scaling: scaling, Applies: answers}
		// A reply that neither hurts nor inflicts anything is a rule with no
		// effect, and the trait around it may well have others — so this is
		// refused here rather than folded into the "changes nothing" check
		// above, which would let a trait with one good half hide a dead one.
		if !replies.Answers() {
			return fail("answers with no damage and no status, so nothing would happen")
		}
	}

	var while *Condition
	if declared.While != nil {
		if declared.While.BelowHealth < 1 || declared.While.BelowHealth > scale.Base {
			return fail("is in force below %d health, want a share in parts per thousand",
				declared.While.BelowHealth)
		}
		while = &Condition{BelowHealth: declared.While.BelowHealth}
	}

	// The upper bound is the same one a skill's drain has, and for the same
	// reason: a share over the base takes back more health than the strike dealt
	// damage, which is not a strong trait but an incoherent one.
	if declared.Drains < 0 || declared.Drains > scale.Base {
		return fail("drains %d, want a share in parts per thousand", declared.Drains)
	}
	// Bounded on both sides for the reason the drain above is. Under nought is a
	// trait that makes its holder's blows *worse* through a field named for the
	// opposite; over the base is more than the whole blow converted, which is not
	// a stronger trait but a meaningless one — a thousand already means every
	// point of it meets no defence.
	if declared.Converts < 0 || declared.Converts > scale.Base {
		return fail("converts %d, want a share in parts per thousand", declared.Converts)
	}
	// Bounded on both sides for the reason the two above are. Under nought is a
	// trait that makes its holder pay MORE through a field named for paying less,
	// which is a vulnerability wearing the wrong name; over the base is a cost
	// that pays the caster, and a skill that heals its user for using it is a
	// different design question than this field answers.
	if declared.Spares < 0 || declared.Spares > scale.Base {
		return fail("spares %d of a skill's cost, want a share in parts per thousand", declared.Spares)
	}

	return Passive{
		ID: declared.ID, Name: strings.TrimSpace(declared.Name), Flavour: flavour,
		Grants: grants, Applies: applies, Renews: renews, Replies: replies,
		While: while, Resists: resists, Drains: declared.Drains,
		Converts: declared.Converts, Spares: declared.Spares, Amplifies: amplifies,
	}, nil
}

// readApplications is the rules a status a trait puts on somebody *else* has to
// obey, said once for the two places that put one there.
//
// A rider and a reply differ in when they fire and in nothing else, so the
// checks were going to be identical — and two copies of identical checks is how
// one of them quietly stops matching the other. The verb is passed in only so
// that the refusal reads as the thing the author wrote.
func readApplications(declared []applicationFile, deps Deps, verb string) ([]skill.Application, error) {
	out := make([]skill.Application, 0, len(declared))
	for _, add := range declared {
		kind, err := deps.Statuses.Lookup(add.Status)
		if err != nil {
			return nil, err
		}
		if add.Chance < 1 || add.Chance > scale.Base {
			return nil, fmt.Errorf("%s %q on a chance of %d, want a share in parts per thousand",
				verb, kind.ID, add.Chance)
		}
		stacks := max(add.Stacks, 1)
		if stacks > kind.MaxStacks {
			return nil, fmt.Errorf("%s %d stacks of %q, which caps at %d",
				verb, stacks, kind.ID, kind.MaxStacks)
		}
		// A trait cannot put a permanent status on somebody else. Those are what
		// a trait *grants*, to its own holder, where the trait's own gate is the
		// only thing that may take one back; on anybody else it would be an
		// effect nothing in the game could ever remove.
		if kind.Permanent {
			return nil, fmt.Errorf("%s %q, which is permanent: nothing could ever take it off the target",
				verb, kind.ID)
		}
		if slices.ContainsFunc(out, func(seen skill.Application) bool {
			return seen.Status == kind.ID
		}) {
			return nil, fmt.Errorf("%s %q twice; say one chance instead", verb, kind.ID)
		}
		out = append(out, skill.Application{Status: kind.ID, Chance: add.Chance, Stacks: stacks})
	}
	return out, nil
}

// Refuses is the share of an application's chance the passive takes off, for one
// status, or nought when it says nothing about it.
func (p Passive) Refuses(statusID string) int {
	for _, resist := range p.Resists {
		if resist.Status == statusID {
			return resist.Amount
		}
	}
	return 0
}

// Boosts is the shares the passive adds to one status's tick and to its chance,
// or nought for either it says nothing about.
//
// Both at once rather than one call each, because the two are one entry: asking
// twice would look up the same row twice and would let a caller read one half and
// forget the other, which is exactly the half that does not reach the log.
func (p Passive) Boosts(statusID string) (effect, chance int) {
	for _, raise := range p.Amplifies {
		if raise.Status == statusID {
			return raise.Effect, raise.Chance
		}
	}
	return 0, 0
}

// Lookup returns a declared passive, or says which one is missing.
func (b *Book) Lookup(id string) (Passive, error) {
	found, known := b.byID[id]
	if !known {
		return Passive{}, fmt.Errorf("unknown passive %q", id)
	}
	return found, nil
}

// All returns every passive in declaration order, as a copy.
func (b *Book) All() []Passive {
	out := make([]Passive, len(b.passives))
	copy(out, b.passives)
	for i := range out {
		out[i].Grants = slices.Clone(out[i].Grants)
		out[i].Applies = slices.Clone(out[i].Applies)
		out[i].Resists = slices.Clone(out[i].Resists)
		// The reply is a pointer holding a slice, so both have to be copied:
		// a caller handed the pointer could edit the book through it, and one
		// handed the same slice could edit it through that.
		if out[i].Replies != nil {
			answer := *out[i].Replies
			answer.Applies = slices.Clone(answer.Applies)
			out[i].Replies = &answer
		}
		// The condition is a pointer, so a caller editing what it was handed
		// would edit the book through it.
		if out[i].While != nil {
			gate := *out[i].While
			out[i].While = &gate
		}
	}
	return out
}

// IDs returns the declared ids in declaration order.
func (b *Book) IDs() []string {
	out := make([]string, 0, len(b.passives))
	for _, current := range b.passives {
		out = append(out, current.ID)
	}
	return out
}

// Marshal writes the book as the declaration a parse would read back, in
// declaration order, so editing one entry is a one-entry diff.
func (b *Book) Marshal() ([]byte, error) {
	file := bookFile{Passives: make([]passiveFile, 0, len(b.passives))}
	for _, current := range b.passives {
		grants := make([]grantFile, 0, len(current.Grants))
		for _, grant := range current.Grants {
			// The depth and the stat it is a share of, or a written book loses a
			// barrier's size and reloads as a guard that stops nothing. Both are
			// omitempty, so every grant that carries no quantity round-trips to
			// the bytes it was authored as.
			written := grantFile{Status: grant.Status, Stacks: grant.Stacks, Power: grant.Power}
			if grant.Power > 0 {
				written.Scaling = grant.Scaling.String()
			}
			grants = append(grants, written)
		}
		// Written only when there are any, so a trait that resists nothing
		// round-trips to the bytes it was authored as — the same reason the name
		// is omitted when there is none.
		var resists []resistanceFile
		for _, resist := range current.Resists {
			resists = append(resists,
				resistanceFile{Status: resist.Status, Amount: resist.Amount})
		}
		var applies []applicationFile
		for _, add := range current.Applies {
			applies = append(applies, applicationFile{
				Status: add.Status, Chance: add.Chance, Stacks: add.Stacks,
			})
		}
		var renewed []applicationFile
		for _, add := range current.Renews {
			renewed = append(renewed, applicationFile{
				Status: add.Status, Chance: add.Chance, Stacks: add.Stacks,
			})
		}
		var replies *replyFile
		if current.Replies != nil {
			var answers []applicationFile
			for _, add := range current.Replies.Applies {
				answers = append(answers, applicationFile{
					Status: add.Status, Chance: add.Chance, Stacks: add.Stacks,
				})
			}
			replies = &replyFile{Power: current.Replies.Power, Applies: answers}
			if current.Replies.Scaling != skill.DefaultScaling() {
				replies.Scaling = &scalingFile{
					Stat:   current.Replies.Scaling.Stat.String(),
					Source: current.Replies.Scaling.Source.String(),
				}
			}
		}
		var while *conditionFile
		if current.While != nil {
			while = &conditionFile{BelowHealth: current.While.BelowHealth}
		}
		var amplifies []amplificationFile
		for _, raise := range current.Amplifies {
			amplifies = append(amplifies, amplificationFile{
				Status: raise.Status, Effect: raise.Effect, Chance: raise.Chance,
			})
		}
		file.Passives = append(file.Passives, passiveFile{
			ID: current.ID, Name: current.Name, Flavour: current.Flavour, Grants: grants,
			Applies: applies, Renews: renewed, Replies: replies, While: while, Resists: resists,
			Drains: current.Drains, Converts: current.Converts, Spares: current.Spares,
			Amplifies: amplifies,
		})
	}
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode passive book: %w", err)
	}
	return append(out, '\n'), nil
}
