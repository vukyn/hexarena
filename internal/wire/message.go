package wire

import (
	"crypto/subtle"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
)

// Format is the room's size: how many units a side fields.
//
// ⚠️ Its value **is** that number, which is what makes it the one enum here that
// is not written by name. A kind and a code serialise by name because their
// values are positions in a declaration and inserting a constant would
// reinterpret every message already written; three is three whichever order
// these two lines are in, so there is nothing for a rename or an insertion to
// break. A `Format` of 4 is refused rather than misread.
type Format int

const (
	// Format3v3 is three units a side.
	Format3v3 Format = 3
	// Format5v5 is five units a side. ⚠️ The shipped balance was read at five a
	// side and the shorter board leaves a summon more free slots, so these are
	// not two settings of one measurement — → TODO.md, "read the balance again
	// at 3v3".
	Format5v5 Format = 5
)

// Units is how many units a side fields, which is the format's own value.
func (f Format) Units() int { return int(f) }

// Valid reports whether the format is one the room offers.
func (f Format) Valid() bool { return f == Format3v3 || f == Format5v5 }

func (f Format) String() string { return fmt.Sprintf("%dv%d", int(f), int(f)) }

// Seat is which of a room's two places a client took, and it holds for the whole
// match.
//
// ⚠️ A seat is not a side. A match fights **both ways round**, because which side
// you get is worth up to sixty points and only fighting from both ends cancels
// it, so the side a client plays changes between battles and arrives on Start;
// the seat does not change at all and arrives once, on Welcome. Two facts, two
// messages, and a client that read one for the other would draw the wrong half
// of the board from the second battle on.
//
// It is a named string rather than an iota with a table of names, and that is
// the cheaper end of the same rule: a string type cannot be reinterpreted by an
// insertion because it has no declaration order to reinterpret, so it needs no
// MarshalJSON, no names table and no count to be held against. Kind and Code
// cannot be strings — they are dispatched on and walked by index — and this is
// neither.
type Seat string

const (
	// SeatHost is the client that opened the room. It is also the one holding
	// the process, which is a fact about the network and not about the game:
	// nothing in the engine knows or cares which peer is which.
	SeatHost Seat = "host"
	// SeatGuest is the client that joined with the room's code.
	SeatGuest Seat = "guest"
)

// Valid reports whether the seat is one a room hands out. The zero value is not:
// an absent seat means what it says rather than quietly meaning the host.
func (s Seat) Valid() bool { return s == SeatHost || s == SeatGuest }

// Password is a room's password, and the design record says plainly what it is
// **not**: it is not security. This is a plain WebSocket on a LAN, and the
// password keeps strangers in the house off the board rather than keeping an
// attacker out.
//
// Two things are owed to it regardless of that, and both are here:
//
//   - It is compared in **constant time** (Password.Equal), so the comparison
//     cannot be timed.
//   - It is **never printed**. String and GoString both redact it, which is what
//     makes a `%v` or a `%+v` of any struct holding one safe — fmt calls a
//     field's own String method — and TestARoomPasswordIsNeverPrinted measures
//     exactly that over every message body by reflection.
//
// ⚠️ What the redaction cannot cover: the password rides in a hello in the clear,
// so a peer that logs the raw envelope bytes logs the password. That is the
// room's discipline to keep, and it is the price of the design record's own
// decision that a self-signed certificate implying security would be worse than
// saying there is none.
type Password string

// Equal reports whether two passwords match, in constant time.
func (p Password) Equal(other Password) bool {
	return subtle.ConstantTimeCompare([]byte(p), []byte(other)) == 1
}

// Set reports whether there is a password at all. A room with none accepts any
// hello that gets past the version gate.
func (p Password) Set() bool { return p != "" }

// String is the redaction, and it says the one thing about a password that is
// safe to say: whether there is one.
func (p Password) String() string {
	if p == "" {
		return "[unset]"
	}
	return "[set]"
}

// GoString redacts under %#v as well, which is the other verb a debug line
// reaches for.
func (p Password) GoString() string { return "wire.Password(" + p.String() + ")" }

// Hello is the first thing a client says. Client → server.
//
// It carries the three version numbers inline (see Version), the squad the
// player built on their own machine, the room's password and a name for the
// other player to read.
//
// The squad is a placement.Squad and not a resolved roster, which is the whole
// reason a client can be trusted with it: a Placement is a *reference* —
// character, level, stage, four skills, one trait, a slot — and carries no stat
// line at all, so an inflated one is unrepresentable rather than caught. The
// server resolves it with Squad.Take, which is already the legality check.
type Hello struct {
	Version
	// Squad is the placement this player brings, by reference.
	Squad placement.Squad `json:"squad"`
	// Name is what the other player sees. It is the one free-text field in the
	// protocol and it is **not** prose in the sense the record bans: nothing
	// words it, nothing branches on it, and it is the player's own writing
	// rather than the server's.
	Name string `json:"name,omitempty"`
	// Password is the room's password, or empty for a room with none.
	Password Password `json:"password,omitempty"`
}

// Kind is KindHello.
func (Hello) Kind() Kind { return KindHello }

// Act is a client spending its turn: a skill and where it is pointed. Client →
// server.
//
// ⚠️ It carries **no unit**. The server knows whose turn it is — it holds the
// authoritative battle and it is the thing that produced the prompt — so a unit
// on this message would be a second statement of a fact one side already owns,
// and the only thing it could add is a disagreement to resolve. The same reason
// keeps a whole battle.Decision off this side of the wire: a Decision carries
// the unit, the turn number and a pass reason, and every one of those three is
// the server's to record.
type Act struct {
	// Skill is the id of the skill to use, out of the four the unit brought.
	Skill string `json:"skill"`
	// Aim is the cell to point it at, absent for a skill that points nowhere.
	// hex.Cell's absence is the real thing rather than a coordinate standing in
	// for one — an aim of nought would be the ally back corner.
	Aim hex.Cell `json:"aim,omitzero"`
}

// Kind is KindAct.
func (Act) Kind() Kind { return KindAct }

// Pass is a client giving its turn up. Client → server. **It carries nothing at
// all**, and the empty struct is the point rather than an oversight.
//
// ⚠️ There is no reason field. A passed turn's wording lives on
// battle.Decision, and battle.NoActionReason is this repository's single
// declaration of it — a client that sent a reason would be a second one, and two
// callers wording the same choice differently is precisely what made a replay
// diverge from the log it was replaying once already (→ CLAUDE.md § Mistakes
// already made here, "One source for a recorded string"). The server records the
// reason because the server is what writes the log.
//
// It is a struct rather than an alias for nothing so that it satisfies Body and
// travels as a body-less envelope, which is the one kind Decode accepts empty.
type Pass struct{}

// Kind is KindPass.
func (Pass) Kind() Kind { return KindPass }

// Welcome is the room accepting a client: the configuration the whole match runs
// under, and which seat this client took. Server → client.
type Welcome struct {
	// Format is how many units a side.
	Format Format `json:"format"`
	// Battles is the series length, and **bo1 is not a special case — it is
	// N = 1**. The room offers 1 or 3; 2 is deliberately not offered, because
	// only an even series cancels the side advantage and only an even series has
	// to invent a rule for a 1–1.
	Battles int `json:"battles"`
	// Allowance is how long a player has to answer one prompt, **in seconds**.
	//
	// ⚠️ Seconds as an int, deliberately not a time.Duration: a Duration
	// JSON-encodes as a count of nanoseconds, which is unreadable in a golden
	// and unreadable to anything that is not Go. And the allowance is **room
	// configuration**, not part of the battle — what enters the engine when it
	// runs out is a Pass, never a timestamp, never a duration, never a reading
	// of a clock. That is why it sits here beside the format and why no
	// battle-carrying message holds a clock reading at all.
	Allowance int `json:"allowance"`
	// TurnCap is how many turns one battle may open before the room stops
	// asking, so a runaway cannot hold two people at a board for ever.
	//
	// ⚠️ It is here for the reason the allowance is, and the argument is the one
	// written above: a cap is **room configuration**, not part of the battle.
	// The allowance is here so a client can count down; the cap is here so a
	// client can **stop on the same turn**. Without it a capped battle is
	// invisible to a mirror — the engine emits no Ended at a cap, and it must
	// not, because a cap is somebody deciding to stop rather than a way a
	// battle can end — so the client would sit holding an open prompt on a
	// battle the room had already stopped asking about.
	//
	// That is sufficient with no new message and no Ended: the client is a
	// mirror, so given the cap it reaches the cap on the same turn by the same
	// arithmetic. Two peers agree because they compute the same thing from the
	// same configuration, which is the mirror contract itself. ⚠️ Skipped turns
	// count towards it — a turn is a turn — so a client counting only the turns
	// that arrive on a Turn would sit behind the cap for a whole battle.
	//
	// ⚠️ Three alternatives were considered and refused. → README.md § PvP over
	// a LAN: a constant both peers read (the host loses the setting, and a
	// version skew then desyncs silently where a config field is checked at the
	// handshake), a "battle was capped" message (a protocol bump *and* a second
	// declaration of how a battle ends), and letting the engine emit Ended at
	// the cap (which would make every renderer and --verify learn a room's
	// policy).
	TurnCap int `json:"turn_cap"`
	// Drafts says the two sides ban and pick before they fight, rather than each
	// bringing a squad it built at home.
	//
	// ⚠️ **It is here so a client knows not to bring a squad**, and that changes
	// what Hello.Squad means in such a room: it becomes **unwanted** rather than
	// silently ignored, which is what CodeSquadUnwanted is for. A squad quietly
	// dropped would be a player watching the side they spent an evening building
	// fail to appear, with nothing anywhere saying why.
	//
	// ⚠️ **The ordering is the awkward half, and it is a room's problem rather
	// than this field's.** A hello is the first thing a client says, so it is
	// sent *before* this welcome arrives and a client cannot read this and then
	// decide. So a room that drafts refuses the squad and the client joins again
	// with none; this flag is what makes the second attempt informed rather than
	// a guess, and it is what a client reads to know a draft is coming at all.
	//
	// It carries no omitempty, like every field above it: a room that does not
	// draft writes `false` rather than nothing, so the configuration a match runs
	// under reads whole in a log and in the golden.
	Drafts bool `json:"drafts"`
	// Seat is which of the room's two places this client took, for the match.
	Seat Seat `json:"seat"`
}

// Kind is KindWelcome.
func (Welcome) Kind() Kind { return KindWelcome }

// Refused is the room turning a client away: a Code and nothing else. Server →
// client.
//
// Nothing accompanies the code — no sentence, no field naming what was wrong
// with the squad, no number. The client holds the same books, the same
// validator and both language books, so it can say more about its own squad than
// the server could, in the language the player reads.
type Refused struct {
	Code Code `json:"code"`
}

// Kind is KindRefused.
func (Refused) Kind() Kind { return KindRefused }

// Start opens one battle of the series. Server → client, once per battle.
//
// This is everything a mirror needs to build its own *battle.Battle: the seed
// its rolls come from and the roster it was composed with. Nothing else about
// the battle ever crosses — no board, no health, no statuses, no queue — because
// the client computes the state by computing the battle.
type Start struct {
	Seed uint64 `json:"seed"`
	// Roster is every unit of both sides, resolved, **in the order the battle
	// enlists them** — which is exactly what battle.Log.Roster is, and it is one
	// slice rather than two fields for a reason that is easy to get wrong.
	//
	// ⚠️ The order is load-bearing. atb.Queue.Add assigns seq off a counter in
	// the order battle.New is handed its roster, and seq is the last tie-break
	// in the turn order — so *the caller's slice order decides which side wins a
	// speed tie*, and that is worth up to sixty points in a mirror. Two fields,
	// ally and enemy, would be a second statement of an order the slice already
	// holds, and the peer would have to re-derive the enlistment by concatenating
	// them in a convention written down nowhere.
	Roster []battle.Roster `json:"roster"`
	// Side is the half of the board this client plays, and it changes between
	// battles: a match is fought both ways round. Compare Welcome.Seat, which
	// does not.
	Side hex.Side `json:"side"`
	// Battle is which battle of the series this is, counting from one.
	Battle int `json:"battle"`
}

// Kind is KindStart.
func (Start) Kind() Kind { return KindStart }

// Turn is one resolved turn. Server → client, every turn of the battle including
// the ones the client itself asked for.
//
// The decision is the whole of what the mirror applies: Replay takes a script of
// a single decision with a nil fallback, walks through whatever is forced after
// it, and hands back the prompt it stopped on. The digest is what makes the
// mirror a *check* rather than a hope — the client compares it against the
// digest of the events its own battle produced, so a divergence is loud on the
// turn it happens, with two digests to compare, rather than a board that quietly
// drifts.
//
// ⚠️ There is deliberately **no series-standing message**, and a reader will ask
// where it went. The client is a mirror: it learns the outcome of each battle
// from its own Ended event, and it already knows the series length from
// Welcome.Battles. A standing message would be a second declaration of a fact
// the client computes — and, worse, the one place two peers could disagree about
// who is winning while both of their battles agreed.
type Turn struct {
	// Decision is the action as it was taken, in the same form the log records
	// and Replay reads.
	Decision battle.Decision `json:"decision"`
	// Events is the digest of the events this turn produced on the server, in
	// the order it produced them.
	Events EventDigest `json:"events"`
}

// Kind is KindTurn.
func (Turn) Kind() Kind { return KindTurn }

// Closed is the room saying the match is over for a reason the board cannot
// show. Server → client, at most once, and **not to the peer that caused it**.
//
// It exists because one thing genuinely needed saying and nothing said it. A
// departure ends a match in the middle of a battle: there is no Ended for the
// battle in progress — the engine concluded nothing about it — and no further
// Start, so a mirror that was handed nothing would simply hang on its own open
// prompt, waiting for a turn that is never coming.
//
// ⚠️ **It is not "the match ended" in general**, and a reader will want to send
// it for every ending. A match played out to its end needs no message: the
// client learns each battle's outcome from its own Ended event and the series
// length from Welcome.Battles, so announcing the result would be a second
// declaration of a fact the client computes — which is exactly what the missing
// series-standing message is missing for. A **capped** battle needs none
// either: Welcome.TurnCap is what lets the mirror stop on the same turn by the
// same arithmetic.
//
// So the field is a Closure and the message is not one kind per reason: a
// further thing a board cannot show is an entry in that enum rather than a kind
// of its own. ClosureStopped and ClosureDraftExpired are what that has cost so
// far — two constants, two names, no new message — which is the evidence for the
// shape rather than a claim about it.
type Closed struct {
	Reason Closure `json:"reason"`
}

// Kind is KindClosed.
func (Closed) Kind() Kind { return KindClosed }
