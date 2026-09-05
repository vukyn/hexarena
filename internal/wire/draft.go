package wire

import (
	"slices"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// The ban-and-pick on the wire: the vocabulary of a draft decision, the one
// message a client sends and the one the room sends back.
//
// ⚠️ **Why the decision's shape is declared HERE rather than in internal/draft,
// which is where a reader will look for it.** The precedent this package sets
// everywhere else is that wire *names* a domain type and the domain knows
// nothing about wire — Turn carries a battle.Decision and internal/core/battle
// imports this package zero times. That precedent is **not available** for the
// draft: internal/draft imports this package, for Format and Seat, so
// internal/wire naming a draft.Entry would be an import cycle. Moving Format and
// Seat down into a package both could import is a refactor of its own and is
// deliberately not this one (→ TODO.md, which carries the measured blast
// radius), so the layer relationship stays inverted for one package and the
// choice is only *where the shape is declared once*.
//
// It is declared once, here, and internal/draft **names it** — there is no local
// alias and no local Entry or Step type at all, which is the same relationship
// Format and Seat already have with that package: one spelling, and it is the
// protocol's. So there is no conversion, nothing to keep in step and no test owed
// for keeping it. The alternative was a shape at each end plus a field-by-field
// test holding them equal, and a test is a weaker guarantee than there being one
// struct.
//
// ⚠️ **This paragraph said "internal/draft's Entry is an alias of DraftEntry"
// and no such alias was ever written** — internal/draft/record.go's own comment
// says in as many words that there is none, for the reason there is none for
// Seat. The claim was written from the plan rather than from the code and is
// corrected here rather than left as a name a reader would go looking for.
//
// ⚠️ **The two directions genuinely need two bodies, and only the Seat differs.**
// A client must not send its own seat — the room knows which connection spoke,
// exactly as it knows whose turn it is, which is why Act carries no unit — and a
// recorded decision must carry one, because the record is what a mirror replays
// and the arrange phase records both seats at once. So DraftDecision is the six
// facts a decision is, Decide is that and nothing else, and DraftEntry is that
// **plus** the seat, by embedding rather than by repetition. Both embeds are
// anonymous, so both bodies serialise flat: the fields a reader sees in the
// golden are the fields of one struct.

// DraftStep is which kind of decision a draft decision is, and it doubles as the
// vocabulary of what a draft's record can hold.
//
// It is a named string rather than an iota with a table of names, which is the
// cheaper end of the rule Kind and Code live under: a string type has no
// declaration order for an insertion to reinterpret, so it needs no MarshalJSON,
// no names table and no count to be held against. Nothing dispatches on it by
// index. → Seat, which is a string for the same reason.
//
// The zero DraftStep is not a step: an absent step is not a decision this
// protocol has, and DraftStep.Valid is the question to ask.
type DraftStep string

const (
	// StepBan is a side taking a character out of the pool for the match. It is
	// optional, and a ban that names **no** character is the skip — see
	// DraftDecision.Character, where that absence is the whole of how a skip
	// travels.
	StepBan DraftStep = "ban"
	// StepPick is a side taking a character for itself, which is the first of the
	// two decisions a pick is made of. The character leaves the pool here.
	StepPick DraftStep = "pick"
	// StepLoadout is the second: the form, four skills and one trait the
	// character just picked will field.
	StepLoadout DraftStep = "loadout"
	// StepArrange is a side putting its picks on its own 3x3 formation, which is
	// the phase that runs once picking closes — privately and simultaneously.
	//
	// ⚠️ **Two of these arrive in one Drafted and neither arrives alone**, which
	// is the one place this vocabulary is not one message per decision as it is
	// taken: a recorded decision is public the moment it is sent, so sending the
	// first arrangement on its own would show it to the other player. →
	// draft.Draft.Arrange.
	StepArrange DraftStep = "arrange"
	// StepTimeout is not a decision anybody made: it is an allowance having run
	// out, which cancels the whole draft. It travels because it is one of the
	// things a draft's record holds, and a second vocabulary beside this one
	// would be two ways to say one idea.
	StepTimeout DraftStep = "timeout"
)

// DraftSteps is every step a draft decision may name, in the order a draft
// reaches them.
//
// It exists so a caller can **walk** the vocabulary rather than range over a
// table of its own and ask it whether it holds what it holds — the shape
// KindCount and CodeCount take for the two enums that do have a declaration
// order. What it buys here is that draft.Draft.Apply's switch can be held total
// against it, so a sixth step is a red test rather than a decision a mirror
// silently drops.
func DraftSteps() []DraftStep {
	return []DraftStep{StepBan, StepPick, StepLoadout, StepArrange, StepTimeout}
}

// Valid reports whether the step is one a draft decision may name. It is derived
// from DraftSteps rather than written as a switch, so the list and the gate
// cannot come to disagree.
func (s DraftStep) Valid() bool { return slices.Contains(DraftSteps(), s) }

// DraftDecision is one draft decision as it was taken, step-tagged, with the
// fields the other steps do not use left empty.
//
// ⚠️ **One shape with a Step rather than five shapes, and the reason is not
// economy.** A ban, a skip, a pick, a loadout, an arrangement and a timeout are
// the six things one record holds, and a draft's record is a single sequence —
// five bodies against one record would be two different answers to one question,
// and the switch that turned one into the other would be a second declaration of
// the draft's own sequence. The same argument keeps a `Skipped` flag off this
// struct: a ban that names nobody is the skip, so a flag beside Character would
// be a second statement of one fact and two fields that could disagree.
//
// ⚠️ **Nothing derived is on it.** The remaining pool, whose turn it is, both
// sides' picks and the form a loadout *resolved* to are all computed by a mirror
// from these decisions, exactly as a battle's state is computed from its
// decisions. A field holding one of them would be the one place two peers could
// come to disagree.
type DraftDecision struct {
	// Step is which of the five things this decision is. → DraftSteps, which is
	// the list, and Valid, which is derived from it.
	Step DraftStep `json:"step"`
	// Character is the id banned or picked.
	//
	// ⚠️ **Absent on a StepBan is the skip**, which is why this is omitempty
	// rather than always written: a ban that names nobody is a ban slot spent on
	// nobody, there is no third state, and the absence *is* the decision. Absent
	// on a loadout and a timeout because neither names a character — a loadout's
	// is the pick's, one decision earlier.
	Character string `json:"character,omitempty"`
	// Stage is the form a loadout **named**, and its absence is a real answer
	// rather than a missing one: the empty string is progression.Furthest, "the
	// furthest the level cap reaches", which is what a line that does not fork
	// means. A line that forks has no such thing and must name an arm.
	//
	// ⚠️ It is what was named and never what it resolved to. The resolution is
	// the mirror's to compute, from this.
	Stage string `json:"stage,omitempty"`
	// Skills and Passives are the loadout as it was named, before
	// cast.ChooseLoadout was asked about it. Empty on every other step.
	//
	// ⚠️ An **absent** list and an **empty** one are one decision here, which is
	// what omitempty makes them: cast.ChooseLoadout reads neither as a request
	// for anything, so a peer that sent `[]` gets back nothing and has lost
	// nothing. TestAnEmptyLoadoutListTravelsAsAnAbsentOne measures that rather
	// than leaving it to be assumed.
	Skills   []string `json:"skills,omitempty"`
	Passives []string `json:"passives,omitempty"`
	// Slots is one side's whole arrangement: the cell each of its picks stands
	// on, in pick order. Empty on every other step.
	//
	// ⚠️ It is one side's **whole** arrangement rather than one cell, because the
	// phase is simultaneous and secret — a cell at a time would need the same
	// buffer and would additionally hand a peer a partial arrangement to be told
	// about.
	Slots []hex.Offset `json:"slots,omitempty"`
}

// Decide is a client taking its draft decision. Client → server.
//
// ⚠️ **It carries no seat**, and that is the same rule Act is written under: the
// room knows which connection spoke, so a seat here would be a second statement
// of a fact one side already owns and the only thing it could add is a
// disagreement to resolve. It is also why this is not a DraftEntry with the seat
// left blank — a field a sender is asked not to fill in is a field somebody will
// fill in.
//
// ⚠️ **It is a request and not a record.** The room hands it to its own draft as
// a call, and what the draft records is the draft's own entry — so a field this
// message carries that its step has no use for is dropped on the way in rather
// than being written down. Compare Turn, which carries the decision **as
// recorded**.
//
// ⚠️ **There is deliberately no sequence number on it, and the step is what
// stands in for one.** A reader who knows the client's chooser will ask: that
// answer carries the turn it is for, because a buffered answer for a turn
// already spent is otherwise indistinguishable from a fresh one (→ CLAUDE.md,
// "A window is not closed by the fact that you have not opened it"). A draft has
// exactly one open decision, so the step tags a stale decision from a previous
// *stage* on its own — a ban arriving while a pick is due is refused by its own
// step. What it does not tell apart is a decision stale within one step, a
// second ban meant for the first ban slot, and that is the hazard Act already
// carries for a battle rather than a new one this message adds. → TODO.md.
type Decide struct {
	DraftDecision
}

// Kind is KindDecide.
func (Decide) Kind() Kind { return KindDecide }

// DraftEntry is one decision as the draft recorded it: who took it, and what it
// was.
//
// The seat is a field of its own and the decision is embedded, so this
// serialises flat and reads as one struct — and, more to the point, the six
// facts a decision is are declared exactly once, in DraftDecision, for both
// directions and for internal/draft's record — which holds a run of *these*
// rather than of a shape of its own. → the file comment, and the ⚠️ there about
// the alias this sentence used to claim.
//
// ⚠️ On a StepTimeout the seat is the one whose allowance ran out, which is the
// seat that was being asked — and during the arrange phase that is any seat that
// had not arranged, since both are being asked at once.
type DraftEntry struct {
	// Seat is who decided.
	Seat Seat `json:"seat"`
	DraftDecision
}

// Drafted is what a draft recorded, in the order it recorded it. Server →
// client, and it is the draft's twin of Turn.
//
// ⚠️ **It is a batch rather than one decision a message, for two reasons and
// each is sufficient.** The arrange phase records **two** entries at once by
// design — both arrangements or neither, because an entry is public the moment
// it is sent — so a message carrying one could not express the phase at all. And
// a spectator joining a draft in progress is a consumer whose cursor is nought,
// which is the whole record in one message; a per-decision message would make
// catching up a count of messages rather than a read.
//
// ⚠️ **There is no digest on it, and this is where that question gets asked.** A
// reader who knows Turn will look for one: Turn carries the digest of the events
// its decision produced, which is what makes a mirror a *check* rather than a
// hope, because a battle's state is a large computation that can drift silently.
// A draft's state is not that. It is a pure function of the decisions **and the
// pool**, the pool is the cast minus every character held back, and the data
// digest already gates that at the join — a peer whose cast is not this cast is
// refused with CodeDataMismatch before it is seated at all. So a per-decision
// digest could catch nothing the join does not already refuse, and it would be a
// second reading of the same fact taken once a turn.
type Drafted struct {
	// Decisions is the run of recorded decisions this message carries, in record
	// order. A Drafted carrying none is a room saying nothing happened, so a
	// room must not send one.
	Decisions []DraftEntry `json:"decisions"`
}

// Kind is KindDrafted.
func (Drafted) Kind() Kind { return KindDrafted }
