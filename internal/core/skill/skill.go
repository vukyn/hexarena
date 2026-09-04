// Package skill is where the rest of the engine meets: a skill names an element,
// a shape, a power, a scaling stat, the statuses it inflicts and the condition
// that makes it hit harder. Everything it names is validated against the book
// that declares it, so a typo in a data file fails at load rather than at the
// moment it would have mattered.
//
// Nothing here computes damage. A skill is a declaration; turning one into a
// combat.Hit is the battle layer's job, because only the battle knows the
// caster's current stats, the target's, and the state of the board.
package skill

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/core/status"
)

// Side is who a skill is aimed at.
type Side uint8

const (
	// Enemy aims across the battlefield.
	Enemy Side = iota
	// Ally aims at the caster's own half, the caster included.
	Ally
	// Self aims only at the caster.
	Self
	// All aims at either half, and a shape aimed this way spreads across the
	// midline instead of stopping at it.
	//
	// It is what a skill that does not care whose side a unit is on needs: a
	// battlefield-wide cleanse, a haste on everybody, and — since nothing here
	// distinguishes a friendly target from a hostile one once a cell is chosen —
	// a damaging skill that hurts the caster's own squad as well. That last one
	// is the point of having the value rather than a flaw in it; resolveAgainst
	// has never asked which side a target is on, so an ally-aimed damaging skill
	// already did the same thing.
	//
	// Declared last on purpose. The value is serialised by name like every
	// other enum here, so appending cannot reinterpret a skill book or a saved
	// log — but the numbering is what SideCount and TargetNames are built from,
	// and putting a new value in the middle would reorder the chooser a
	// front-end offers.
	All
)

// SideCount is the number of targeting sides.
const SideCount = int(All) + 1

var sideNames = [SideCount]string{Enemy: "enemy", Ally: "ally", Self: "self", All: "all"}

func (s Side) String() string {
	if int(s) >= SideCount {
		return fmt.Sprintf("side(%d)", uint8(s))
	}
	return sideNames[s]
}

// Reaches reports whether a skill aimed by a unit on one side of the board may
// be pointed at a cell on another.
//
// The rule is relational rather than absolute — "the other side" depends on
// whose turn it is — and it lives here because it is one rule with two callers:
// the legal aims a unit is offered and the aim an action is checked against.
// Before, battle worked out an absolute side and then flipped it for an enemy
// caster, which is the same fact written as a special case.
//
// Self is not asked here and answers no. It reaches exactly the caster's own
// cell, which is a unit rather than a side, so nothing can decide it from two
// sides alone; battle answers that one before it asks this.
func (s Side) Reaches(from, at hex.Side) bool {
	switch s {
	case All:
		return true
	case Ally:
		return at == from
	case Self:
		return false
	default:
		return at != from
	}
}

// CrossesSides reports whether a shape aimed by this skill may spread onto the
// other half of the board.
//
// pattern.Targets drops a splash cell that lands on the far side of the midline,
// which is right for every skill that aims at one side: an area attack on the
// enemy front rank must not catch the caster's own. A skill aimed at both sides
// is the one case where that drop is wrong, and this is what says so — once,
// for the two places that walk a shape.
func (s Side) CrossesSides() bool { return s == All }

// ParseSide resolves a targeting name as written in the data files.
func ParseSide(name string) (Side, error) {
	for i, candidate := range sideNames {
		if candidate == name {
			return Side(i), nil
		}
	}
	return 0, fmt.Errorf("unknown target side %q", name)
}

// Scaling names which stat a skill's damage comes from and which version of it
// to read.
type Scaling struct {
	Stat   progression.Kind
	Source combat.ScalingSource
}

// Application is a status a skill inflicts, with its own chance to take hold.
//
// The chance is fixed. It is not raised by accuracy, not lowered by dodge, and no
// stat touches it, so it stays a property of the skill rather than of whoever
// happens to be holding it. Landing the hit and inflicting the status are two
// separate questions: a skill that connects nine times in ten and poisons half
// the time poisons on 45 percent of casts, and both halves of that stay legible.
type Application struct {
	Status string
	Chance int
	Stacks int
}

// Condition is the amplifier that pays a skill off for arriving after a status.
type Condition struct {
	// Status and MinStacks are what the target must already be carrying. A
	// condition may leave them out when it reads health instead.
	Status    string
	MinStacks int
	// BelowHealth is the share of its maximum health the unit must be at or
	// under, in parts per thousand, and nought is "does not ask". Half is 500.
	//
	// *Which* unit is the field's business and not this type's: Skill.Requires
	// reads the target and Skill.SelfRequires reads the caster, exactly as
	// Applies and SelfApplies say who receives a status. That is why the pair is
	// two fields rather than one field with a "whose" flag -- the reader already
	// knows what the self_ prefix means here, and a flag would be a third thing
	// to get wrong.
	//
	// passive.Condition still reads its holder through a type of its own. The two
	// share their arithmetic through scale.AtOrBelowShare and deliberately not
	// their type, because passive imports this package and the language refuses
	// the shortcut anyway.
	BelowHealth int
	// Gates makes this a condition on **casting** rather than on the payoff.
	//
	// Every other condition in this engine is an amplifier: it is read while the
	// skill resolves, and a caster that fails it simply casts for its own figure.
	// A gating condition is read *before* the skill is offered, so a caster short
	// of what it names does not have the option at all -- which is the only shape
	// a cooldownless spender can take. Without it the spender is cast every turn
	// for the unamplified blow, and the design is a suggestion.
	//
	// ⚠️ **Opt-in, and it has to be.** Nine shipped skills carry min_stacks
	// across four characters -- flare, deluge, pyre, thorn_volley, cauterise,
	// bloom_burst, verdant_mend, tide_break and wellspring -- and every one was
	// authored as an amplifier that pays more when it holds. Reading min_stacks as
	// a gate globally would rebalance four characters at once, inside a change
	// whose subject is a tenth skill.
	//
	// ⚠️ **The caster's own condition only, it must read a status, and it must
	// spend one.** resolveCondition refuses the other three shapes and says why at
	// each: what they have in common is that the reading would have to be taken
	// per aim, and battle.aims is required to stay blind to a gate.
	Gates bool
	// BonusPower is added to the skill's power when the condition holds.
	BonusPower int
	// Consume removes the status, which is what a detonate does: the burst is
	// paid for by the ticks it throws away.
	Consume bool
	// ConsumeStacks is how many stacks a consume takes, and nought means all of
	// them, which is what every consume did before the field existed.
	//
	// The distinction is the difference between a detonate and a discharge. A
	// detonate spends a whole pile at once because the pile *is* the payment —
	// the burst is worth the ticks thrown away, so leaving some behind would be
	// leaving the payment behind. A counter spent one at a time is a magazine
	// instead, and taking the whole magazine for one shot is what would make
	// accumulating it pointless.
	ConsumeStacks int
	// Chains makes the consume travel: from the unit at the aim to every
	// hex-adjacent unit carrying the same status, and on from those, for as far
	// as an unbroken run of carriers reaches.
	//
	// It replaced a fixed pattern, and the difference is the whole idea. A
	// pattern is geometry — it covers the same three cells whoever is standing
	// in them — so a charged unit one cell outside it was never reached and an
	// uncharged unit inside it was hit anyway. A chain is the opposite: it goes
	// exactly where the charge is, and **it needs a charged body to step on**. A
	// gap of one uncharged cell stops it dead, which is what makes laying the
	// charge down a decision about *where* rather than only about how much.
	//
	// ⚠️ **The aim gates the whole thing.** A chain from an uncharged unit is
	// empty however much charge is standing beside it — nothing is consumed,
	// nothing arcs, and the skill is simply its own blow.
	Chains bool
	// ArcPower is what one consumed stack deals, as a share of the caster's
	// scaling stat, exactly as a skill's own power is.
	//
	// ⚠️ **It is added to the skill's own blow and never traded against it.** An
	// earlier design damped the blow to pay for the discharge, on the reasoning
	// that a skill hitting for its full figure *and* firing the charge would be
	// strictly better whenever the charge was there. That reasoning was wrong
	// about where the payment comes from: a stack does not appear on a target by
	// itself, and the turn that put it there is the price. A conduit is paid for
	// in **tempo** — the charging turns and the cooldown it sits on — so its own
	// figure is a constant, and a reader comparing two skills is comparing the
	// numbers on them rather than the numbers minus a condition.
	//
	// ⚠️ **It is not the skill's damage and does not behave like it.** It is the
	// charge going off, so it is not aimed, not rolled against accuracy or dodge,
	// and **not stopped by a shield**: a guard that swallows the blow does not
	// stop what was already sitting on the target. That is the one thing a
	// conduit has over an ordinary attack, and it is the reason the counter is
	// worth laying down in front of a wall.
	ArcPower int
	// StackPower is what one consumed stack adds to the skill's own power, and it
	// is the caster's side of the board answering ArcPower.
	//
	// ⚠️ **It is the only payment that scales with the size of the spend**, which
	// is the whole reason it exists. BonusPower is flat, so a skill that empties a
	// reserve of nine hundred for it pays exactly what a skill that spends twenty
	// pays — "spend all of it" written with a flat payment is a sentence with no
	// arithmetic in it. This one multiplies, so how much is left is a decision
	// every turn rather than a threshold crossed once.
	//
	// ⚠️ **It is not ArcPower and does not behave like it.** An arc is the charge
	// going off on the target: unaimed, unrolled, and not stopped by a guard. This
	// is the caster hitting harder, so it goes into the skill's power and is
	// therefore aimed, rolled against accuracy and dodge, divided by defence and
	// eaten by a shield exactly as the rest of the blow is. That is why it can be
	// caster-side while an arc may not be: it reads nothing about the board.
	//
	// ⚠️ **Caster-side only, and refused on `requires`.** The target's condition
	// already has a per-stack currency, and taking both would be the ceiling
	// charged twice with nothing downstream able to tell which half a figure came
	// from — the same refusal BonusPower and ArcPower already share.
	StackPower int
	// StackRestore is the health twin of StackPower: what one consumed stack
	// restores to the caster, in parts per thousand of the scaling stat, which is
	// the same unit Skill.Restores is written in.
	//
	// ⚠️ **It is what makes a reserve buy something other than a blow.** A skill
	// declaring `restores` alone pays out in full to a caster holding no fuel at
	// all — the condition is an amplifier, so failing it charges nothing and stops
	// nothing — and a heal that is free when the tank is empty is not paid for by
	// the tank. Written per stack instead, with the flat `restores` left at
	// nought, "no fuel, no heal" is arithmetic rather than a gate somebody has to
	// remember to write.
	//
	// ⚠️ **A condition may not carry both currencies.** Takes clamps the stacks a
	// spend removes against the ceiling the payment can buy, and there is one
	// Takes: a condition holding both a power rate and a health rate would have
	// two ceilings answering one question, and whichever was applied would be
	// wrong for the other half.
	StackRestore int
	// Applies are riders the condition pays out only when it holds, on the same
	// target it read. They are the condition's third currency: a bonus buys
	// damage, a restore buys health, and these buy a status the skill does not
	// otherwise inflict.
	//
	// ⚠️ **A target's condition only.** The caster's own reads the caster, so a
	// rider here would land on whoever cast it, which is what `self_applies`
	// already says plainly and without a condition in the way.
	//
	// They go out through the same inflict, the same roll and the same shield
	// filter as the skill's own `applies`, because a rider that survived a block
	// on a different rule from its neighbour would be a difference no reader
	// could find on either.
	Applies []Application
}

// AppliesOnHold reports whether the condition carries riders of its own.
func (c *Condition) AppliesOnHold() bool { return c != nil && len(c.Applies) > 0 }

// Arcs reports whether the condition discharges the status it consumes into
// damage of its own.
func (c *Condition) Arcs() bool { return c != nil && c.ArcPower > 0 }

// Scales reports whether the condition pays its caster per stack it spends.
func (c *Condition) Scales() bool { return c != nil && c.StackPower > 0 }

// ScalesRestore reports whether the condition pays its caster health per stack
// it spends.
//
// A second predicate rather than a widened Scales, because the two are read to
// decide which sentence to write and which figure to quote: a heal-only spender
// answering true to Scales would have the description promise that every stack
// adds to the blow, off a StackPower of nought.
func (c *Condition) ScalesRestore() bool { return c != nil && c.StackRestore > 0 }

// GatesCast reports whether the condition decides that the skill may be cast at
// all, rather than what the cast pays out.
func (c *Condition) GatesCast() bool { return c != nil && c.Gates }

// ChainsOn reports whether the consume travels to adjacent carriers.
func (c *Condition) ChainsOn() bool { return c != nil && c.Chains }

// Takes is how many stacks a consume removes from a target carrying the given
// number, and nought when this condition consumes nothing.
//
// The "all of them" default lives here rather than at the call site because two
// callers ask — the battle that spends the stacks and the pricing that values
// them — and a default written twice is a default that will disagree with itself.
//
// ⚠️ **It is asked once per STRIKE, not once per cast.** One blow spends one
// charge, so a skill that lands three times spends three — which is what makes a
// repeating strike and a pile of counters the same bet twice over, and what keeps
// a multi-strike conduit from being a single-strike one with better numbers.
func (c *Condition) Takes(held int) int {
	taken := 0
	switch {
	case c == nil || !c.Consume:
		return 0
	case c.ConsumeStacks <= 0 || c.ConsumeStacks > held:
		taken = held
	default:
		taken = c.ConsumeStacks
	}
	// ⚠️ **A scaling spend never takes more than it can pay for**, and that is a
	// rule about the STACKS rather than about the power.
	//
	// MaxSpendPower bounds what one cast may buy, and the obvious place to apply
	// it is the product — clamp the bonus and be done. That is the shape of the
	// bug this engine has already shipped twice: the blow would land paid for by
	// a pile the caster handed over and did not use, so a full reserve emptied
	// into a capped bonus would be worth less per stack than a small one, and the
	// rating would prefer to spend at exactly the wrong moment. Clamping here
	// instead means what is removed and what is bought are the same figure read
	// once, the leftovers stay in the tank, and "spend all of it" means "spend all
	// of it that this skill can use".
	if c.StackPower > 0 {
		if most := MaxSpendPower / c.StackPower; taken > most {
			taken = most
		}
	}
	// The same rule on the health currency, and the mirror is exact: it is the
	// STACKS that are clamped and not the health, for every word of the reason
	// above. A spend that handed over a pile and healed a capped figure would pay
	// most per stack at the shallowest depth, and a rating that could see that
	// would cash the tank in the moment it opened.
	//
	// Two clauses rather than one, because the two ceilings are different numbers
	// and a condition may hold only one of the two rates — see resolveCondition,
	// which refuses the pair rather than leaving this to pick between them.
	if c.StackRestore > 0 {
		if most := MaxSpendRestore / c.StackRestore; taken > most {
			taken = most
		}
	}
	return taken
}

// ReadsStatus reports whether the condition asks what the target is carrying.
// A condition that reads only health leaves Status empty.
func (c *Condition) ReadsStatus() bool { return c != nil && c.Status != "" }

// ReadsHealth reports whether the condition asks how hurt the target is.
func (c *Condition) ReadsHealth() bool { return c != nil && c.BelowHealth > 0 }

// Holds reports whether a target satisfies the condition.
//
// Every clause must hold, not any of them: a condition naming both a status and
// a threshold is "already burning *and* nearly dead", which is the only reading
// that lets a second clause narrow a skill rather than widen it.
func (c *Condition) Holds(against Target) bool {
	if c == nil {
		return false
	}
	if c.ReadsStatus() && against.Stacks < c.MinStacks {
		return false
	}
	if c.ReadsHealth() && !scale.AtOrBelowShare(against.Health, against.Maximum, c.BelowHealth) {
		return false
	}
	return true
}

// Gradient is how much harder a skill hits as its caster falls, declared by the
// one number an author has to choose: what it is worth at the very bottom.
//
// One field rather than a pair, because the top of the curve is not a choice —
// a caster at full health has nothing to be desperate about, so the gradient is
// worth nothing there by definition. An author picking a floor as well as a
// ceiling would be picking a threshold, and the threshold already exists.
type Gradient struct {
	// AtEmpty is the share added to the skill's power as the caster approaches
	// no health, in parts per thousand. A thousand is double power at the bottom.
	AtEmpty int
}

// Share returns what the gradient adds at a given health, in parts per thousand,
// and nought for a skill that declares none.
//
// Nil-safe like Condition.Holds, and for the same reason: the caller is asking
// what a skill does, and "it does not have one" is an answer rather than a state
// worth branching on at every call site.
func (g *Gradient) Share(health, maximum int64) int {
	if g == nil {
		return 0
	}
	return combat.Gradient(health, maximum, g.AtEmpty)
}

// Target is what a condition is allowed to know about the unit a skill is aimed
// at: the stacks it carries of the named status, and where its health sits.
//
// It is a struct rather than three parameters because two of the three are
// int64 health values that mean nothing apart, and a caller that swapped them
// would compile. The battle fills it in from the unit; a report fills it in from
// the numbers it is measuring against.
type Target struct {
	Stacks  int
	Health  int64
	Maximum int64
}

// Carrying is the target of a condition that only reads a status, which is what
// a report or a test measuring the status half wants to say.
func Carrying(stacks int) Target { return Target{Stacks: stacks} }

// Satisfying is the cheapest target the condition holds against: exactly the
// stacks it asks for, and health as low as it wants.
//
// It is what a preview and a report want — both are answering "what is this
// skill worth when it goes off", and neither has a real unit to ask. Health of
// nought out of one is not a unit anybody could meet, and that is the point: it
// is the *threshold satisfied*, written so it cannot be mistaken for a
// measurement of somebody.
func (c *Condition) Satisfying() Target {
	if c == nil {
		return Target{}
	}
	return Target{Stacks: c.MinStacks, Health: 0, Maximum: 1}
}

// Cleanse is the statuses a skill strips.
type Cleanse struct {
	Categories []status.Category
	Stacks     int
}

// Restriction is who a skill declares itself available to.
//
// Every list is an allowlist and every one is optional: a list that is absent
// restricts nothing, and a skill with no restriction at all is carried by
// whoever CanCarry allows. A list that is present and empty is a data error
// rather than "unrestricted" — an allowlist nobody satisfies is a mistake every
// time, and reading it as "no restriction" would turn that mistake into a skill
// silently available to everyone.
//
// # Why four of the five are plain strings
//
// Elements are parsed here because this package already knows what an element
// is. Archetypes, characters, species and origins are not: internal/core/cast
// declares all four, and cast imports this package, so a skill that named a cast
// type would be an import cycle. So they travel as the ids they were written
// with and are checked one layer up, by cast.ParseBook and cast.ParseArchetypes
// — exactly the way a skill's pattern and status names are checked by whoever
// holds those books rather than by whoever declares the skill.
//
// The layering has a consequence worth stating rather than working around:
// battle.Roster carries stats, skills, an affinity and a slot, and no archetype,
// no character identity and no species, because all three are resolved before a
// battle starts. So Elements is enforceable at battle load and the other four
// are not. They are authoring-time rules, and pushing any of them into the engine
// to "complete" the feature would put a fact into the replayable core that no
// replay needs. See CLAUDE.md, "What a restriction can enforce".
type Restriction struct {
	// Elements is the affinities allowed to carry the skill: a unit qualifies
	// by holding any one of them.
	//
	// Any rather than all, because a unit holds at most two elements and an
	// all-of rule of more than two could never be met. The list is what makes a
	// *neutral* skill restrictable at all — CanCarry lets every affinity carry
	// a neutral skill, so a neutral skill that should belong to two elements has
	// nowhere else to say so.
	Elements []element.Element
	// Archetypes is the role presets allowed to carry it, by id in the
	// archetype book.
	Archetypes []string
	// Characters is the characters allowed to carry it, by id in the cast book.
	// A list of one is a unique skill.
	Characters []string
	// Species is the kinds of creature allowed to carry it, by id in the
	// species book: a unit qualifies by being any one of them.
	//
	// Any rather than all, because a unit may be several things at once and a
	// skill about being a dragon has no opinion about what else the holder is.
	// This is the axis a body-bound skill wants: a lineage outlives the
	// character that first had it, so `dragon_rage` restricted to one character
	// said "only this one may carry it" when it meant "only a dragon may".
	Species []string
	// Origins is the works allowed to carry it, by id in the origin book: a
	// character qualifies by having been borrowed from any one of them.
	//
	// The broadest gate here, and the one that says a skill belongs to a
	// fiction rather than to a body or a fighting style. A work outlives every
	// character in it, which is the same argument Species makes about a
	// lineage: `rasengan` restricted to naruto.naruto would say "only this one
	// may carry it" when it means "only somebody from Naruto may", and would
	// have to be edited every time that work lends another character.
	//
	// It is a list rather than one id because a crossover is a thing that
	// happens, and because every other axis here is a list.
	Origins []string
}

// AllowsElement reports whether the element allowlist admits an affinity.
//
// The nil receiver is the unrestricted case and answers yes, so a caller never
// has to ask whether there is a restriction before asking what it says.
func (r *Restriction) AllowsElement(affinity element.Affinity) bool {
	if r == nil || len(r.Elements) == 0 {
		return true
	}
	for _, allowed := range r.Elements {
		if affinity.Has(allowed) {
			return true
		}
	}
	return false
}

// AllowsArchetype reports whether the preset allowlist admits an id.
func (r *Restriction) AllowsArchetype(id string) bool {
	return r == nil || len(r.Archetypes) == 0 || slices.Contains(r.Archetypes, id)
}

// AllowsCharacter reports whether the character allowlist admits an id.
func (r *Restriction) AllowsCharacter(id string) bool {
	return r == nil || len(r.Characters) == 0 || slices.Contains(r.Characters, id)
}

// AllowsSpecies reports whether the species allowlist admits a unit that is
// any of the given kinds.
//
// A unit with no species satisfies an unrestricted skill and fails a restricted
// one, which is the same answer an empty character id would get: the allowlist
// names what a holder must be, and being nothing is not one of them.
func (r *Restriction) AllowsSpecies(kinds []string) bool {
	if r == nil || len(r.Species) == 0 {
		return true
	}
	for _, allowed := range r.Species {
		if slices.Contains(kinds, allowed) {
			return true
		}
	}
	return false
}

// AllowsOrigin reports whether the origin allowlist admits the work a character
// was borrowed from.
//
// A character with no origin fails a restricted list, for AllowsSpecies's
// reason: the allowlist names where a holder must be from, and nowhere is not
// one of those places.
func (r *Restriction) AllowsOrigin(id string) bool {
	return r == nil || len(r.Origins) == 0 || slices.Contains(r.Origins, id)
}

// NamesCharacters reports whether the skill belongs to named characters, which
// is what makes it unusable in a preset shared by every character built from
// it.
func (r *Restriction) NamesCharacters() bool { return r != nil && len(r.Characters) > 0 }

// NamesSpecies reports whether the skill belongs to named kinds of creature,
// which makes it unusable in a preset for exactly the same reason: a preset says
// how a character fights and nothing about what it is, so every character built
// from one that is not of those kinds would be refused — and the refusal would
// land on whoever wrote the character rather than on whoever wrote the preset.
func (r *Restriction) NamesSpecies() bool { return r != nil && len(r.Species) > 0 }

// NamesOrigins reports whether the skill belongs to named works, which keeps it
// out of a preset for the third time and the same reason: a preset says how a
// character fights, so every character built from one that was borrowed from
// another work would be refused, and the refusal would land on whoever wrote
// the character.
func (r *Restriction) NamesOrigins() bool { return r != nil && len(r.Origins) > 0 }

// SpeciesNames is the species allowlist written out.
//
// It exists so that a caller can walk the list without reaching through the
// pointer: r.Species on a nil restriction is a nil dereference, while every
// method here answers for the unrestricted case. That trap cost a panic once.
func (r *Restriction) SpeciesNames() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.Species...)
}

// OriginNames is the origin allowlist written out, and reaches through the
// pointer safely for SpeciesNames's reason.
func (r *Restriction) OriginNames() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.Origins...)
}

// ElementNames is the element allowlist written out, which is what a refusal
// and a listing want.
func (r *Restriction) ElementNames() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.Elements))
	for _, allowed := range r.Elements {
		out = append(out, allowed.String())
	}
	return out
}

// Skill is one declared action.
type Skill struct {
	ID string
	// Name is the skill's authored display name, and this package knows nothing
	// about it beyond that it is text: not what language it is in, not how wide
	// it is on a screen, not whether anything shows it. Nothing in
	// internal/core reads it — no rule branches on it and no event carries it —
	// which is what lets it be here at all.
	//
	// # Why the name is on the declaration
	//
	// A skill's Vietnamese name used to be a compiled table in internal/i18n,
	// which meant a name could not be authored by the tool that authors the
	// skill. Putting it here rather than in a translations file beside the book
	// is the same judgement `Applies` and `Restrict` are here for: a skill and
	// its name are authored in one sitting, and a second file is a second thing
	// to keep in step — one that would go stale exactly when a skill was
	// renamed or removed.
	//
	// It is absent by default and absent is a real answer: a skill with no name
	// renders as its bare id, which is the rule a data id has always followed
	// when nothing has a name for it. internal/i18n prefers this over its own
	// table, so the compiled names remain the fallback for the skills that
	// shipped before this field existed. Being absent by default is also why no
	// golden moved when it arrived — see skillFile.
	Name string
	// Flavour is one clause saying what the skill *does*, in words rather than in
	// figures: "vung dây leo quật kẻ địch từ xa". It opens the sentence a
	// description is built into, and the numbers are appended to it.
	//
	// # Why it is authored where nothing else about a description is
	//
	// Everything else in a description is derived, on purpose, because an
	// authored figure drifts the moment the figure it describes moves. Nothing
	// derives "dây leo" from `vine_whip`: the name is the one fact about a skill
	// that only a person holds, and a generator reading ids would be guessing.
	//
	// ⚠️ **It may not contain a digit**, and ParseBook refuses one that does.
	// That is the whole of what keeps the guarantee: a clause with no number in
	// it cannot be made wrong by changing a number, so this is authored prose
	// that still cannot go stale. A skill that wants to say "twice over" says it
	// by having a bonus, and the sentence built around this clause reports it.
	//
	// It is absent by default, and absent is a real answer: a skill with no
	// flavour opens with the derived clause instead, which is what every skill
	// read like before this existed.
	Flavour string
	Element element.Element
	// Range is the hex distance the skill reaches, measured on the shared board.
	Range int
	// Pattern is the shape it covers, by name in the pattern book.
	Pattern string
	// Power is the skill's strength per strike, in parts per thousand.
	Power int
	// Strikes is how many times it lands, at least. A multi-strike skill is
	// expected to divide its power rather than repeat it at full strength.
	Strikes int
	// Repeat is the chance a strike is followed by another, in parts per
	// thousand, rolled again after each; MaxStrikes is where it stops. Together
	// they turn the count from a number into a distribution — mostly the floor,
	// occasionally a great deal more — and ExpectedStrikes is the mean every
	// caller outside the roll reads. Zero for every skill that shipped before
	// they existed, which is why adding them moved no golden.
	Repeat     int
	MaxStrikes int
	// Accuracy is its own chance to connect, before the caster's accuracy stat
	// and the target's dodge. A full thousand cannot miss and cannot be dodged.
	Accuracy int
	// Pierce is the share of the target's defence the skill ignores, in parts
	// per thousand. It is the answer to armour, which every other defence in
	// the game already has one of, and it is a ratio rather than a switch for
	// the reason combat.Pierced records.
	//
	// It reaches the skill's own strikes and nothing else. A status this skill
	// applies ticks against full defence, because a tick's damage is computed
	// once when the stack is applied and frozen for the stack's whole life — so
	// a pierced tick is worth several pierced hits, which is a larger effect
	// than an author setting a per-strike ratio is asking for.
	Pierce int
	// Unblockable says nothing stops this skill's strikes: neither a block
	// charge, which would otherwise cancel one whole, nor an absorbing pool,
	// which would otherwise eat the damage before health did.
	//
	// A switch rather than a share, and it is the only switch of its kind here on
	// purpose. Pierce is a ratio because defence is continuous and a ratio walks
	// an armoured unit's edge instead of jumping to the end of it; a guard is
	// **discrete** — a charge is spent or it is not, a pool is there or it is not
	// — so there is no middle for a ratio to express. "Half a block charge" is
	// not a quantity this engine has.
	//
	// ⚠️ It says nothing about defence. A skill that wants to ignore armour as
	// well says so with Pierce, and a skill that wants to ignore everything says
	// both — which is a great deal for one skill to be given, and the reason to
	// price it with a cooldown rather than to hand it out.
	Unblockable bool
	// Crit is the chance each of its strikes lands critically, in parts per
	// thousand. What a critical strike is worth is one game-wide constant on
	// combat.Rules, so this is the whole of what makes one skill crit more than
	// another — and it belongs to the skill rather than to whoever casts it,
	// because progression.Values has no seventh kind to hold it.
	//
	// Every shipped skill declares nought.
	Crit    int
	Scaling Scaling
	// Applies are the statuses inflicted on each target it hits.
	Applies []Application
	// SelfApplies are the statuses the caster gains, which is how a shield or a
	// self buff is written.
	SelfApplies []Application
	// Requires is the amplifier read against the target, if the skill has one.
	Requires *Condition
	// SelfRequires is the amplifier read against the caster.
	//
	// It is what a skill that spends something of its own needs, and until it
	// existed such a skill was unwritable: Requires reads the target, so "hits
	// harder while I am furied" and "hits harder while I am cornered" had no
	// spelling at all. The two are the same shape asked of two different units,
	// which is why they are the same type.
	//
	// Read once per use rather than once per target -- see Battle.spend. A
	// condition that consumed per target would pay for a splash three times over
	// and a single-target skill once, for a difference written on neither.
	SelfRequires *Condition
	// SelfGradient is how much harder the skill hits the further its caster has
	// fallen, and it is the smooth twin of SelfRequires rather than a second
	// copy of it.
	//
	// A condition answers yes or no, so it can only ever say "at or below this
	// line, take this much". There is no line here and no yes or no: every point
	// of health lost is worth the same as the last. That is why it is a share
	// multiplied into the power rather than a bonus added to it, why the
	// arithmetic lives in combat.Gradient, and why it is a type of its own
	// instead of a fourth field on Condition that the other three would have to
	// leave empty.
	//
	// Read once per use, exactly like SelfRequires and for the same reason.
	SelfGradient *Gradient
	// Strips is the cleanse or dispel the skill performs, if any.
	Strips *Cleanse
	// Restrict is who may carry the skill, or nil when anybody may.
	Restrict *Restriction
	// Summons is the unit the skill puts on the board, or nil for a skill that
	// only acts on units already standing there.
	Summons *Summon
	// Restores is how much health the skill gives its targets, in parts per
	// thousand of the caster's scaling stat. It does not pass through defence:
	// see combat.Rules.Restore for why.
	Restores int
	// Drains is the share of the damage actually dealt that comes back to the
	// caster, in parts per thousand. It reads damage *dealt* rather than damage
	// rolled, so a strike that missed or was blocked drains nothing.
	Drains int
	// Cost is the health the caster pays to use the skill, in parts per thousand
	// of its MAXIMUM health.
	//
	// # Paid up front, and paid whether or not anything lands
	//
	// That is what makes it a cost rather than a recoil. A share taken out of the
	// damage dealt would be free on a turn the skill missed, and a skill that is
	// free when it fails is a skill with no decision in it — the whole of what
	// this field is for is that using it is worse than not using it unless the
	// blow is worth the health.
	//
	// # A share of maximum rather than current, and never lethal
	//
	// Of maximum, so the price is the same on the first turn and the last: a
	// share of *current* health gets cheaper exactly as it gets more dangerous,
	// which prices a desperate cast lowest. And the caster is left standing on at
	// least one point — see Battle.spendHealth. A skill that could kill its own
	// user is a different design question, and one nothing in the rating is built
	// to answer: Suggest prices what a turn is worth, not whether there is a
	// next one.
	//
	// ⚠️ It is refused on a skill that throws no strike, the way Pierce is. A
	// price with nothing bought is a skill an author wrote by accident.
	Cost int
	// Cooldown is how many of the caster's own turns must pass before it can be
	// used again. Counting the caster's turns rather than cycles means a fast
	// unit really does get its skill back sooner, in step with everything else
	// that is timed.
	Cooldown int
	Target   Side
}

// TotalPower is the skill's power across every strike, which is the figure to
// compare two skills of different strike counts by.
func (s Skill) TotalPower() int { return s.Power * s.StrikeCount() }

// StrikeCount treats an unset count as one.
func (s Skill) StrikeCount() int {
	if s.Strikes < 1 {
		return 1
	}
	return s.Strikes
}

// Repeats reports whether the skill's strike count is rolled rather than fixed.
func (s Skill) Repeats() bool { return s.Repeat > 0 && s.MaxStrikes > s.StrikeCount() }

// ExpectedStrikes is how many times the skill lands on average, in parts per
// thousand, and it is the figure every caller outside the roll should read.
//
// ⚠️ **Neither end of the range is usable and that is the point of the field.**
// The floor prices a repeating skill as though the tail never happened, and a
// rating reading it would never pick one; the ceiling prices every cast as the
// best cast anybody ever had, and a rating reading that would pick nothing else.
// A repeating strike is a *distribution*, so what it is worth is its mean.
//
// The arithmetic is combat.Hit.ExpectedStrikes and this is the same sum written
// where a skill can be asked without building a hit — a description has no
// caster and no target, and it still has to quote a figure.
func (s Skill) ExpectedStrikes() int {
	count := s.StrikeCount()
	total := count * scale.Base
	if !s.Repeats() {
		return total
	}
	odds := scale.Base
	for i := count; i < s.MaxStrikes; i++ {
		odds = odds * s.Repeat / scale.Base
		total += odds
	}
	return total
}

// PowerAgainst returns the power the skill lands with, given how many stacks of
// its required status the target carries.
func (s Skill) PowerAgainst(against Target) int {
	if !s.Amplified(against) {
		return s.Power
	}
	return s.Power + s.Requires.BonusPower
}

// Amplified reports whether the condition holds against a given target.
func (s Skill) Amplified(against Target) bool { return s.Requires.Holds(against) }

// SelfAmplified reports whether the caster's own condition holds.
func (s Skill) SelfAmplified(caster Target) bool { return s.SelfRequires.Holds(caster) }

// SelfBonus is what the caster's own condition adds to the skill's power, and
// nought when it does not hold or is not declared.
//
// A separate function from PowerAgainst rather than folded into it, because the
// two are read at different moments: the target's condition is read per target,
// and this one is read once for the whole use. Folding them would make the
// second follow the first around the shape.
// ⚠️ **A per-stack payment is read here and nowhere else.** It is what the spend
// buys, so it belongs to the same reading as the flat bonus rather than to the
// consume — and it is multiplied by what the consume will actually take, which is
// Condition.Takes of what the caster is holding right now. That is the same
// figure Battle.spend removes, from the same function, so the power the blow
// lands with and the stacks that paid for it cannot disagree.
func (s Skill) SelfBonus(caster Target) int {
	if !s.SelfAmplified(caster) {
		return 0
	}
	if s.SelfRequires.Scales() {
		return s.SelfRequires.StackPower * s.SelfRequires.Takes(caster.Stacks)
	}
	return s.SelfRequires.BonusPower
}

// SelfCeiling is the most the caster's own condition can add to this skill's
// power, and it is the figure any bound on the swing must be read against.
//
// ⚠️ **Not SelfBonus at Satisfying, and the difference is new.** Satisfying is
// the CHEAPEST target a condition holds against, and for a flat bonus the
// cheapest case and the dearest one are the same number — so one reading served
// both until stack_power existed. A per-stack payment is worth more the deeper
// the reserve, so a bound taken at the threshold would be a bound on the
// smallest case the skill can produce.
//
// The ceiling comes from Takes rather than from the status book, because Takes
// is where the ceiling actually bites: a scaling spend refuses stacks past what
// MaxSpendPower will pay for, whatever the book allows anybody to hold.
func (s Skill) SelfCeiling() int {
	if s.SelfRequires == nil {
		return 0
	}
	if s.SelfRequires.Scales() {
		return s.SelfRequires.StackPower * s.SelfRequires.Takes(MaxSpendPower)
	}
	return s.SelfBonus(s.SelfRequires.Satisfying())
}

// SelfRestore is the health the caster's own condition buys with the stacks it
// spends, in parts per thousand of the scaling stat, and nought when the
// condition does not hold or is not declared.
//
// The health twin of SelfBonus, and read at the same moment for the same reason:
// once per use, off the same Takes the spend removes, so the heal that lands and
// the stacks that paid for it cannot disagree.
//
// ⚠️ **Nought when the condition does not hold, and that is the whole feature.**
// A skill's flat `restores` pays out whatever the caster is carrying, because a
// condition is an amplifier and a caster who fails it is charged nothing and
// stopped by nothing. Written here instead, an empty tank buys an empty heal
// without a single gate anywhere.
func (s Skill) SelfRestore(caster Target) int {
	if !s.SelfAmplified(caster) {
		return 0
	}
	return s.SelfRequires.StackRestore * s.SelfRequires.Takes(caster.Stacks)
}

// SelfRestoreCeiling is the most the caster's own condition can restore in one
// cast, and it is SelfCeiling's twin: read through Takes, because Takes is where
// the ceiling bites, whatever depth the status book allows anybody to bank to.
func (s Skill) SelfRestoreCeiling() int {
	if s.SelfRequires == nil || !s.SelfRequires.ScalesRestore() {
		return 0
	}
	return s.SelfRequires.StackRestore * s.SelfRequires.Takes(MaxSpendRestore)
}

// SelfScale is the share the caster's own wounds add to the skill's power, in
// parts per thousand, and nought for a skill with no gradient.
//
// It takes the two numbers rather than a Target because a Target carries a stack
// count as well, and a gradient has nothing to do with what anybody is carrying.
// Handing it one would say it might.
func (s Skill) SelfScale(health, maximum int64) int {
	return s.SelfGradient.Share(health, maximum)
}

// Guaranteed reports whether the skill cannot miss.
func (s Skill) Guaranteed() bool { return s.Accuracy >= scale.Base }

// Book is the declared skills.
type Book struct {
	skills []Skill
	byID   map[string]Skill
}

// skillFile is the shape a skill is written in, and therefore the shape it is
// read in: Skill.MarshalJSON builds one of these rather than carrying its own
// tags, so the writer cannot describe a field the parser does not read.
//
// The field order is the order the keys are written in — the nine core fields
// first, in the order the data files have always listed them, then the optional
// blocks. Every core field is written even at its zero value, so a skill's
// first line is always complete; each optional block is omitted when it is
// absent, so a skill that declares none reads exactly as it did before any of
// them existed.
type skillFile struct {
	ID string `json:"id"`
	// Name is written only when there is one, which is what kept every golden
	// still when it was added: the shipped book declares none, so the shipped
	// file round-trips byte for byte and the tables measured from it do not move.
	// It sits beside the id because that is where it reads — a skill and the name
	// it is called by, then the numbers.
	Name string `json:"name,omitempty"`
	// Flavour sits beside the name for the same reason: both are the words a
	// skill is read in, and both are omitted when absent so a book that declares
	// neither round-trips to the bytes it was authored as.
	Flavour    string `json:"flavour,omitempty"`
	Element    string `json:"element"`
	Range      int    `json:"range"`
	Pattern    string `json:"pattern"`
	Power      int    `json:"power"`
	Strikes    int    `json:"strikes"`
	Repeat     int    `json:"repeat_chance,omitempty"`
	MaxStrikes int    `json:"max_strikes,omitempty"`
	Accuracy   int    `json:"accuracy"`
	// Pierce is written only when there is some, like the two healing figures
	// below it: no shipped skill pierces, so the shipped book round-trips byte
	// for byte and the tables measured from it did not move when it arrived.
	Pierce int `json:"pierce,omitempty"`
	// Unblockable is written only when set, for the reason Pierce is written
	// only when there is some: the shipped book round-trips byte for byte and no
	// golden moved when the field arrived.
	Unblockable bool `json:"unblockable,omitempty"`
	// Crit is written only when there is some, for the same reason Pierce is:
	// no shipped skill crits, so the shipped book round-trips byte for byte.
	Crit         int               `json:"crit,omitempty"`
	Restores     int               `json:"restores,omitempty"`
	Drains       int               `json:"drains,omitempty"`
	Cost         int               `json:"cost,omitempty"`
	Cooldown     int               `json:"cooldown"`
	Target       string            `json:"target"`
	Restrict     *restrictFile     `json:"restrict,omitempty"`
	Scaling      *scalingFile      `json:"scaling,omitempty"`
	Applies      []applicationFile `json:"applies,omitempty"`
	SelfApplies  []applicationFile `json:"self_applies,omitempty"`
	Requires     *conditionFile    `json:"requires,omitempty"`
	SelfRequires *conditionFile    `json:"self_requires,omitempty"`
	SelfGradient *gradientFile     `json:"self_gradient,omitempty"`
	Strips       *cleanseFile      `json:"strips,omitempty"`
	Summons      *summonFile       `json:"summons,omitempty"`
}

type summonFile struct {
	Count       int                 `json:"count,omitempty"`
	Name        string              `json:"name,omitempty"`
	Share       int                 `json:"share,omitempty"`
	ShareOfBase int                 `json:"share_of_base,omitempty"`
	Stats       *progression.Values `json:"stats,omitempty"`
	Affinity    string              `json:"element,omitempty"`
	Skills      []string            `json:"skills"`
	Lasts       int                 `json:"lasts,omitempty"`
	Bound       bool                `json:"bound,omitempty"`
}

type scalingFile struct {
	Stat   string `json:"stat"`
	Source string `json:"source"`
}

type conditionFile struct {
	Status        string            `json:"status,omitempty"`
	MinStacks     int               `json:"min_stacks,omitempty"`
	Gates         bool              `json:"gates,omitempty"`
	BelowHealth   int               `json:"below_health,omitempty"`
	BonusPower    int               `json:"bonus_power"`
	Consume       bool              `json:"consume,omitempty"`
	ConsumeStacks int               `json:"consume_stacks,omitempty"`
	Chains        bool              `json:"chains,omitempty"`
	ArcPower      int               `json:"arc_power,omitempty"`
	StackPower    int               `json:"stack_power,omitempty"`
	StackRestore  int               `json:"stack_restore,omitempty"`
	Applies       []applicationFile `json:"applies,omitempty"`
}

// gradientFile is its own shape rather than a field on conditionFile, because a
// conditionFile with nothing but an at_empty would be a condition that asks
// nothing — which the parser refuses, and rightly.
type gradientFile struct {
	AtEmpty int `json:"at_empty"`
}

type cleanseFile struct {
	Categories []string `json:"categories"`
	Stacks     int      `json:"stacks"`
}

type restrictFile struct {
	Elements   []string `json:"elements,omitempty"`
	Archetypes []string `json:"archetypes,omitempty"`
	Characters []string `json:"characters,omitempty"`
	Species    []string `json:"species,omitempty"`
	Origins    []string `json:"origins,omitempty"`
}

type applicationFile struct {
	Status string `json:"status"`
	Chance int    `json:"chance"`
	Stacks int    `json:"stacks"`
}

type bookFile struct {
	Skills []skillFile `json:"skills"`
}

// marshalFile is the shape Book.Marshal writes. It holds Skill rather than
// skillFile because Skill.MarshalJSON already turns one into the other.
type marshalFile struct {
	Skills []Skill `json:"skills"`
}

// DefaultScaling is what a skill scales off when it says nothing: the caster's
// attack as it stands, which is what almost every skill wants.
//
// It is named once because three places need it — resolve, which fills it in,
// file, which leaves it out again, and an authoring tool building a skill from
// answers — and a second copy of the pair would let a written file disagree with
// the one it was read from.
//
// It is exported because the zero Scaling is *not* this: progression.HP is the
// zero stat, and a skill scaling off health is refused, so a Skill built in Go
// without setting this field is refused the moment it is written. That refusal
// is loud on purpose. Silently substituting the default here would make the
// field's zero value mean two different things depending on where the skill came
// from.
func DefaultScaling() Scaling {
	return Scaling{Stat: progression.Attack, Source: combat.CurrentStat}
}

// file is a resolved skill as the declaration it came from.
//
// Writing goes through the parse shape rather than through tags on Skill, which
// is what makes the write lossless by construction: the only fields that can be
// written are the fields the parser reads, and a field added to one is a
// compile error in the other until it is added there too.
//
// Every optional block is written when it is present and left out when it is
// not, and the one derived default — the scaling — is left out when it is the
// default, so a file that declared none reads back exactly as it was authored.
func (s Skill) file() skillFile {
	out := skillFile{
		ID: s.ID, Name: s.Name, Flavour: s.Flavour,
		Element: s.Element.String(), Range: s.Range, Pattern: s.Pattern,
		Power: s.Power, Strikes: s.Strikes, Repeat: s.Repeat, MaxStrikes: s.MaxStrikes,
		Accuracy: s.Accuracy,
		Pierce:   s.Pierce, Unblockable: s.Unblockable,
		Crit: s.Crit, Restores: s.Restores, Drains: s.Drains, Cost: s.Cost,
		Cooldown: s.Cooldown, Target: s.Target.String(),
		Applies: applicationFiles(s.Applies), SelfApplies: applicationFiles(s.SelfApplies),
	}
	if s.Restrict != nil {
		out.Restrict = &restrictFile{
			Elements:   s.Restrict.ElementNames(),
			Archetypes: append([]string(nil), s.Restrict.Archetypes...),
			Characters: append([]string(nil), s.Restrict.Characters...),
			Species:    append([]string(nil), s.Restrict.Species...),
			Origins:    append([]string(nil), s.Restrict.Origins...),
		}
	}
	if s.Scaling != DefaultScaling() {
		out.Scaling = &scalingFile{
			Stat: s.Scaling.Stat.String(), Source: s.Scaling.Source.String(),
		}
	}
	if s.SelfRequires != nil {
		out.SelfRequires = &conditionFile{
			Status: s.SelfRequires.Status, MinStacks: s.SelfRequires.MinStacks,
			Gates:       s.SelfRequires.Gates,
			BelowHealth: s.SelfRequires.BelowHealth,
			BonusPower:  s.SelfRequires.BonusPower, Consume: s.SelfRequires.Consume,
			ConsumeStacks: s.SelfRequires.ConsumeStacks, Chains: s.SelfRequires.Chains,
			ArcPower: s.SelfRequires.ArcPower, StackPower: s.SelfRequires.StackPower,
			Applies:      applicationFiles(s.SelfRequires.Applies),
			StackRestore: s.SelfRequires.StackRestore,
		}
	}
	if s.SelfGradient != nil {
		out.SelfGradient = &gradientFile{AtEmpty: s.SelfGradient.AtEmpty}
	}
	if s.Requires != nil {
		out.Requires = &conditionFile{
			Status: s.Requires.Status, MinStacks: s.Requires.MinStacks,
			Gates:       s.Requires.Gates,
			BelowHealth: s.Requires.BelowHealth,
			BonusPower:  s.Requires.BonusPower, Consume: s.Requires.Consume,
			ConsumeStacks: s.Requires.ConsumeStacks, Chains: s.Requires.Chains,
			ArcPower: s.Requires.ArcPower, StackPower: s.Requires.StackPower,
			Applies:      applicationFiles(s.Requires.Applies),
			StackRestore: s.Requires.StackRestore,
		}
	}
	if s.Strips != nil {
		categories := make([]string, 0, len(s.Strips.Categories))
		for _, category := range s.Strips.Categories {
			categories = append(categories, category.String())
		}
		out.Strips = &cleanseFile{Categories: categories, Stacks: s.Strips.Stacks}
	}
	if s.Summons.Summons() {
		out.Summons = &summonFile{
			Count: s.Summons.Count, Name: s.Summons.Name,
			Share: s.Summons.Share, ShareOfBase: s.Summons.ShareOfBase,
			Stats: s.Summons.Stats, Affinity: s.Summons.Affinity,
			Skills: append([]string(nil), s.Summons.Skills...),
			Lasts:  s.Summons.Lasts, Bound: s.Summons.Bound,
		}
	}
	return out
}

func applicationFiles(applications []Application) []applicationFile {
	if len(applications) == 0 {
		return nil
	}
	out := make([]applicationFile, 0, len(applications))
	for _, application := range applications {
		out = append(out, applicationFile{
			Status: application.Status, Chance: application.Chance, Stacks: application.Stacks,
		})
	}
	return out
}

// MarshalJSON writes a skill as the declaration a parse would read back.
func (s Skill) MarshalJSON() ([]byte, error) { return json.Marshal(s.file()) }

// Deps are the books a skill's declarations are checked against. Validating here
// rather than at use is the whole point: a skill naming a shape or a status that
// does not exist is a data error, and a data error should stop the load.
type Deps struct {
	Patterns *pattern.Book
	Statuses *status.Book
}

// ParseBook reads a skill declaration and checks every name it uses. It never
// touches the filesystem.
func ParseBook(raw []byte, deps Deps) (*Book, error) {
	if deps.Patterns == nil || deps.Statuses == nil {
		return nil, fmt.Errorf("skills cannot be validated without the pattern and status books")
	}
	var file bookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode skill book: %w", err)
	}
	book := &Book{byID: make(map[string]Skill, len(file.Skills))}
	for _, declared := range file.Skills {
		built, err := resolve(declared, deps)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[built.ID]; clash {
			return nil, fmt.Errorf("skill %q is declared twice", built.ID)
		}
		book.byID[built.ID] = built
		book.skills = append(book.skills, built)
	}
	if len(book.skills) == 0 {
		return nil, fmt.Errorf("the skill book is empty")
	}
	// After the loop, because a summon names skills that may be declared below
	// it: the rule is about the finished book rather than about one entry, the
	// same division that keeps a status name out of this file's own reach.
	for _, declared := range book.skills {
		if err := WhySummonsCannotRecurse(book, declared); err != nil {
			return nil, err
		}
	}
	return book, nil
}

// maxRange is how deep into the far side a skill can be pointed, so it cannot
// declare a reach that means nothing.
//
// It is the number of columns in one formation, because reach is counted in
// **ranks** rather than in cells: a range of three already clears the enemy's
// whole half, and a four meant exactly what a three meant. It used to be the
// board's longest diagonal — five — which was the right bound while a range was
// a distance and is a lie now.
const maxRange = hex.FormationCols

// MaxSpendPower is the most one spend of a counter may add to a skill's own
// power, in parts per thousand of the caster's scaling stat.
//
// It exists because Condition.StackPower is the only authored power in this
// package that is multiplied before it is used, and the multiplier is a stack
// count capped at nine hundred and ninety nine. Every other ceiling here bounds
// a term an author can read at a glance; this one bounds a product the author
// never writes down.
//
// Applied by Condition.Takes rather than by clamping the bonus, so what a spend
// removes and what it buys stay one figure — read that comment before moving it.
//
// Four times the scaling stat, which is a little over twice the heaviest single
// skill in the shipped book. A spend is meant to be the biggest thing a turn can
// do and not a different game: at this ceiling emptying a full reserve is worth
// about two ordinary casts, so hoarding stays a real decision and never becomes
// the only one.
const MaxSpendPower = 4 * scale.Base

// MaxSpendRestore is the most one spend of a counter may restore to its caster,
// in parts per thousand of the caster's scaling stat.
//
// MaxSpendPower's twin, and it exists for the same reason: Condition.StackRestore
// is multiplied by a stack count before it is used, so it bounds a product the
// author never writes down. Applied by Condition.Takes on the STACKS, exactly as
// its twin is — read that comment before moving this one.
//
// Twice the scaling stat, derived from the shipped book rather than picked: the
// heaviest restore anybody carries is `synthesis` at 900, so this is a little
// over twice the biggest heal in the game. At the golden's reference 800 attack
// it caps one spend at 1600 health, which sits just under the 1611 damage
// `tide_break` deals for a spend of the same size — so emptying a reserve into
// health and emptying it into a blow are worth about the same turn, which is what
// makes the choice between them a decision rather than an answer.
const MaxSpendRestore = 2 * scale.Base

func resolve(declared skillFile, deps Deps) (Skill, error) {
	if declared.ID == "" {
		return Skill{}, fmt.Errorf("a skill needs an id")
	}
	fail := func(format string, args ...any) (Skill, error) {
		return Skill{}, fmt.Errorf("skill %q: "+format, append([]any{declared.ID}, args...)...)
	}

	affinity, err := element.Parse(declared.Element)
	if err != nil {
		return fail("%w", err)
	}
	target, err := ParseSide(declared.Target)
	if err != nil {
		return fail("%w", err)
	}
	shape, err := deps.Patterns.Lookup(declared.Pattern)
	if err != nil {
		return fail("%w", err)
	}

	// The order matters: a negative figure should say so rather than fall into
	// the broader "does nothing" complaint, which would send an author looking
	// in the wrong place.
	switch {
	case declared.Power < 0:
		return fail("has power %d, want zero or more", declared.Power)
	// A caster's own per-stack restore is a body of work like any other, and it is
	// the one that carries no flat figure anywhere on the skill: the whole heal
	// lives on the condition, so a skill spending a reserve to patch itself up
	// reads as `restores: 0` here and would have been refused as a wasted turn.
	case declared.Power == 0 && len(declared.Applies) == 0 && len(declared.SelfApplies) == 0 &&
		declared.Strips == nil && declared.Restores == 0 && declared.Summons == nil &&
		(declared.SelfRequires == nil || declared.SelfRequires.StackRestore == 0):
		return fail("has no power and does nothing else, so it would be a wasted turn")
	case declared.Restores < 0:
		return fail("restores %d, want zero or more", declared.Restores)
	case declared.Drains < 0:
		return fail("drains %d, want zero or more", declared.Drains)
	case declared.Drains > scale.Base:
		return fail("drains %d, more than the damage it deals", declared.Drains)
	case declared.Drains > 0 && declared.Power == 0:
		return fail("drains from damage it never deals")
	case declared.Strikes < 0:
		return fail("has %d strikes, want zero or more", declared.Strikes)
	case declared.Repeat < 0 || declared.Repeat > scale.Base:
		return fail("repeats at %d, want a share in parts per thousand", declared.Repeat)
	// A repeat with no ceiling is a skill that can take a whole battle in one
	// turn, however rarely, and "however rarely" is not a bound.
	case declared.Repeat > 0 && declared.MaxStrikes <= 0:
		return fail("repeats at %d with no max_strikes, so its tail is unbounded", declared.Repeat)
	case declared.MaxStrikes < 0:
		return fail("caps at %d strikes, want zero or more", declared.MaxStrikes)
	case declared.MaxStrikes > 0 && declared.Repeat == 0:
		return fail("caps at %d strikes but never repeats, so the cap bounds nothing", declared.MaxStrikes)
	case declared.Repeat > 0 && declared.Power == 0:
		return fail("repeats strikes it never deals damage with")
	// The floor has to be under the ceiling or the roll is a fixed count wearing
	// a distribution's clothes, and every figure derived from it would be a mean
	// of one outcome.
	case declared.MaxStrikes > 0 && declared.MaxStrikes <= max(declared.Strikes, 1):
		return fail("lands %d times and caps at %d, so it never repeats at all",
			max(declared.Strikes, 1), declared.MaxStrikes)
	case declared.Accuracy < 0 || declared.Accuracy > scale.Base:
		return fail("has accuracy %d, want a share in parts per thousand", declared.Accuracy)
	case declared.Power > 0 && declared.Accuracy == 0:
		return fail("deals damage but can never connect")
	case declared.Pierce < 0 || declared.Pierce > scale.Base:
		return fail("pierces %d, want a share in parts per thousand", declared.Pierce)
	case declared.Pierce > 0 && declared.Power == 0:
		return fail("pierces defence it never attacks through")
	case declared.Unblockable && declared.Power == 0:
		return fail("declares itself unblockable and throws no strike for anything to stop")
	case declared.Cost < 0 || declared.Cost >= scale.Base:
		return fail("costs %d of its caster's health, want a share under the whole of it", declared.Cost)
	case declared.Cost > 0 && declared.Power == 0 && declared.Restores == 0 &&
		len(declared.Applies) == 0 && len(declared.SelfApplies) == 0 &&
		declared.Strips == nil && declared.Summons == nil:
		// ⚠️ **This asked for a strike until the book had more than one thing to
		// buy.** When it was written a skill spent its power and nothing else, so
		// "no strike" and "nothing bought" were the same sentence. A skill now buys
		// health, statuses, a cleanse and a summon as well, and a heal that pays
		// its caster's own health for somebody else's is a trade the old wording
		// refused for a reason that had stopped being true. What is refused is a
		// cost that buys NOTHING, which is what the reason always was.
		return fail("costs its caster health and buys nothing with it")
	case declared.Crit < 0 || declared.Crit > scale.Base:
		return fail("crits %d, want a share in parts per thousand", declared.Crit)
	// Not tidiness: turn.go's power <= 0 branch never reaches combat.Roll, so a
	// crit chance on a skill that deals no damage is data nothing would ever
	// read.
	case declared.Crit > 0 && declared.Power == 0:
		return fail("crits on damage it never deals")
	case declared.Restores > 0 && target == Enemy:
		return fail("restores health to the enemy, which nobody means")
	// The same sentence one currency along. A caster's own condition pays its own
	// caster, so the aim is the caster's business rather than the payment's -- but
	// an enemy-aimed skill reaches resolveAgainst for every cell it covers, and a
	// heal filed there would be read as a heal for whoever is standing in them.
	case declared.SelfRequires != nil && declared.SelfRequires.StackRestore > 0 && target == Enemy:
		return fail("restores health per stack on a skill aimed at the enemy, which nobody means")
	case declared.Cooldown < 0:
		return fail("has cooldown %d, want zero or more", declared.Cooldown)
	}

	// A self-targeted skill has nowhere to reach and no shape to spread over.
	if target == Self {
		if declared.Range != 0 {
			return fail("targets itself but declares a range of %d", declared.Range)
		}
		if shape.MaxTargets() != 1 {
			return fail("targets itself but uses the %q shape, which covers %d cells",
				shape.Name, shape.MaxTargets())
		}
	} else if declared.Range < 1 || declared.Range > maxRange {
		return fail("has range %d, want between 1 and %d", declared.Range, maxRange)
	}

	scaling := DefaultScaling()
	if declared.Scaling != nil {
		var err error
		if scaling, err = ParseScaling(declared.Scaling.Stat, declared.Scaling.Source); err != nil {
			return fail("%w", err)
		}
	}

	// The name is text and this package has no opinion about it beyond that: it
	// is trimmed, so that a name of nothing but spaces is the absent answer
	// rather than a name made of spaces, and it is not measured, not checked
	// against a character set and not compared with the id.
	name := strings.TrimSpace(declared.Name)

	// A flavour clause carrying a figure is the one way authored prose can go
	// stale, so it is refused rather than trusted: every number in a description
	// is derived, and a clause saying "gấp đôi" would outlive the bonus that made
	// it true. The check is for digits rather than for a percent sign because
	// "110" and "gấp 2" are the same mistake wearing different clothes.
	flavour := strings.TrimSpace(declared.Flavour)
	if index := strings.IndexFunc(flavour, unicode.IsDigit); index >= 0 {
		return fail("has a flavour clause carrying the figure %q; every number in a description is derived, and an authored one would outlive what it describes",
			flavour[index:index+1])
	}

	applies, err := resolveApplications(declared.ID, "applies", declared.Applies, deps)
	if err != nil {
		return Skill{}, err
	}
	selfApplies, err := resolveApplications(declared.ID, "self_applies", declared.SelfApplies, deps)
	if err != nil {
		return Skill{}, err
	}

	requires, err := resolveCondition(declared.ID, "requires", declared.Requires, deps)
	if err != nil {
		return Skill{}, err
	}
	// A conduit rides on its own strikes, and turn.go never rolls a strike of no
	// power — so a skill that deals none would never reach the discharge it
	// declares.
	if requires.Arcs() && declared.Power <= 0 {
		return Skill{}, fmt.Errorf("skill %q requires: arcs for %d but the skill deals no damage, so it never rolls a strike and never discharges",
			declared.ID, requires.ArcPower)
	}
	selfRequires, err := resolveCondition(declared.ID, "self_requires", declared.SelfRequires, deps)
	if err != nil {
		return Skill{}, err
	}
	// A skill aimed at its caster never reaches resolveAgainst, so a bonus power
	// on one lands nowhere. Refused rather than accepted and ignored, which is
	// the same reason a status may not carry a health term.
	if selfRequires != nil && declared.Target == Self.String() && selfRequires.BonusPower > 0 {
		return Skill{}, fmt.Errorf("skill %q: self_requires adds power to a skill aimed at itself, which deals none",
			declared.ID)
	}
	selfGradient, err := resolveGradient(declared, selfRequires)
	if err != nil {
		return Skill{}, err
	}

	restrict, err := resolveRestriction(declared.ID, declared.Restrict)
	if err != nil {
		return Skill{}, err
	}

	var strips *Cleanse
	if declared.Strips != nil {
		if len(declared.Strips.Categories) == 0 {
			return fail("strips nothing, because it names no categories")
		}
		if declared.Strips.Stacks < 1 {
			return fail("strips %d stacks, want at least 1", declared.Strips.Stacks)
		}
		categories := make([]status.Category, 0, len(declared.Strips.Categories))
		seen := make(map[status.Category]bool, len(declared.Strips.Categories))
		for _, name := range declared.Strips.Categories {
			category, err := status.ParseCategory(name)
			if err != nil {
				return fail("strips: %w", err)
			}
			if seen[category] {
				return fail("strips the %s category twice", category)
			}
			seen[category] = true
			categories = append(categories, category)
		}
		strips = &Cleanse{Categories: categories, Stacks: declared.Strips.Stacks}
	}

	summons, err := resolveSummon(declared.Summons, func(format string, args ...any) error {
		return fmt.Errorf("skill %q: "+format, append([]any{declared.ID}, args...)...)
	})
	if err != nil {
		return Skill{}, err
	}

	return Skill{
		ID: declared.ID, Name: name, Flavour: flavour,
		Element: affinity, Range: declared.Range, Pattern: shape.Name,
		Power: declared.Power, Strikes: declared.Strikes,
		Repeat: declared.Repeat, MaxStrikes: declared.MaxStrikes,
		Accuracy: declared.Accuracy,
		Pierce:   declared.Pierce, Unblockable: declared.Unblockable,
		Crit:    declared.Crit,
		Scaling: scaling, Applies: applies, SelfApplies: selfApplies,
		Requires: requires, SelfRequires: selfRequires, SelfGradient: selfGradient,
		Strips: strips, Restrict: restrict,
		Summons:  summons,
		Restores: declared.Restores, Drains: declared.Drains, Cost: declared.Cost,
		Cooldown: declared.Cooldown, Target: target,
	}, nil
}

// resolveRestriction checks the half of a restriction this package can see.
//
// Element names are real, no list is present-but-empty, no entry is blank and
// no entry is repeated. What it deliberately does not check is whether an
// archetype, a character, a species or an origin id exists, because the books
// that declare those are one layer up — see Restriction. cast.ParseArchetypes and cast.ParseBook make
// that check, which is the same division as a skill's pattern and status names.
func resolveRestriction(skillID string, declared *restrictFile) (*Restriction, error) {
	if declared == nil {
		return nil, nil
	}
	fail := func(format string, args ...any) error {
		return fmt.Errorf("skill %q restricts "+format, append([]any{skillID}, args...)...)
	}
	if declared.Elements == nil && declared.Archetypes == nil &&
		declared.Characters == nil && declared.Species == nil &&
		declared.Origins == nil {
		return nil, fail("nothing, because it names no lists; leave the block out to restrict nothing")
	}
	// A present-but-empty list is refused rather than read as "unrestricted".
	// The two readings differ by everything — one skill nobody may carry
	// against one skill everybody may — and the wrong one is silent.
	for _, list := range []struct {
		name    string
		entries []string
	}{
		{"elements", declared.Elements},
		{"archetypes", declared.Archetypes},
		{"characters", declared.Characters},
		{"species", declared.Species},
		{"origins", declared.Origins},
	} {
		if list.entries != nil && len(list.entries) == 0 {
			return nil, fail("its %s to an empty list, which nobody satisfies; leave the list out to restrict nothing",
				list.name)
		}
		seen := make(map[string]bool, len(list.entries))
		for _, entry := range list.entries {
			if entry == "" {
				return nil, fail("its %s with an empty name", list.name)
			}
			if seen[entry] {
				return nil, fail("its %s to %q twice", list.name, entry)
			}
			seen[entry] = true
		}
	}
	restriction := &Restriction{
		Archetypes: append([]string(nil), declared.Archetypes...),
		Characters: append([]string(nil), declared.Characters...),
		Species:    append([]string(nil), declared.Species...),
		Origins:    append([]string(nil), declared.Origins...),
	}
	for _, name := range declared.Elements {
		member, err := element.Parse(name)
		if err != nil {
			return nil, fail("its elements: %w", err)
		}
		restriction.Elements = append(restriction.Elements, member)
	}
	return restriction, nil
}

// resolveCondition checks one condition, whichever unit it will be read
// against.
//
// One function for both fields because they are one rule: everything below is
// about the shape of a condition and nothing about whose health or whose stacks
// it counts. A second copy would be the one an author trusted, and the two would
// disagree the first time either was edited.
//
// The field name is carried through every message, because "condition asks
// nothing" on a skill with two of them is a refusal that does not say which.
// resolveGradient checks the caster's health gradient, and it takes the whole
// declared skill rather than just its own block because every refusal below is
// about the gradient's relationship with something else on the skill.
//
// It is a second function beside resolveCondition rather than a branch inside
// it: a gradient shares no rule with a condition. It names no status, asks no
// threshold, consumes nothing, and has exactly one number — so the four
// refusals resolveCondition exists for would all be dead code here, and the one
// refusal that matters (a gradient beside a health threshold) is not a rule
// about conditions at all.
func resolveGradient(declared skillFile, selfRequires *Condition) (*Gradient, error) {
	if declared.SelfGradient == nil {
		return nil, nil
	}
	fail := func(format string, args ...any) (*Gradient, error) {
		return nil, fmt.Errorf("skill %q self_gradient: %s", declared.ID, fmt.Sprintf(format, args...))
	}
	// No upper bound, deliberately, and unlike pierce. Piercing more than all of
	// the armour is meaningless, so that one caps at the base; a share added to
	// power has no such ceiling — doubling at the bottom is a thousand and
	// tripling is two, and both are designs somebody may want.
	if declared.SelfGradient.AtEmpty < 1 {
		return fail("adds %d at no health, want a share in parts per thousand", declared.SelfGradient.AtEmpty)
	}
	// The two refusals a bonus power already has, for the same two reasons: a
	// skill aimed at its caster never reaches a target, and a share of nothing
	// is nothing however the caster is doing.
	if declared.Target == Self.String() {
		return fail("scales the power of a skill aimed at itself, which deals none")
	}
	if declared.Power == 0 {
		return fail("scales a power of nought, so the whole curve is worth nothing")
	}
	// ⚠️ A gradient and a health threshold are two answers to one question, and
	// an author reading the skill back could not say which of the two produced a
	// number. A threshold on a *status* is a different question and composes
	// fine, which is why this asks what the condition reads rather than whether
	// there is one.
	if selfRequires.ReadsHealth() {
		return fail("reads the caster's health, and self_requires already reads it as a threshold: " +
			"two curves off one number is a skill nobody can price")
	}
	return &Gradient{AtEmpty: declared.SelfGradient.AtEmpty}, nil
}

func resolveCondition(skillID, field string, declared *conditionFile, deps Deps) (*Condition, error) {
	if declared == nil {
		return nil, nil
	}
	fail := func(format string, args ...any) (*Condition, error) {
		return nil, fmt.Errorf("skill %q %s: %s", skillID, field, fmt.Sprintf(format, args...))
	}
	// A condition may read a status, or health, or both. What it may not do is
	// read neither: an empty condition holds against everybody, so it is a flat
	// power bonus written in the one shape that hides that it is one.
	readsStatus := declared.Status != ""
	readsHealth := declared.BelowHealth != 0
	if !readsStatus && !readsHealth {
		return fail("asks nothing, so it would hold against everybody")
	}

	statusID, minStacks := "", 0
	if readsStatus {
		kind, err := deps.Statuses.Lookup(declared.Status)
		if err != nil {
			return fail("%v", err)
		}
		minStacks = declared.MinStacks
		if minStacks < 1 {
			minStacks = 1
		}
		if minStacks > kind.MaxStacks {
			return fail("needs %d stacks of %q, which caps at %d", minStacks, kind.ID, kind.MaxStacks)
		}
		statusID = kind.ID
	} else if declared.MinStacks != 0 {
		// Stated stacks with nothing to count them of is a half-written
		// condition, and reading it as "no status" would silently drop the half
		// the author did write.
		return fail("asks for %d stacks but names no status", declared.MinStacks)
	}

	if readsHealth && (declared.BelowHealth < 1 || declared.BelowHealth > scale.Base) {
		return fail("holds below %d health, want a share in parts per thousand", declared.BelowHealth)
	}
	if declared.BonusPower < 0 {
		return fail("adds %d power, want zero or more", declared.BonusPower)
	}
	// A gate is a condition on CASTING, and the three refusals below are each a
	// place where that reading would have to be taken somewhere it cannot be.
	//
	// It has to read a status, because a gate on health alone is a different
	// mechanic wearing this field's name. Health is read off a unit, so "may this
	// be cast" would become a question with one answer per cell -- which is
	// battle.aims, and aims is required to stay blind to a gate: four callers ask
	// it hypothetically, one of them while the tank is empty and precisely so that
	// filling the tank can be priced.
	if declared.Gates && !readsStatus {
		return fail("gates its cast but names no status, and a gate on health alone is a reading taken per target rather than per cast")
	}
	// The target's condition is read once per aim; battle.options builds one
	// reason per SKILL. A target-side gate would therefore have to live in aims(),
	// which is the one place it may not -- see above, and battle/turn.go.
	if declared.Gates && field == "requires" {
		return fail("gates its cast on the target's condition, which is read per aim: only the caster's own condition may gate a cast")
	}
	// A gate that spends nothing is a permanent unlock rather than a cost: the
	// caster crosses the threshold once and the skill is free from that turn on.
	// What makes a gate a price is that casting empties what opened it.
	if declared.Gates && !declared.Consume {
		return fail("gates its cast without consuming anything, so the threshold would be crossed once and never paid again")
	}
	// Consuming is a status rule, so a condition that reads only health has
	// nothing to consume and saying so is a mistake rather than a no-op.
	if declared.Consume && !readsStatus {
		return fail("consumes a status, but it names none")
	}
	// A spread is the second currency a consume may be paid in, so the refusal
	// below is now "paid in neither" rather than "not paid in power". Everything
	// about the spread is checked first, so a half-written one is named as itself
	// instead of arriving as the older, vaguer complaint.
	// A caster's own condition may not chain or arc. Both are answers about the
	// board in front of the caster — which bodies are carrying what, and where
	// they are standing — and a caster-side one would be reading a board it is
	// not pointed at.
	if field != "requires" && (declared.Chains || declared.ArcPower != 0) {
		return fail("chains or arcs, which only the target's condition may do: both are read off the unit at the aim")
	}
	// And the mirror of it: a per-stack payment into the caster's own power is
	// the caster's side to make. The target's condition already pays per stack,
	// through the arc, and a condition holding both currencies would be the
	// ceiling charged twice with nothing downstream able to say which half a
	// figure came from — which is the refusal bonus_power and arc_power already
	// share two clauses below.
	if field != "requires" && len(declared.Applies) != 0 {
		// The caster's own condition reads the caster, so a rider here would land
		// on whoever cast the skill -- which self_applies already says without a
		// condition standing in front of it, and says unconditionally. Two ways to
		// write one thing is a reader asking which of them a status came from.
		return fail("carries %d rider(s) on the caster's own condition, which self_applies already writes without one",
			len(declared.Applies))
	}
	if field == "requires" && declared.StackPower != 0 {
		return fail("adds %d power per stack, which only the caster's own condition may do: a target's condition is paid per stack through arc_power",
			declared.StackPower)
	}
	if declared.StackPower < 0 {
		return fail("adds %d power per stack, want zero or more", declared.StackPower)
	}
	// The product is bounded at resolve time — see Condition.Takes, which stops a
	// spend taking stacks the ceiling will not pay for. What is refused here is
	// the one figure that clamp cannot rescue: a term already over the ceiling
	// buys nothing on its second stack however deep the reserve, so the skill an
	// author meant to write scales and the one they wrote does not.
	if declared.StackPower > MaxSpendPower {
		return fail("adds %d power per stack, over the %d one spend may buy: the second stack would be worth nothing",
			declared.StackPower, MaxSpendPower)
	}
	// The health currency, refused on the target's side for the reason the power
	// one is: a caster's own condition is the only side that may pay its caster,
	// and a target's condition filed here would be the enemy's stacks buying the
	// caster health with nothing downstream able to say whose pile paid.
	if field == "requires" && declared.StackRestore != 0 {
		return fail("restores %d health per stack, which only the caster's own condition may do: a target's condition never pays the caster",
			declared.StackRestore)
	}
	if declared.StackRestore < 0 {
		return fail("restores %d health per stack, want zero or more", declared.StackRestore)
	}
	// The mirror of the clause above, against the health ceiling. A rate already
	// over it buys nothing on its second stack however deep the reserve, so the
	// skill the author meant to write scales and the one they wrote does not.
	if declared.StackRestore > MaxSpendRestore {
		return fail("restores %d health per stack, over the %d one spend may buy: the second stack would be worth nothing",
			declared.StackRestore, MaxSpendRestore)
	}
	if declared.ArcPower < 0 {
		return fail("arcs for %d power, want zero or more", declared.ArcPower)
	}
	if declared.ConsumeStacks < 0 {
		return fail("consumes %d stacks, want zero for all of them or a positive count", declared.ConsumeStacks)
	}
	if declared.ConsumeStacks > 0 && !declared.Consume {
		return fail("names %d stacks to consume but does not consume", declared.ConsumeStacks)
	}
	if !declared.Consume && declared.StackPower != 0 {
		// A per-stack payment off a status the skill never spends is a bonus that
		// grows for ever off a counter nothing brings down, which is the same free
		// shape the arc rule below refuses.
		return fail("adds %d power per stack without consuming anything, so it would be paid for the same stack every turn",
			declared.StackPower)
	}
	if !declared.Consume && declared.StackRestore != 0 {
		// The same free shape in health: a heal that grows with a pile nothing
		// brings down is a heal paid for once and collected every turn.
		return fail("restores %d health per stack without consuming anything, so it would be paid for the same stack every turn",
			declared.StackRestore)
	}
	if !declared.Consume && (declared.Chains || declared.ArcPower != 0) {
		// Both describe what spending the status does. Without the consume they
		// would be a skill that arcs and chains for ever off a status it never
		// spends, which is the free shape the old spread rule refused for the same
		// reason.
		return fail("chains or arcs without consuming anything, so the discharge would be free every turn the condition held")
	}
	// ⚠️ **A gating condition is exempt, because the gate is the fourth currency.**
	// Every payment named below is one the condition makes ON TOP of the skill:
	// the cast happens either way, so a consume buying none of them really is a
	// status handed over for nothing. A gate is not on top of anything. The
	// consume buys the CAST -- the skill does not exist without the fuel -- so the
	// flat `power` on the skill's own face is the whole figure, rather than a
	// figure the condition failed to move.
	//
	// It is the same argument one currency along from the StackRestore exemption
	// in resolve, on the "has no power and does nothing else" clause: there the
	// whole heal lives on the condition so the skill reads `restores: 0`, here the
	// whole purchase is the cast so the condition reads as paying nothing.
	if declared.Consume && !declared.Gates && declared.BonusPower == 0 && declared.ArcPower == 0 &&
		declared.StackPower == 0 && declared.StackRestore == 0 && len(declared.Applies) == 0 {
		return fail("consumes %q for neither a bonus, a discharge, a per-stack payment nor a rider, which throws the status away for nothing", statusID)
	}
	if declared.BonusPower != 0 && declared.ArcPower != 0 {
		// Two payments for one stack. A detonate is paid in its own power and a
		// conduit in the charge's; taking both is the ceiling charged twice, and
		// nothing downstream could tell which half a figure came from.
		return fail("is paid both %d bonus power and %d arc power for one stack, which is the same purchase made twice",
			declared.BonusPower, declared.ArcPower)
	}
	if declared.StackPower != 0 && declared.BonusPower != 0 {
		// The same refusal one currency along. A flat bonus and a per-stack one
		// are both paid into the skill's power out of one spend, and a reader
		// comparing two skills could not say which half a figure came from.
		return fail("is paid both %d bonus power and %d power per stack for one spend, which is the same purchase made twice",
			declared.BonusPower, declared.StackPower)
	}
	if declared.StackRestore != 0 && declared.StackPower != 0 {
		// Two rates off one spend, and the refusal is arithmetic rather than
		// taste: Condition.Takes clamps the stacks against the ceiling the payment
		// can buy, there is one Takes, and the two currencies have different
		// ceilings. Whichever it applied would be the wrong one for the other half.
		return fail("is paid both %d power per stack and %d health per stack for one spend, which is two ceilings answering one question",
			declared.StackPower, declared.StackRestore)
	}
	if declared.StackRestore != 0 && declared.BonusPower != 0 {
		// The same purchase made twice, one currency along from the clause above: a
		// flat bonus and a per-stack heal are both paid out of one spend, and a
		// reader comparing two skills could not say which half a figure came from.
		return fail("is paid both %d bonus power and %d health per stack for one spend, which is the same purchase made twice",
			declared.BonusPower, declared.StackRestore)
	}
	applies, err := resolveApplications(skillID, field+".applies", declared.Applies, deps)
	if err != nil {
		return nil, err
	}
	return &Condition{
		Status: statusID, MinStacks: minStacks, Gates: declared.Gates,
		BelowHealth: declared.BelowHealth,
		BonusPower:  declared.BonusPower, Consume: declared.Consume,
		ConsumeStacks: declared.ConsumeStacks, Chains: declared.Chains,
		ArcPower: declared.ArcPower, StackPower: declared.StackPower,
		StackRestore: declared.StackRestore, Applies: applies,
	}, nil
}

func resolveApplications(skillID, field string, declared []applicationFile, deps Deps) ([]Application, error) {
	if len(declared) == 0 {
		return nil, nil
	}
	out := make([]Application, 0, len(declared))
	seen := make(map[string]bool, len(declared))
	for _, entry := range declared {
		kind, err := deps.Statuses.Lookup(entry.Status)
		if err != nil {
			return nil, fmt.Errorf("skill %q %s: %w", skillID, field, err)
		}
		if seen[kind.ID] {
			return nil, fmt.Errorf("skill %q %s: %q appears twice", skillID, field, kind.ID)
		}
		seen[kind.ID] = true
		if entry.Chance < 1 || entry.Chance > scale.Base {
			return nil, fmt.Errorf("skill %q %s: %q has a chance of %d, want between 1 and %d",
				skillID, field, kind.ID, entry.Chance, scale.Base)
		}
		stacks := entry.Stacks
		if stacks < 1 {
			stacks = 1
		}
		if stacks > kind.MaxStacks {
			return nil, fmt.Errorf("skill %q %s: applies %d stacks of %q, which caps at %d",
				skillID, field, stacks, kind.ID, kind.MaxStacks)
		}
		out = append(out, Application{Status: kind.ID, Chance: entry.Chance, Stacks: stacks})
	}
	return out, nil
}

// ParseScaling resolves a scaling declaration as written in a data file.
//
// Exported because a trait's reply is authored the same way and must be read the
// same way: "off which stat, and the base one or the modified one" is one
// question, and two parsers answering it would answer it differently the first
// time either was edited. An empty source is the current value, which is what a
// skill that says nothing has always meant.
func ParseScaling(statName, source string) (Scaling, error) {
	stat, err := parseStat(statName)
	if err != nil {
		return Scaling{}, err
	}
	// Health is refused wherever it is asked for. A thing that scaled off health
	// would grow as its owner was healed, which reads as a reward for being
	// patched up rather than as a stat line.
	if stat == progression.HP {
		return Scaling{}, fmt.Errorf("scales off health, which would make damage grow as its owner is healed")
	}
	switch source {
	case "", "current":
		return Scaling{Stat: stat, Source: combat.CurrentStat}, nil
	case "base":
		return Scaling{Stat: stat, Source: combat.BaseStat}, nil
	default:
		return Scaling{}, fmt.Errorf("scales off the %q value, want \"base\" or \"current\"", source)
	}
}

func parseStat(name string) (progression.Kind, error) {
	for _, kind := range progression.Kinds() {
		if kind.String() == name {
			return kind, nil
		}
	}
	return 0, fmt.Errorf("unknown scaling stat %q", name)
}

// Skills returns every declared skill in declaration order.
func (b *Book) Skills() []Skill {
	out := make([]Skill, len(b.skills))
	copy(out, b.skills)
	return out
}

// Marshal writes the book as a data file: two-space indented JSON, in the order
// the skills were declared.
//
// Declaration order rather than sorted by id, which is where this differs from
// cast.Book.Marshal, and the difference is the point. A cast is a set looked up
// by id, so sorting it is free and makes an addition a one-block diff. A skill
// book's order is authored information — the shipped file reads basic attacks,
// then the elemental ones, then the utility skills, and skills.golden's table is
// that order — so sorting would shuffle a design record to buy the same
// one-block diff that appending already gives.
func (b *Book) Marshal() ([]byte, error) {
	out, err := json.MarshalIndent(marshalFile{Skills: b.Skills()}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode skill book: %w", err)
	}
	return append(out, '\n'), nil
}

// Append returns a new book holding the declared skills plus the extra ones,
// validated exactly as a parse would validate them.
//
// It works by marshalling and re-parsing rather than by re-implementing the
// checks, which is what guarantees the bytes an authoring tool is about to write
// are bytes that load.
func (b *Book) Append(deps Deps, extra ...Skill) (*Book, error) {
	raw, err := json.Marshal(marshalFile{Skills: append(b.Skills(), extra...)})
	if err != nil {
		return nil, fmt.Errorf("encode skill book: %w", err)
	}
	return ParseBook(raw, deps)
}

// Replace returns a new book with one skill's declaration changed, in the place
// that skill already had.
//
// Keeping the position is the whole of it, and it is the same reason Marshal
// keeps declaration order rather than sorting: the shipped skills.json is
// committed in the form Marshal writes, and its order is authored information —
// basic attacks, then the elemental ones, then utility, which is the order
// skills.golden's table reads in. Moving an edited skill to the end would
// rewrite every line of the file and every row of that table to change one
// number, which is exactly what committing the file in Marshal form was meant to
// avoid.
//
// It validates the way Append does, by marshalling and re-parsing rather than by
// re-implementing a check, so the bytes an authoring tool is about to write are
// bytes that load. A skill the book does not already hold is refused: this
// changes a declaration and never adds one, so an id that is not there is a
// caller's mistake rather than a new skill.
//
// The id itself cannot be changed through this, because the id is what the
// position is found by. Renaming a skill has to walk every kit and every
// restriction that names the old one, which is a different operation.
func (b *Book) Replace(deps Deps, edited Skill) (*Book, error) {
	skills := b.Skills()
	at := slices.IndexFunc(skills, func(current Skill) bool { return current.ID == edited.ID })
	if at < 0 {
		return nil, fmt.Errorf("unknown skill %q", edited.ID)
	}
	skills[at] = edited
	raw, err := json.Marshal(marshalFile{Skills: skills})
	if err != nil {
		return nil, fmt.Errorf("encode skill book: %w", err)
	}
	return ParseBook(raw, deps)
}

// Lookup returns a skill by id.
func (b *Book) Lookup(id string) (Skill, error) {
	found, ok := b.byID[id]
	if !ok {
		return Skill{}, fmt.Errorf("unknown skill %q", id)
	}
	return found, nil
}

// CarryRefusal is why an affinity may not carry a skill, or that it may.
//
// It is a classification rather than a sentence for the same reason
// internal/forge returns values: three callers refuse a kit and each words it
// for its own reader — the engine for a log, the parser for a data file, the
// authoring tool in the author's language — and a sentence cannot be reworded
// after it is built without being taken apart again.
type CarryRefusal uint8

const (
	// CarryAllowed is a skill the affinity may use.
	CarryAllowed CarryRefusal = iota
	// CarryWrongElement is a skill of an element the unit does not have.
	CarryWrongElement
	// CarryElementRestricted is a skill whose element allowlist excludes the
	// unit. It is a separate answer from CarryWrongElement because the two need
	// different advice: one is fixed by changing the affinity to the skill's
	// element, and the other cannot be, because the skill's own element is
	// already shared.
	CarryElementRestricted
)

// WhyCannotCarry is the whole of which skills an affinity may carry.
//
// Two conditions, in the order they are worth reporting. A unit may only use a
// skill of an element it shares — a neutral skill is universal, which is what
// makes a second element worth carrying — and it must also satisfy whatever
// element allowlist the skill declares, which is the narrower rule a
// restriction adds. Nothing else here is enforceable: an archetype, a character
// identity and a species do not reach the engine, so those three halves of a
// restriction are checked where a character is authored. See Restriction.
//
// This is the single declaration. battle.enlist refuses a roster entry that
// breaks it, cast.ParseBook refuses an authored character that breaks it and
// forge.CheckCarry brings the same answer forward to the moment an answer is
// typed — all three by calling this, because two callers wording one rule in
// their own words is how the two come to disagree, and the disagreement here
// would be a character that writes cleanly and then cannot enter a battle.
func WhyCannotCarry(affinity element.Affinity, carried Skill) CarryRefusal {
	if carried.Element != element.Neutral && !affinity.Has(carried.Element) {
		return CarryWrongElement
	}
	if !carried.Restrict.AllowsElement(affinity) {
		return CarryElementRestricted
	}
	return CarryAllowed
}

// CanCarry reports whether a unit of the given affinity may use this skill.
// It is WhyCannotCarry for a caller that only needs the yes or no.
func CanCarry(affinity element.Affinity, carried Skill) bool {
	return WhyCannotCarry(affinity, carried) == CarryAllowed
}

// Demands returns the distinct non-neutral elements a kit requires, in the
// order the skills were listed.
//
// It is what a kit says about the affinity that may hold it: a unit must have
// every element in this set, so a kit demanding more than two can never be
// carried at all, and a kit demanding two can only be carried by exactly that
// pair. Deriving it from the skills is the point — an authored hint would be
// free to drift from the kit it described.
func Demands(kit []Skill) []element.Element {
	out := make([]element.Element, 0, 2)
	for _, carried := range kit {
		if carried.Element == element.Neutral {
			continue
		}
		if !slices.Contains(out, carried.Element) {
			out = append(out, carried.Element)
		}
	}
	return out
}
