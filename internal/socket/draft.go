package socket

import (
	"fmt"
	"slices"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/wire"
)

// The client's half of the ban and pick: this client's own mirror of the draft,
// a consistent reading of it for a screen, and the one message it sends.
//
// ⚠️ **A drafting room was unjoinable by any real client until this file
// existed, and it is worth knowing how that was possible.** Mirror.Receive had
// arms for five messages and a default answering *"was sent a X, which no server
// sends"*, so a wire.Drafted **errored the client and tore the connection down**
// — while every test of the room's draft passed, because they drive the room with
// in-process fakes rather than through socket.Client. A fake that mirrors a draft
// measures internal/draft and internal/room and says nothing at all about whether
// the transport can carry one.
//
// ⚠️ **The state machine is not here and must not be restated here.**
// internal/draft owns whose decision is due, what the pool has left and every
// refusal about it; draft.Draft.Apply is this repository's one declaration of the
// draft's own sequence, and this file is its second caller — internal/room being
// the first. That is what makes a client's computed draft comparable to the
// room's, rather than two switches that agree today.
//
// ⚠️ **Nothing here hands a pointer out from under the mirror's lock.** →
// DraftSight, which is the whole of why this file has as many types in it as it
// does.

// DraftDue names one decision of a draft, and it is what an answer has to say it
// is for.
//
// ⚠️ **The discriminator is the decision and never the moment.** CLAUDE.md §
// *Mistakes already made here* carries this in full: the chooser's answer slot
// was drained on entry, on the premise that nothing could be in it for a turn
// that had not been asked about yet — but a client learns whose turn it is from
// its own mirror, a message and a redraw earlier, so a player answering off the
// board already in front of them landed in the slot first. The drain ate a real
// decision, the screen had recorded the turn as answered and would not offer it
// again, and both ends stood still for a whole allowance. The fix was that the
// answer names its turn and the chooser asks. A draft decision needs the same.
//
// It is four scalars so that it is **comparable**: DecideDraft's whole check is
// `answer.For == prompt.Due`, and a struct carrying a slice could not be compared
// with one operator.
//
// ⚠️ **Recorded is what closes the hazard the step alone leaves open, and it is
// deliberately local.** wire.Decide carries no sequence number and says why: a
// draft has exactly one open decision, so the step tags a decision stale across a
// *stage* on its own — a ban arriving while a pick is due — and what it cannot
// tell apart is one stale **within** a step, a second ban meant for the first ban
// slot. Two of one seat's ban slots have the other seat's between them, so the
// record has grown and this number has moved. It never travels: the room holds
// its own record and has no use for a client's count of it, so this is routing
// between a screen and its own chooser rather than protocol content.
type DraftDue struct {
	// Seat is whose decision it is. It is this client's own wherever a chooser is
	// handed one, because DraftAsking answers nothing for the other seat — it is
	// carried so that a Due is a whole name for a decision rather than half of
	// one.
	Seat wire.Seat
	// Step is which of the kinds it is. → wire.DraftStep.
	Step wire.DraftStep
	// Character is what the step is about where it is about a character already
	// chosen: the pick a loadout is owed for. Empty on a ban, a pick and an
	// arrangement, none of which is about one.
	//
	// ⚠️ A loadout's own wire.DraftDecision cannot carry this — it names no
	// character, because the pick one decision earlier is what named it — which is
	// half of why an answer needs a Due beside its decision rather than only the
	// decision's own Step.
	Character string
	// Recorded is how many decisions the draft had recorded when this one was
	// raised. → the ⚠️ above, which is the whole reason it is here.
	Recorded int
}

// DraftPrompt is the open draft decision as the client being asked for it sees
// it: which decision, and what there is to choose from.
//
// It is the draft's twin of *battle.Prompt and it is a **value** for the reason
// every field of DraftSight is one: a prompt read out from under the mirror's
// lock and kept would be a reading of a draft the Play goroutine is about to
// step.
type DraftPrompt struct {
	// Due names the decision, and is what an answer must say it is for.
	Due DraftDue
	// Candidates is what a ban or a pick may name: the pool minus every character
	// already banned or picked, in the pool's own order.
	//
	// Nil for a loadout and for an arrangement, neither of which chooses a
	// character — which is draft.Draft.Candidates' own answer, unchanged, and its
	// own comment says why nil is a different answer from an empty list.
	Candidates []cast.Character
	// Mine is this seat's own picks in the order it took them, which is what the
	// two decisions that are not about the pool need: a loadout is owed for the
	// last of them, and an arrangement names one cell per pick, so `Slots[i]` is
	// the cell for `Mine[i]`.
	Mine []draft.Pick
}

// DraftAnswer is a draft chooser's answer: the decision it is for, and the
// decision itself.
//
// ⚠️ **For is not derivable from Decision, and that is the point rather than an
// inconvenience.** A wire.DraftDecision carries its own Step, so an answer for
// the wrong stage could be caught without this — but nothing on a decision counts
// the record and a loadout's names no character, so an answer stale within one
// step is indistinguishable from a fresh one unless it says which decision it was
// given for. → DraftDue.
type DraftAnswer struct {
	// For is the decision this answers, which a chooser copies off the prompt it
	// was handed.
	For DraftDue
	// Decision is the answer. Its Step has to be the one For names: this is one
	// answer to one question, and an answer disagreeing with itself about which
	// question is refused here rather than by the room.
	Decision wire.DraftDecision
}

// DraftChooser is what decides a draft decision, and it is a different question
// from battle.Chooser asked in a different vocabulary: a ban, a pick, a loadout
// or an arrangement, out of a pool rather than off a board.
//
// ⚠️ **A false is a failure and not a pass.** A battle has wire.Pass; a draft has
// nothing of the kind. A ban that names nobody is a **skip**, which is a decision
// somebody takes (→ draft.Draft.SkipBan) and not one this package may invent on
// a player's behalf, and a draft whose allowance runs out is cancelled outright
// with no auto-pick at all (→ draft.Draft.TimedOut). So Play reports a false
// rather than sending anything, which is the loud ending the whole timeout
// mechanism exists to have instead of a client sitting on a decision for ever.
type DraftChooser func(DraftPrompt) (DraftAnswer, bool)

// DraftSight is a consistent reading of this client's own mirror of the draft, by
// value.
//
// ⚠️ **It is a snapshot and never the *draft.Draft, and the decisive reason is
// the import graph rather than the lock.** internal/screen — where the screen
// that draws this lives — imports internal/core/battle and does **not** import
// internal/draft, which could not import it either way: internal/draft imports
// internal/wire, so pulling it into the package two clients share would drag the
// protocol in behind it. That is the same constraint that keeps the lobby's three
// screens in cmd/hexarena-tui. So a *draft.Draft handed out here would be a
// pointer no renderer could name, whatever the locking said.
//
// The lock is the second reason and it is real but weaker: every field here is a
// copy, and internal/draft is what makes that cheap to be sure of — Picks,
// Squads, Candidates and AwaitingArrangement each build a fresh slice, so nothing
// here can alias the draft the Play goroutine is stepping.
//
// ⚠️ **This paragraph said "a pointer handed out from under an RWMutex is a shape
// this package has already paid for twice", and that overstates it** — it came
// from the brief for this step and is corrected here rather than repeated. What
// this package paid for twice was holding the lock **across a callback** (which
// deadlocks against the next writer → Decide, DecideDraft) and a **drain that ate
// a real answer** (→ CLAUDE.md § *Mistakes already made here*). Neither is a
// pointer escaping, and Sight.Fight is proof that one may.
//
// ⚠️ **Sight.Fight is the deliberate exception and stays one.** A renderer
// computes the board by computing the battle, so the *battle.Battle has to be
// handed over and Mirror.Read's comment is what holds the rule about it. A draft
// has no such computation: what a screen draws is the pool, the picks and whose
// decision is due, which is exactly what this carries.
type DraftSight struct {
	// Mirrored is this client holding a draft at all, which is Welcome.Drafts
	// having said the room drafts. Every field below means nothing without it.
	Mirrored bool
	// OnTurn and Step are whose decision is due and which kind, and Open whether
	// one is — draft.Draft.Turn's three answers, unchanged.
	//
	// ⚠️ Open is **false during the arrange phase**, and that is not this reading
	// being incomplete: Turn answers one seat and the phase has two pending at
	// once, so the phase is asked about through Arranging and Awaiting instead. →
	// draft.Draft.Turn, which says why it never widened for it.
	OnTurn wire.Seat
	Step   wire.DraftStep
	Open   bool
	// Asking is the open decision when it is **this** client's to answer, and
	// Asked whether it is — the draft's twin of Sight.Asking, and the whole of how
	// a screen knows to offer a choice. It covers the arrange phase, where Open is
	// false and this client may still be one of the two being asked.
	Asking DraftPrompt
	Asked  bool
	// Candidates is what the open decision may choose from, **whosever it is**: a
	// screen draws the remaining pool while the other side thinks, so this is not
	// Asking.Candidates narrowed to this client's own turns. When Asked it is the
	// same list the prompt holds rather than a second reading of it.
	Candidates []cast.Character
	// Picks is both sides' picks so far, indexed by seat in the order a room hands
	// seats out — **[0] is wire.SeatHost and [1] is wire.SeatGuest** — which is
	// draft.Draft.Picks' own indexing. A pick with no skills is one whose loadout
	// is still open.
	Picks [2][]draft.Pick
	// Arranging is the arrange phase being open, and Awaiting which seats have
	// still to arrange, in seats order.
	Arranging bool
	Awaiting  []wire.Seat
	// Picked is the ban and pick played out, Done that and both arrangements in,
	// and Cancelled the draft having been abandoned. → draft.Draft.Picked, which
	// says why the first two are two questions and not one.
	Picked    bool
	Done      bool
	Cancelled bool
	// Squads is what a whole draft produces, indexed the way Picks is, and two
	// squads with **nobody in them** until Done — draft.Draft.Squads' own promise,
	// which is an honestly incomplete answer rather than a plausible one.
	Squads [2]placement.Squad
	// Replayed is how many recorded decisions this client has taken back through
	// its own draft, so a caller can say the mirror ran rather than hope it did —
	// the same argument Mirror.Compared is exposed under.
	Replayed int
}

// Draft is a consistent reading of this client's own mirror of the draft.
//
// → Read, which is how a renderer gets this and the battle in one reading, and
// which is the only safe way to look at more than one of them at once.
func (m *Mirror) Draft() DraftSight {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.draftSight()
}

// draftSight is Draft with the lock already held, so that Read can answer it
// without taking the read lock a second time. → over, for why every reading here
// comes in a locked and an unlocked spelling.
func (m *Mirror) draftSight() DraftSight {
	if m.drafting == nil {
		return DraftSight{}
	}
	prompt, asked := m.draftAsking()
	onTurn, step, open := m.drafting.Turn()
	// ⚠️ The candidate list is the **open** decision's, whosever it is, so it is
	// read again when this client is not the one being asked. When it is, the
	// prompt already holds the same list and asking twice would allocate a second
	// copy of one answer.
	candidates := prompt.Candidates
	if !asked {
		candidates = m.drafting.Candidates()
	}
	return DraftSight{
		Mirrored:   true,
		OnTurn:     onTurn,
		Step:       step,
		Open:       open,
		Asking:     prompt,
		Asked:      asked,
		Candidates: candidates,
		Picks:      m.drafting.Picks(),
		Arranging:  m.drafting.Arranging(),
		Awaiting:   m.drafting.AwaitingArrangement(),
		Picked:     m.drafting.Picked(),
		Done:       m.drafting.Done(),
		Cancelled:  m.drafting.Cancelled(),
		Squads:     m.drafting.Squads(),
		Replayed:   m.replayed,
	}
}

// DraftAsking is the open draft decision when it is **this** client's to answer,
// and it is the draft's twin of Asking: the whole of how a client knows a
// decision has come to it.
//
// ⚠️ Nothing on the wire says so, exactly as nothing on the wire says whose turn
// it is in a battle. The room sends the decisions and each client derives the
// open one from its own replayed draft.
func (m *Mirror) DraftAsking() (DraftPrompt, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.draftAsking()
}

// draftAsking is DraftAsking with the lock already held. → over.
//
// ⚠️ **The arrange phase is asked about separately, because Turn answers one seat
// and the phase has two.** That is draft.Draft.Turn's own note: Arranging is the
// phase's accessor and AwaitingArrangement is who it is still waiting on, so a
// client that read Turn alone would sit through the whole phase being asked
// nothing and never send its formation.
func (m *Mirror) draftAsking() (DraftPrompt, bool) {
	if m.drafting == nil {
		return DraftPrompt{}, false
	}
	mine := m.ownPicks()
	if m.drafting.Arranging() {
		if !slices.Contains(m.drafting.AwaitingArrangement(), m.seat) {
			return DraftPrompt{}, false
		}
		return DraftPrompt{
			Due:  DraftDue{Seat: m.seat, Step: wire.StepArrange, Recorded: m.replayed},
			Mine: mine,
		}, true
	}
	onTurn, step, due := m.drafting.Turn()
	if !due || onTurn != m.seat {
		return DraftPrompt{}, false
	}
	// A loadout is the one step that is about a character already chosen, and it
	// is whoever this seat picked last — derived rather than stored, which is the
	// same derivation draft.Draft.Turn makes of a loadout's owner.
	subject := ""
	if step == wire.StepLoadout && len(mine) > 0 {
		subject = mine[len(mine)-1].Character
	}
	return DraftPrompt{
		Due:        DraftDue{Seat: m.seat, Step: step, Character: subject, Recorded: m.replayed},
		Candidates: m.drafting.Candidates(),
		Mine:       mine,
	}, true
}

// ownPicks is this client's own side of draft.Draft.Picks, and nil for a seat the
// draft does not hold — which is a mirror nobody has welcomed yet.
func (m *Mirror) ownPicks() []draft.Pick {
	index, seated := seatIndex(m.seat)
	if !seated {
		return nil
	}
	return m.drafting.Picks()[index]
}

// DecideDraft is the message this client sends for the open draft decision, read
// off its own draft by the given chooser.
//
// ⚠️ It **applies nothing**, for the reason Decide applies nothing: a mirror
// steps its draft from the wire.Drafted that comes back rather than from its own
// input, so that the state it computes comes out of the same call on both sides —
// which is why the room sends every recorded decision to both clients including
// the one that asked for it. Deciding locally and applying locally would be two
// paths into one draft.
//
// It reports false four ways: nothing is due, there is no chooser, the chooser
// declined, and the answer names a decision that is not the open one.
//
// ⚠️ **The read lock is taken to find the decision and RELEASED before the
// chooser is called, and that ordering is load-bearing rather than a copy of
// Decide's shape.** An RWMutex held across a callback deadlocks against the next
// **writer**: Go queues a waiting writer ahead of new readers, so the next
// Receive blocks behind the held read lock and the renderer then blocks behind
// the writer — and the renderer is what the player is waiting to see before they
// can choose. Note what that means for a test: two readers sit happily beside
// each other, so a version built on "two readers cannot both be in" passes with
// the lock held. → Decide, which carries the measurement, and
// TestDecideDraftDoesNotHoldTheLockAcrossTheChooser, which builds the same
// three-way for a draft.
//
// The prompt handed to the chooser outlives the lock, which is safe for the one
// reason Play is what calls this: nothing steps the draft between the read and
// the answer, because the goroutine that would is the one waiting here. And it is
// a **value**, so keeping it reaches nothing.
//
// ⚠️ **A stale answer is refused rather than applied to the next decision**, and
// it is counted so the refusal leaves a trace. → DraftDue, which is what an
// answer names and why naming it is not optional, and Stale.
func (m *Mirror) DecideDraft(choose DraftChooser) (wire.Body, bool) {
	m.mu.RLock()
	prompt, asking := m.draftAsking()
	m.mu.RUnlock()
	if !asking || choose == nil {
		return nil, false
	}
	answer, decided := choose(prompt)
	if !decided {
		// ⚠️ Nothing is invented here. → DraftChooser, on why a draft has no
		// wire.Pass for a false to become.
		return nil, false
	}
	if answer.For != prompt.Due || answer.Decision.Step != prompt.Due.Step {
		m.mu.Lock()
		m.stale++
		m.mu.Unlock()
		return nil, false
	}
	// ⚠️ wire.Decide and never a wire.DraftEntry: **a client must not send its own
	// seat**, and Decide is the shape that has nowhere to put one — the room knows
	// which connection spoke, exactly as it knows whose turn it is. → wire.Decide.
	return wire.Decide{DraftDecision: answer.Decision}, true
}

// Stale is how many draft answers this client refused for naming a decision that
// was not the open one.
//
// It is exposed for the reason Compared is, and counted for the reason table.late
// is: a refusal that sends nothing and closes nothing leaves no other trace at
// all, so a test asserting the path was taken has to be able to read it.
func (m *Mirror) Stale() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stale
}

// openDraft is the client computing its own draft, out of the two facts a
// wire.Welcome gives it about one and nothing else. It runs with the write lock
// already held, like every other private step behind Receive.
//
// ⚠️ **Who decides first is COMPUTED here and is deliberately not on the wire.**
// It is the host, always: a First field would be a second statement of a
// constant, and a second statement is the one place two peers can disagree. So
// this line and the room's own New agree by both being written from the same
// rule, which is what a whole drafting match measures — a client that computed
// the other seat is refused on its very first ban. (It is bo1's rule; *"the
// previous winner bans first"* is a real question for a bo3 draft, which
// room.Config.Validate refuses outright today, and draft.Config.First stays a
// parameter so that item can answer it differently.)
//
// ⚠️ **The pool is draft.NewPool over this client's OWN cast book**, which is the
// mirror's whole shape: what makes the two pools equal is the data digest at the
// gate refusing a peer whose cast is not this cast, which is also why
// wire.Drafted carries no digest of its own. And it is NewPool rather than the
// book itself — that function is the single declaration of "the cast minus every
// character held back", so a client that took the whole book would offer a
// held-back character its own screen could ban and the room would refuse.
func (m *Mirror) openDraft(welcome wire.Welcome) error {
	if m.characters == nil {
		return fmt.Errorf("%s was welcomed into a room that drafts and holds no cast book, so it "+
			"cannot build the pool a ban and a pick choose from", m.seat)
	}
	drafting, err := draft.New(draft.Config{
		Format: welcome.Format,
		Pool:   draft.NewPool(m.characters.All()),
		First:  wire.SeatHost,
	})
	if err != nil {
		return fmt.Errorf("%s builds its mirror of the draft: %w", m.seat, err)
	}
	m.drafting = drafting
	return nil
}

// drafted takes one batch of recorded decisions through this client's own draft,
// which is the whole of how a draft is mirrored: the entries are the only thing
// the room owes it, because a draft is a pure function of the decisions taken and
// its pool.
//
// ⚠️ **Every entry goes through draft.Draft.Apply in the order the batch carries
// them**, and the order is load-bearing rather than incidental — a pick before
// the bans are spent is refused, so a batch applied backwards is a mirror that
// has stopped being one. Most batches hold one entry and reverse to themselves;
// the arrange phase's pair and a spectator's catch-up from cursor nought are the
// two that do not.
//
// ⚠️ It refuses an **empty** batch by name, because a wire.Drafted carrying none
// is a room saying nothing happened and a room must not send one. That is a real
// path rather than a hypothetical: the first of the two arrangements records
// nothing, so exactly one decision of a draft is answered with no message at all.
//
// ⚠️ **It applies nothing of its own**, exactly as apply steps the battle from
// the wire.Turn that came back rather than from what this client sent. A client
// that applied its own decision locally would have two paths into one draft, and
// one path is the whole of what a mirror is for.
//
// Every refusal here is the state machine's own, unreworded, because Apply routes
// and does not judge. → draft.Draft.Apply.
func (m *Mirror) drafted(drafted wire.Drafted) error {
	if m.drafting == nil {
		return fmt.Errorf("%s was sent a draft record with no draft open, so it was never told "+
			"the room drafts", m.seat)
	}
	if len(drafted.Decisions) == 0 {
		return fmt.Errorf("%s was sent a wire.Drafted carrying no decisions, which is a room "+
			"saying nothing happened", m.seat)
	}
	for _, entry := range drafted.Decisions {
		if err := m.drafting.Apply(entry); err != nil {
			return fmt.Errorf("%s replays %s's %s: %w", m.seat, entry.Seat, entry.Step, err)
		}
		m.replayed++
	}
	return nil
}

// seatIndex is a seat's position in the arrays internal/draft indexes by seat,
// which is the order a room hands them out: the host, then the guest. It reports
// false for anything that is not one of the two — including the zero Seat, which
// means "no seat" and must not quietly mean the host.
//
// ⚠️ It is a third copy of three lines rather than a shared helper, for
// internal/draft's own stated reason: room's and draft's are both unexported, and
// what is worth sharing is the **rule** — that this order is the order seats are
// handed out, and that it reaches an output — which is why the rule is written out
// again here instead of being cited.
func seatIndex(seat wire.Seat) (int, bool) {
	for index, candidate := range [2]wire.Seat{wire.SeatHost, wire.SeatGuest} {
		if candidate == seat {
			return index, true
		}
	}
	return 0, false
}
