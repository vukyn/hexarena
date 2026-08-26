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
	// Power is the share of the holder's attack the reply deals, in parts per
	// thousand. Nought is a reply that only applies statuses.
	Power int
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
type Resistance struct {
	Status string
	Amount int
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
	Status string `json:"status"`
	Stacks int    `json:"stacks"`
}

// passiveFile is the shape a passive is written in, and therefore the shape it
// is read in: Marshal builds one of these, so the writer cannot describe a field
// the parser does not read.
type passiveFile struct {
	ID string `json:"id"`
	// Written only when there is one, so a book that names none round-trips to
	// the bytes it was authored as.
	Name    string            `json:"name,omitempty"`
	Grants  []grantFile       `json:"grants"`
	Applies []applicationFile `json:"applies,omitempty"`
	Replies *replyFile        `json:"replies,omitempty"`
	While   *conditionFile    `json:"while,omitempty"`
	Resists []resistanceFile  `json:"resists,omitempty"`
}

type applicationFile struct {
	Status string `json:"status"`
	Chance int    `json:"chance"`
	Stacks int    `json:"stacks,omitempty"`
}

type replyFile struct {
	Power   int               `json:"power,omitempty"`
	Applies []applicationFile `json:"applies,omitempty"`
}

type conditionFile struct {
	BelowHealth int `json:"below_health"`
}

type resistanceFile struct {
	Status string `json:"status"`
	Amount int    `json:"amount"`
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
		len(declared.Applies) == 0 && declared.Replies == nil {
		return fail("grants nothing, resists nothing, adds nothing and answers nothing, so holding it would change nothing")
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
		grants = append(grants, Grant{Status: kind.ID, Stacks: stacks})
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
		if resist.Amount < 1 || resist.Amount > scale.Base {
			return fail("resists %q by %d, want a share in parts per thousand",
				kind.ID, resist.Amount)
		}
		if slices.ContainsFunc(resists, func(seen Resistance) bool { return seen.Status == kind.ID }) {
			return fail("resists %q twice; say one share instead", kind.ID)
		}
		resists = append(resists, Resistance{Status: kind.ID, Amount: resist.Amount})
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
		if declared.Replies.Power < 0 {
			return fail("answers for %d power, and a reply cannot heal what attacked it",
				declared.Replies.Power)
		}
		replies = &Reply{Power: declared.Replies.Power, Applies: answers}
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

	return Passive{
		ID: declared.ID, Name: strings.TrimSpace(declared.Name),
		Grants: grants, Applies: applies, Replies: replies,
		While: while, Resists: resists,
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
			grants = append(grants, grantFile{Status: grant.Status, Stacks: grant.Stacks})
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
		var replies *replyFile
		if current.Replies != nil {
			var answers []applicationFile
			for _, add := range current.Replies.Applies {
				answers = append(answers, applicationFile{
					Status: add.Status, Chance: add.Chance, Stacks: add.Stacks,
				})
			}
			replies = &replyFile{Power: current.Replies.Power, Applies: answers}
		}
		var while *conditionFile
		if current.While != nil {
			while = &conditionFile{BelowHealth: current.While.BelowHealth}
		}
		file.Passives = append(file.Passives, passiveFile{
			ID: current.ID, Name: current.Name, Grants: grants,
			Applies: applies, Replies: replies, While: while, Resists: resists,
		})
	}
	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode passive book: %w", err)
	}
	return append(out, '\n'), nil
}
