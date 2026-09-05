package room

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/wire"
)

// seatCount is how many seats a room has, and it is two: spectators are later
// work and a third client is a full room today.
//
// The seats are an array indexed by this rather than a map keyed by wire.Seat,
// which is the engine's own rule applied one layer out — the order two seats are
// visited in reaches the roster, and the roster's order decides which side wins
// a speed tie. A map would randomise it.
const seatCount = 2

// seats is every seat a room hands out, in the order a room hands them out.
var seats = [seatCount]wire.Seat{wire.SeatHost, wire.SeatGuest}

// indexOf is a seat's position, and reports false for anything that is not one
// of the two — including the zero Seat, which means "no seat" and must not
// quietly mean the host.
func indexOf(seat wire.Seat) (int, bool) {
	for index, candidate := range seats {
		if candidate == seat {
			return index, true
		}
	}
	return 0, false
}

// other is the seat that is not this one.
func other(seat wire.Seat) wire.Seat {
	index, known := indexOf(seat)
	if !known {
		return ""
	}
	return seats[(index+1)%seatCount]
}

// Config is everything a room is set up with, and every one of these is a
// decision the host makes before anybody joins.
//
// ⚠️ There is no clock in it and there is no clock anywhere in this package.
// Allowance is a *number the room carries and hands to its clients*; whoever
// owns the transport counts it down and tells the room when it ran out. See the
// package comment, and TestTheRoomReadsNoClock, which holds it mechanically.
type Config struct {
	// Format is how many units a side, and the gate refuses a squad that is not
	// that size.
	Format wire.Format
	// Battles is the series length, and **bo1 is not a special case — it is
	// N = 1**. Only 1 and 3 are accepted: only an even series cancels the side
	// advantage, and only an even series has to invent a rule for a 1–1. The
	// aggregate-health tie-break an earlier draft proposed is dropped, so no
	// invented metric ships anywhere in here.
	Battles int
	// Allowance is how long a player has to answer one prompt, in seconds. The
	// room hands it to both clients on wire.Welcome and never counts it down.
	Allowance int
	// Seed is the match's one seed. Every battle's seed is derived from it and
	// the battle's index (SeedFor), so a whole match is reproducible from a
	// single number.
	Seed uint64
	// TurnCap is how many turns one battle may open before the room stops
	// asking, so a runaway cannot hold two people at a board for ever.
	//
	// ⚠️ **It rides on wire.Welcome**, beside the allowance and for the same
	// reason: a cap is room configuration the client needs in order to behave
	// correctly, and the client is a mirror — given the cap it stops on the same
	// turn by the same arithmetic, so no message and no Ended is needed to tell
	// it a battle was capped. → wire.Welcome.TurnCap, which carries the three
	// alternatives that were refused.
	TurnCap int
	// Password keeps strangers in the house off the board and is **not**
	// security. Empty is a room with none, which accepts any hello that gets
	// past the version gate.
	Password wire.Password
	// Drafts says the two sides **ban and pick** before they fight, out of one
	// shared pool, rather than each bringing a squad it built at home. It rides
	// to both clients on wire.Welcome, which is what tells a client not to bring
	// one.
	//
	// ⚠️ **Whether the pool can seat the format is NOT checked here, and cannot
	// be.** draft.Fits measures a pool against a format, the pool is the cast
	// minus every character held back, and Validate is a method on this struct
	// with no Deps to read a cast book out of. So that check lives in New, which
	// has both — and a room whose draft could never finish therefore fails when
	// it is opened rather than when somebody joins it, which is the arrangement
	// internal/draft's own New rests on. Measured on the shipped cast: a pool of
	// **19**, so 3v3 has nine characters to spare and 5v5 has three.
	//
	// ⚠️ What Validate **can** say about it is that the series is a bo1, which is
	// below.
	Drafts bool
}

// DefaultTurnCap is the backstop a host has no reason to change: about eight
// times the longest battle measured on the shipped 3v3 (34–55 decisions), so it
// can only be reached by something genuinely endless — two units buffing
// themselves at each other for ever — rather than by a long fight.
const DefaultTurnCap = 400

// DefaultAllowance is ninety seconds a prompt, which is the figure the design
// record argues with rather than the one it recommends: at 34–55 decisions a
// battle it is sixty-eight minutes a battle and three and a half hours for a
// bo3. It is the default because it is what was decided; it is configuration
// because that argument is not settled.
const DefaultAllowance = 90

// Validate checks a room's configuration, and refuses a bo2 by name because
// that refusal is a design decision rather than a bounds check.
func (c Config) Validate() error {
	if !c.Format.Valid() {
		return fmt.Errorf("a room of %s is not a format this game offers", c.Format)
	}
	if c.Battles != 1 && c.Battles != 3 {
		if c.Battles > 0 && c.Battles%2 == 0 {
			return fmt.Errorf("a series of %d battles is even, and an even series has to invent a rule for a %d–%d: the room offers 1 or 3",
				c.Battles, c.Battles/2, c.Battles/2)
		}
		return fmt.Errorf("a series of %d battles is not one the room offers: 1 or 3", c.Battles)
	}
	if c.Allowance <= 0 {
		return fmt.Errorf("an allowance of %d seconds gives a player no turn to take", c.Allowance)
	}
	if c.TurnCap <= 0 {
		return fmt.Errorf("a turn cap of %d ends every battle before it starts", c.TurnCap)
	}
	// ⚠️ **The ban and pick is bo1's, by name, and this refusal is a design
	// decision rather than a bounds check** — the same kind as the bo2 above.
	// "A ban lasts the match, and the first cut is bo1 only" is what was settled;
	// what a draft means in a *series* is three different games — three drafts,
	// one draft carried across all three battles, or a draft per battle with the
	// previous winner banning first — and choosing one of them by accepting the
	// configuration would be taking that decision here. A room that accepted it
	// would silently ship the second reading. → TODO.md § "Ban and pick for a
	// bo3", which is its own item.
	if c.Drafts && c.Battles != 1 {
		return fmt.Errorf("a drafting room of %d battles has no rule to run under: a ban lasts "+
			"the match and the first cut is bo1 only, so what a draft means across a series — "+
			"three drafts, one draft carried, or a draft a battle with the previous winner "+
			"banning first — is a decision nobody has taken", c.Battles)
	}
	return nil
}

// SeedFor is the seed of one battle of the series, counting from one.
//
// The derivation is the first eight bytes of `sha256(seed ‖ index)`, both
// written as fixed-width big-endian integers — the same framing internal/seed
// and internal/wire hash their inputs under, which is why it is spelled this way
// rather than invented. It reads no clock and draws no randomness of its own: it
// is a pure function of two integers, which is what makes a whole match
// reproducible from one number and what will make a match log verifiable the day
// the room writes one out.
//
// ⚠️ **The obvious version was written first, measured, and is wrong.** Reusing
// the mixer rng already declares — `rng.New(Seed + index).Next()`, one round of
// splitmix64 — looks like exactly the right move under this repository's rule
// about not restating arithmetic, and it collides **structurally**: splitmix64
// advances by adding a constant, so one round of it is a function of `Seed +
// index` alone, and battle two of a match seeded 6 *is* battle one of a match
// seeded 7. Two different matches sharing a battle is not a rounding error; it
// is the coincidence a balance run reads as a result and a bug report cannot
// see. Every counter-based generator has that shape, so no arrangement of rng
// fixes it — a derivation from *two* numbers needs a function of two numbers.
// TestAWholeMatchIsReproducibleFromOneNumber is what caught it, by asking the
// question rather than by asserting three values.
//
// The mixing is also what makes the low bit worth reading, which is what picks
// the side of an uncancelled battle below.
func (c Config) SeedFor(index int) uint64 {
	var framed [16]byte
	binary.BigEndian.PutUint64(framed[0:8], c.Seed)
	binary.BigEndian.PutUint64(framed[8:16], uint64(index))
	digest := sha256.Sum256(framed[:])
	return binary.BigEndian.Uint64(digest[:8])
}

// HomeFor is which seat is **home** for one battle of the series: enlisted
// first, and therefore on hex.SideAlly.
//
// Battles alternate, starting with the host, because which side you get is
// worth up to sixty points in a mirror and fighting both ways round is the only
// thing that cancels it. What alternation cannot cancel is a battle with no
// partner — bo1's only battle, and the third battle of a 1–1 bo3 — and those
// two are the same problem, so they get one rule: **the seed picks the side**,
// off the low bit of that battle's own derived seed.
//
// ⚠️ It is honestly uncancelled and the room says so rather than dressing a
// coin as fairness. bo3 is worse than it reads for exactly this reason: an odd
// series does not remove the side advantage, it concentrates it into the one
// battle that matters most.
//
// ⚠️ The refinement the design record pairs with this — the lead of each
// contested speed group alternating, which was measured at 49.6% against 54.2%
// for ally-first on a two-unit mirror — is **not here**. It needs the roster
// slice composed against the queue rather than as the squads were authored, the
// side is worth up to sixty points, and what it is worth at 3v3 or 5v5 is
// unmeasured. → TODO.md, its own item.
func (c Config) HomeFor(index int) wire.Seat {
	// The battles that pair off, which is every battle but the last of an odd
	// series. For N = 1 that is none of them.
	paired := 2 * (c.Battles / 2)
	if index >= 1 && index <= paired {
		return seats[(index-1)%seatCount]
	}
	return seats[c.SeedFor(index)&1]
}

// sideOf is the half of the board a seat plays in the battle in progress: ally
// for the home seat, because home is what "enlisted first" means.
func (r *Room) sideOf(seat wire.Seat) hex.Side {
	if seat == r.home {
		return hex.SideAlly
	}
	return hex.SideEnemy
}

// seatOnSide is the inverse, for reading a battle's winner back out as a seat.
func (r *Room) seatOnSide(side hex.Side) wire.Seat {
	if side == hex.SideAlly {
		return r.home
	}
	return other(r.home)
}

// seriesOver reports whether the series is decided: a seat holds more battles
// than the rest of the series can take back, or every battle has been fought.
//
// The lead that decides it is `Battles/2` — one win in a bo1, two in a bo3 —
// and that expression is only right for an odd series, which is one of the two
// reasons Validate refuses an even one. The other is that an even series has to
// invent a tie-break.
func (r *Room) seriesOver() bool {
	if len(r.played) >= r.config.Battles {
		return true
	}
	for _, won := range r.standing {
		if won > r.config.Battles/2 {
			return true
		}
	}
	return false
}

// leader is the seat holding the most battles, and the zero Seat when nobody is
// ahead — which is a draw rather than a tie to be broken.
func (r *Room) leader() wire.Seat {
	best, at := -1, -1
	tied := false
	for index, won := range r.standing {
		switch {
		case won > best:
			best, at, tied = won, index, false
		case won == best:
			tied = true
		}
	}
	if tied || at < 0 {
		return ""
	}
	return seats[at]
}
