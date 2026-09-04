// Package draft is the ban-and-pick that runs before a PvP match: who a draft
// may seat, and the arithmetic that says whether a format can be drafted at all.
//
// It sits beside internal/room and lives under the same rules, for the same
// reason. A draft is a **pure function of the decisions taken**, so a client can
// replay one from the decisions alone and the server needs to send nothing else;
// a clock or an entropy source here would break that mirror exactly as it would
// in a room. TestTheDraftReadsNoClock refuses both by import.
//
// # Why a shortfall is a refused configuration and never a runtime failure
//
// Every ban and every pick takes exactly one character out of the shared pool,
// and a ban is optional — a side that leaves a slot unspent takes out **fewer**.
// So the removals a whole draft can make are at most `2*PicksPerSide +
// 2*BansPerSide`, which is the number Fits requires the pool to hold. A draft
// that Fits allowed therefore cannot run dry: before the k-th of at most that
// many decisions at most k-1 characters are gone, so at least one is left for
// the decision about to be taken, and the pool never falls below the picks the
// two sides still owe.
//
// The consequence is worth stating plainly, because the design record used to
// say the opposite: **there is no "the pool would no longer seat both sides"
// rule to write.** Optional bans can only ever help, so the worst case is the
// one where every ban is spent, and that is precisely the case Fits is computed
// against. Nothing here, and nothing in the state machine above it, has to
// refuse a ban or grey a slot at runtime.
// TestNoDraftThatFitsCanRunOutOfCharacters drives that exhaustively, and
// TestADraftThatDoesNotFitCanRunOutOfCharacters is the half that proves the walk
// can see the failure it says never happens.
//
// None of it depends on the order the decisions come in — bans first, picks
// first or interleaved all remove one character apiece — so the sequence is the
// state machine's business and not this file's.
package draft

import (
	"fmt"
	"slices"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/wire"
)

// Pool is the characters one draft may seat: the authored cast minus the ones
// held back. It is fixed for the whole of a draft — what a ban and a pick take
// out is the state machine's business, and this is the list they take it out of.
type Pool struct {
	characters []cast.Character
}

// NewPool is the cast a draft may ban and pick from: `all` minus every
// cast.Character.Hidden. `all` is what a caller got from cast.Book.All().
//
// # Hidden is a design statement here, not an authoring convenience
//
// ⚠️ The flag's own comment calls itself "an authoring convenience rather than a
// design statement", and it was exactly that while the squad builder was its
// only reader: holding a character back kept it off a list an author was
// choosing from, and a saved squad naming one stayed as valid as any other. This
// gate makes it a **rule of the game** — a held-back character cannot be banned,
// cannot be picked, and so cannot be fielded in a drafted match at all. Nothing
// about the flag changed; what changed is that a second reader honours it for a
// reason a player can feel.
//
// It filters here rather than through a `cast.Book.Offered()` accessor, and that
// was decided rather than deferred — see internal/screen/squads.go's
// offeredCharacters, which predicted this package and asked the same question. A
// draft wants plain Hidden with nobody kept; the squad builder wants
// Hidden-minus-a-`keep`, which is a fact about the squad under edit that no book
// can know. An accessor would answer half of each caller and both would still
// write a loop around it.
//
// ⚠️ And this is not "filter Hidden everywhere": internal/screen/picker.go's
// CharacterOptions offers a held-back character **on purpose**, because a skill
// restriction that already names one has to stay authorable. The two lists share
// a shape and share nothing else.
//
// # The order is the book's own declaration order, and it is NOT sorted by id
//
// ⚠️ **Do not "fix" this by sorting.** Order here is a presentation choice and
// not a determinism one: nothing in this package iterates a map, Book.All() is
// already a slice, and a ban or a pick names a character by id on the wire, so
// the record a draft leaves does not depend on it either way. Where the order
// does reach is the draft screen — and five other lists in this repository (the
// browser, the builds screen, `hexforge list`, the squad builder's chooser and
// the restriction picker) are all in the book's declaration order. A draft
// screen that sorted would lay the cast out differently from every screen the
// player has just come from, which is the one thing an order can get wrong here.
//
// ⚠️ The shipped cast happens to be in id order today, so sorting it would be a
// no-op and a test taken against it would measure nothing.
// TestThePoolKeepsTheBooksDeclarationOrder uses a book of its own whose
// declaration order is not its id order, for exactly that reason.
func NewPool(all []cast.Character) Pool {
	seated := make([]cast.Character, 0, len(all))
	for _, character := range all {
		if character.Hidden {
			continue
		}
		seated = append(seated, character)
	}
	return Pool{characters: seated}
}

// Len is how many characters the draft may seat.
func (p Pool) Len() int { return len(p.characters) }

// Has reports whether a character is in the pool, by id.
//
// It scans rather than holding an index, and at this size that is deliberate:
// the collection is the whole authored cast, so the scan is free, and a map
// beside the slice would be a second copy of the pool that a later `range` could
// turn into an ordering — the one thing a draft, like a room, may not do.
func (p Pool) Has(id string) bool {
	return slices.ContainsFunc(p.characters, func(character cast.Character) bool {
		return character.ID == id
	})
}

// All is the pool in the cast book's declaration order, as a copy — the same
// promise cast.Book.All() makes, for the same reason: a caller that sorted or
// appended to what it was handed would otherwise reorder the pool itself.
//
// ⚠️ The copy is of the **slice**. cast.Character carries slices of its own and
// cast's deep clone is unexported, so those stay shared — which is the depth
// `all` arrived at anyway, because NewPool's argument is what a caller got from
// Book.All() and that already cloned each character once out of the book. What
// this copy stops is a caller changing the pool's shape or its order, which is
// the only thing anything here does to a list.
func (p Pool) All() []cast.Character {
	out := make([]cast.Character, len(p.characters))
	copy(out, p.characters)
	return out
}

// PicksPerSide is how many characters a side picks, which is the format's own
// unit count: a drafted squad is the squad it fights with.
func PicksPerSide(format wire.Format) int { return format.Units() }

// BansPerSide is how many characters a side may ban: two at 3v3, three at 5v5.
// Mirrored, and **optional** — a side may leave every slot unspent.
//
// Settled in TODO.md § "Ban and pick" (b), and read as *a side* rather than as a
// total across both, which is what "three bans, mirrored" meant when it was
// first settled.
//
// ⚠️ An unknown format answers nought, and that is no answer rather than "a
// format with no bans" — the same shape wire.Format.Units() has, which answers 4
// for a Format(4) the game does not offer. Format.Valid is the gate for both,
// and Fits is where this package applies it, so a caller doing this arithmetic
// without going through Fits is a caller that never asked whether its format
// exists.
func BansPerSide(format wire.Format) int {
	switch format {
	case wire.Format3v3:
		return 2
	case wire.Format5v5:
		return 3
	default:
		return 0
	}
}

// Slack is what the pool has left over once both sides have picked and spent
// every ban: `poolSize - 2*PicksPerSide - 2*BansPerSide`. It goes negative for a
// pool that cannot seat the format, which is what Fits reports.
//
// ⚠️ It is worth drawing even when it is comfortable, because **the last pick is
// not a decision whenever slack is nought**: a draft whose final pick has one
// candidate should say so rather than present a list of one.
func Slack(poolSize int, format wire.Format) int {
	return poolSize - 2*PicksPerSide(format) - 2*BansPerSide(format)
}

// Fits reports whether a format can be drafted out of a pool this size, and
// names the shortfall in characters when it cannot.
//
// It measures against every ban being **spent**, which is the worst case and so
// the only one worth refusing: bans are optional, a side that skips one takes
// fewer characters out, and a draft this allows provably cannot run out of
// characters partway through — see the package comment for the argument and the
// test that drives it.
//
// The refusal is therefore deliberately stricter than the arithmetic strictly
// needs — a 5v5 in which both sides happened to skip every ban would sit inside
// a pool of fifteen — and that is the trade being taken on purpose. A
// configuration refused when the room opens beats a draft that fails halfway
// through, and the alternative costs a runtime rule in every step above this
// one.
func Fits(poolSize int, format wire.Format) error {
	if !format.Valid() {
		return fmt.Errorf("a draft of %s is not a format this game offers", format)
	}
	slack := Slack(poolSize, format)
	if slack < 0 {
		picks, bans := 2*PicksPerSide(format), 2*BansPerSide(format)
		return fmt.Errorf("a %s draft takes %d picks and %d bans out of one shared pool, "+
			"which is %d characters, and the pool holds %d: it is short by %s",
			format, picks, bans, picks+bans, poolSize, characterCount(-slack))
	}
	return nil
}

// characterCount writes a count of characters the way the rest of this
// repository writes a count of one — "one character" rather than "1 characters",
// which is a wording internal/i18n has already had to fix once.
func characterCount(n int) string {
	if n == 1 {
		return "one character"
	}
	return fmt.Sprintf("%d characters", n)
}
