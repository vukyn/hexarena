// The tests in this package are in room_test rather than room: everything the
// room does is observable through the messages it hands back and the accessors a
// transport reads, which is the whole claim of building it as a state machine.
// A test reaching inside would be measuring the implementation instead.
package room_test

import (
	"slices"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// fixturePassword is named rather than written into a struct literal, so a
// secret scanner reading this file finds a constant with an obvious purpose
// instead of a bare string beside a field called Password.
const fixturePassword = "the-cat-sat-on-the-mat"

// fixtureBuild is the build string this binary would announce. It is printed and
// never acted on, which is what the version test in internal/wire pins.
const fixtureBuild = "room-test"

// deps is the parsed data and the version a room is handed.
//
// ⚠️ The version carries the **real** data digest, unlike internal/wire's
// fixtures, and here that is right rather than fragile: the gate's job is to
// compare a peer's digest against this binary's, so a hand-made one would
// measure a comparison of two invented values. Nothing golden depends on it, so
// a balance commit moves it and no test notices.
func deps(t *testing.T) room.Deps {
	t.Helper()
	books, err := seed.Books()
	if err != nil {
		t.Fatalf("load the books: %v", err)
	}
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	version, err := wire.Local(fixtureBuild)
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	return room.Deps{Books: books, Characters: characters, Version: version}
}

// config is a room set up for a bo3 at 3v3 from one seed.
func config(seed uint64, battles int) room.Config {
	return room.Config{
		Format:    wire.Format3v3,
		Battles:   battles,
		Allowance: room.DefaultAllowance,
		Seed:      seed,
		TurnCap:   room.DefaultTurnCap,
	}
}

// draftingConfig is a room that bans and picks instead of taking squads.
//
// It is a **bo1** and not a parameter, because Config.Validate refuses a
// drafting series by name: what a draft means across three battles is three
// different games and nobody has chosen one. → Config.Drafts.
func draftingConfig(seed uint64) room.Config {
	out := config(seed, 1)
	out.Drafts = true
	return out
}

// newRoom is a room over the shipped data, so a test refuses to start rather
// than reporting a data problem as a room problem.
func newRoom(t *testing.T, cfg room.Config) *room.Room {
	t.Helper()
	opened, err := room.New(cfg, deps(t))
	if err != nil {
		t.Fatalf("open a room: %v", err)
	}
	return opened
}

// theThreeSlots are the formation cells the fixture squads stand in: one per
// rank, so a squad holds a front, a middle and a back. Reach is counted in
// occupied ranks from the far side, so a squad stacked in one column would be a
// board a range of one clears in a turn.
var theThreeSlots = []hex.Offset{{Col: 0, Row: 1}, {Col: 1, Row: 1}, {Col: 2, Row: 1}}

// squadOf builds a legal squad out of shipped characters, and it builds every
// part of "legal" itself rather than borrowing a shipped squad: a fixture that
// came out of internal/seed/data would move with the balance — the leaf rule and
// the level rule would then be measured against whatever an author had most
// recently saved. (It used to have a second reason, that the shipped squads were
// **two** units against this room's 3v3; they are three a side since #268, so
// the size no longer says anything and the drift argument is the whole of it.)
//
// The stage is named explicitly, off progression.Line.Leaves, rather than left
// absent: an absent stage means "the furthest the level reaches", and on a line
// that forks there is no such thing, so poliwag would refuse for the wrong
// reason. The kit is the first four skills and the first trait the level and
// form unlock, which is a legal loadout without being an interesting one.
func squadOf(t *testing.T, characters *cast.Book, id string, wanted ...string) placement.Squad {
	t.Helper()
	if len(wanted) > len(theThreeSlots) {
		t.Fatalf("squadOf was asked for %d units and has %d slots", len(wanted), len(theThreeSlots))
	}
	squad := placement.Squad{ID: id, Name: id}
	for index, name := range wanted {
		squad.Units = append(squad.Units,
			placeUnit(t, characters, name, theThreeSlots[index], progression.LevelCap))
	}
	return squad
}

// placeUnit is one legal placement of a character at a level, fielded as the
// first tip of its line.
func placeUnit(t *testing.T, characters *cast.Book, name string, slot hex.Offset, level int) placement.Placement {
	t.Helper()
	character, known := characters.Get(name)
	if !known {
		t.Fatalf("no character is called %q", name)
	}
	leaves, err := character.Stages.Leaves()
	if err != nil {
		t.Fatalf("the tips of %q: %v", name, err)
	}
	return stagedUnit(t, characters, name, leaves[0].Name, slot, level)
}

// stagedUnit is one placement fielded as a **named** form, with a kit that is
// legal for that form.
//
// It exists for the leaf rule, which is the one rule at the gate that turns on
// the stage alone: a squad differing from a legal one only in which form it
// fields is what tells "an interior stage is refused" apart from "the squad was
// wrong about something else". The kit is re-derived per form rather than
// carried over, because a learnset entry can be gated on a stage and a kit
// borrowed from the tip would then be refused for the loadout instead.
func stagedUnit(t *testing.T, characters *cast.Book, name, stage string, slot hex.Offset, level int) placement.Placement {
	t.Helper()
	character, known := characters.Get(name)
	if !known {
		t.Fatalf("no character is called %q", name)
	}
	return placement.Placement{
		// The unit's own id, which is unique within a squad rather than
		// globally: Take prefixes it with the side, so two squads holding the
		// same character still read apart in a log.
		ID:        name + "@" + stage,
		Character: name,
		Level:     level,
		Stage:     stage,
		Slot:      slot,
		Skills:    upTo(character.SkillsAt(level, stage), cast.SkillSlots),
		Passives:  upTo(character.PassivesAt(level, stage), cast.TraitSlots),
	}
}

func upTo(available []string, slots int) []string {
	if len(available) > slots {
		return available[:slots:slots]
	}
	return available[:len(available):len(available)]
}

// hello is a peer announcing itself with the version this binary would.
func hello(t *testing.T, squad placement.Squad, name string) wire.Hello {
	t.Helper()
	version, err := wire.Local(fixtureBuild)
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	return wire.Hello{Version: version, Squad: squad, Name: name}
}

// helloWithNoSquad is a peer joining a room that **drafts**, which is a peer
// that brought nobody: the side it will field is the side it is about to draft.
// The zero placement.Squad is what a client sends there, and a room that drafts
// refuses anything else with wire.CodeSquadUnwanted.
func helloWithNoSquad(t *testing.T, name string) wire.Hello {
	t.Helper()
	return hello(t, placement.Squad{}, name)
}

// mirror is a fake client, and it is a **mirror** in the sense the design record
// means: it holds its own *battle.Battle built from the seed and roster on
// wire.Start and steps it with the decisions on wire.Turn, so it computes the
// state by computing the battle rather than by being told it.
//
// Which is what makes it worth having as a fixture. A fake that merely recorded
// the messages would let every digest in the protocol pass unmeasured; this one
// hashes the events *its own* engine produced and compares them against the
// room's, every turn, so the headline test's claim — that the two are fighting
// the same battle — is checked rather than assumed.
//
// ⚠️ **It mirrors the DRAFT the same way**, and the second half of that sentence
// is the point: it holds its own *draft.Draft and steps it with the entries on
// wire.Drafted through draft.Draft.Apply — never with its own input, exactly as
// apply below steps the battle from the decision that came back rather than from
// what this client sent. So the draft the room recorded and the draft a client
// computed have to agree, and a test can compare the two by value.
type mirror struct {
	t          *testing.T
	seat       wire.Seat
	books      battle.Books
	characters *cast.Book
	// limit is how many turns one Replay may walk through, which is the client's
	// own business: a decision plus however many skipped turns follow it.
	limit int

	welcome  wire.Welcome
	side     hex.Side
	fight    *battle.Battle
	prompt   *battle.Prompt
	cursor   int
	starts   []wire.Start
	refusals []wire.Code
	closures []wire.Closure
	// turns is how many turns this client's own battle has opened, skipped ones
	// included, and capped is that count having passed the cap the room told it
	// about on wire.Welcome.
	//
	// ⚠️ This is the mirror doing the arithmetic the cap exists for rather than a
	// counter for a test to read. A capped battle emits no Ended — the engine
	// concluded nothing about it — and no further wire.Start arrives, so without
	// the cap on the welcome a client would sit holding the prompt it stopped on
	// for ever. Given the cap it stops on the same turn the room did, because it
	// counts the same thing: the room counts every Advance and every Advance
	// emits exactly one battle.TurnBegan.
	turns  int
	capped bool
	// compared is how many digests this client has checked, so a test can say
	// the check happened rather than hoping it did.
	compared int
	// events is everything this client's own battle has produced in the battle
	// in progress, which is what a renderer would be drawing.
	events []battle.Event

	// drafting is this client's own mirror of the ban and pick, built the moment
	// a welcome says the room drafts. Nil in a room that does not.
	drafting *draft.Draft
	// records is every wire.Drafted this client was sent, in arrival order, and
	// applied how many entries it replayed out of them.
	//
	// ⚠️ The two are **different counts on purpose**: the arrange phase records
	// two entries in one message, so a test asserting "one message a decision"
	// would be asserting something false, and a test asserting only the entry
	// count could not see an empty message at all. records is what makes "never
	// sent empty" measurable and applied is what makes "the mirror really ran"
	// measurable.
	records []wire.Drafted
	applied int
	// decided is every entry this client was sent, flattened in arrival order,
	// which is what "in that order" is compared against.
	decided []wire.DraftEntry
}

func newMirror(t *testing.T, seat wire.Seat, dependencies room.Deps, limit int) *mirror {
	return &mirror{
		t: t, seat: seat,
		books: dependencies.Books, characters: dependencies.Characters,
		limit: limit,
	}
}

// receive is the client's whole message loop.
func (m *mirror) receive(body wire.Body) {
	m.t.Helper()
	switch message := body.(type) {
	case wire.Welcome:
		m.welcome = message
		m.openDraft()
	case wire.Refused:
		m.refusals = append(m.refusals, message.Code)
	case wire.Start:
		m.open(message)
	case wire.Turn:
		m.apply(message)
	case wire.Drafted:
		m.replay(message)
	case wire.Closed:
		m.closures = append(m.closures, message.Reason)
	default:
		m.t.Fatalf("%s was sent a %T, which no server sends", m.seat, body)
	}
}

// openDraft is the client computing its own draft, out of the two facts a
// wire.Welcome gives it about one and nothing else.
//
// ⚠️ **Who decides first is COMPUTED here and is deliberately not on the wire.**
// It is the host, always, for bo1: a `first` field would be a second statement of
// a constant, and a second statement is the one place two peers can disagree. So
// this line and the room's own New have to agree by both being written from the
// same rule — which is the thing the whole-match test measures, since a client
// that computed the other seat would be refused on its very first ban.
// (⚠️ bo1's rule: "the previous winner bans first" is a real question for a bo3
// draft, and that is a different game and a later item.)
//
// The pool is built here from this client's **own** cast book, which is the
// mirror's whole shape: what makes the two pools equal is the data digest at the
// gate refusing a peer whose cast is not this cast, which is also why
// wire.Drafted carries no digest of its own.
func (m *mirror) openDraft() {
	m.t.Helper()
	if !m.welcome.Drafts {
		return
	}
	drafting, err := draft.New(draft.Config{
		Format: m.welcome.Format,
		Pool:   draft.NewPool(m.characters.All()),
		First:  wire.SeatHost,
	})
	if err != nil {
		m.t.Fatalf("%s builds its mirror of the draft: %v", m.seat, err)
	}
	m.drafting = drafting
}

// replay takes one batch of recorded decisions through this client's own draft,
// which is the whole of how a draft is mirrored: the entries are the only thing
// the room owes it, because a draft is a pure function of the decisions taken and
// the pool.
//
// ⚠️ It refuses an **empty** batch by name. A wire.Drafted carrying none is a
// room saying nothing happened, and the room must not send one — the first
// arrangement of the arrange phase records nothing, so this is a real path rather
// than a hypothetical. Held here so that every test driving a draft measures it
// on every message rather than only the one test that thought to look.
func (m *mirror) replay(drafted wire.Drafted) {
	m.t.Helper()
	if m.drafting == nil {
		m.t.Fatalf("%s was sent a draft record with no draft open, so it was never told the "+
			"room drafts", m.seat)
	}
	if len(drafted.Decisions) == 0 {
		m.t.Fatalf("%s was sent a wire.Drafted carrying no decisions, which is a room saying "+
			"nothing happened", m.seat)
	}
	for _, entry := range drafted.Decisions {
		if err := m.drafting.Apply(entry); err != nil {
			m.t.Fatalf("%s replays %s's %s: %v", m.seat, entry.Seat, entry.Step, err)
		}
		m.applied++
	}
	m.records = append(m.records, drafted)
	m.decided = append(m.decided, drafted.Decisions...)
}

// decide is what this client sends when the room asks it for a draft decision,
// read off its **own** draft — so a whole ban and pick plays out with nobody
// typing, and the room and the client agree about whose decision is due or the
// test fails.
//
// ⚠️ The sharpest thing it asserts, and it is asserted on every decision of every
// draft: **the seat the room is waiting on is the seat this client's own draft
// says is due.** Two independent readings of whose turn it is — one off the
// room's, one off a draft computed from the record alone.
func (m *mirror) decide() wire.Body {
	m.t.Helper()
	if m.drafting == nil {
		m.t.Fatalf("%s was asked to decide with no draft open", m.seat)
	}
	if m.drafting.Arranging() {
		pending := m.drafting.AwaitingArrangement()
		if !slices.Contains(pending, m.seat) {
			m.t.Fatalf("the room asked %s to arrange and its own draft is waiting on %v",
				m.seat, pending)
		}
		return m.sending(wire.DraftDecision{
			Step:  wire.StepArrange,
			Slots: firstCells(len(m.drafting.Picks()[seatIndexOf(m.t, m.seat)])),
		})
	}
	onTurn, step, due := m.drafting.Turn()
	if !due {
		m.t.Fatalf("the room asked %s to decide and its own draft is waiting on nobody", m.seat)
	}
	if onTurn != m.seat {
		m.t.Fatalf("the room asked %s to %s and its own draft is waiting on %s",
			m.seat, step, onTurn)
	}
	switch step {
	case wire.StepBan:
		return m.sending(wire.DraftDecision{
			Step: wire.StepBan, Character: m.candidate().ID,
		})
	case wire.StepPick:
		return m.sending(wire.DraftDecision{
			Step: wire.StepPick, Character: m.candidate().ID,
		})
	case wire.StepLoadout:
		side := m.drafting.Picks()[seatIndexOf(m.t, m.seat)]
		open := side[len(side)-1]
		form, skills, passives := legalKit(m.t, m.characters, open.Character)
		return m.sending(wire.DraftDecision{
			Step: wire.StepLoadout, Stage: form, Skills: skills, Passives: passives,
		})
	}
	m.t.Fatalf("the room asked %s for a %q, which is not a decision a seat can make",
		m.seat, step)
	return nil
}

// sending is one decision as the message a client sends, and it exists so that
// the fixture cannot accidentally build a wire.DraftEntry: **a client must not
// send its own seat**, and wire.Decide is the shape that has nowhere to put one.
func (m *mirror) sending(decision wire.DraftDecision) wire.Body {
	return wire.Decide{DraftDecision: decision}
}

// candidate is what a decision naming a character takes — the first the pool
// still offers — and the assertion that there was one to take. A draft
// draft.New allowed cannot run out of characters, so an empty list here is that
// proof failing rather than a case to handle.
func (m *mirror) candidate() cast.Character {
	m.t.Helper()
	candidates := m.drafting.Candidates()
	if len(candidates) == 0 {
		m.t.Fatalf("%s is due a decision and the pool offers nothing: a draft draft.New "+
			"allowed cannot run dry", m.seat)
	}
	return candidates[0]
}

// legalKit is *a* legal loadout for a character at the cap: a form named
// explicitly, the first four skills that form knows and the first trait.
//
// It names the form rather than leaving it absent because an absent form means
// "the furthest the level reaches", and on a line that **forks** — poliwag is in
// the shipped pool — there is no such thing, so a pick of one would be refused
// for a reason that has nothing to do with what is being measured.
//
// "The first four declared" is legal without being a decision, which is the whole
// reason builds.json exists; these tests are about the room's routing and not
// about the kit.
func legalKit(t *testing.T, characters *cast.Book, id string) (string, []string, []string) {
	t.Helper()
	character, known := characters.Get(id)
	if !known {
		t.Fatalf("the draft picked %q, which is not a character in this cast", id)
	}
	arms, err := character.FurthestAt(progression.LevelCap)
	if err != nil {
		t.Fatalf("the forms %s reaches at level %d: %v", id, progression.LevelCap, err)
	}
	form := arms[0].Name
	skills := character.SkillsAt(progression.LevelCap, form)
	if len(skills) == 0 {
		t.Fatalf("%s knows no skills at level %d as %s, so no loadout of it is legal",
			id, progression.LevelCap, form)
	}
	passives := character.PassivesAt(progression.LevelCap, form)
	return form,
		skills[:min(cast.SkillSlots, len(skills))],
		passives[:min(cast.TraitSlots, len(passives))]
}

// firstCells is the cheapest legal arrangement of n picks: the first n cells of a
// side's own formation, which is what every test that is not about the cells
// themselves wants.
func firstCells(n int) []hex.Offset {
	out := make([]hex.Offset, 0, n)
	for at := range n {
		out = append(out, hex.Offset{Col: at / hex.FormationRows, Row: at % hex.FormationRows})
	}
	return out
}

// seatIndexOf is a seat's position in the arrays internal/draft indexes by seat,
// which is the order a room hands seats out: the host, then the guest.
func seatIndexOf(t *testing.T, seat wire.Seat) int {
	t.Helper()
	for index, candidate := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		if candidate == seat {
			return index
		}
	}
	t.Fatalf("%q is not one of the two seats a room hands out", seat)
	return -1
}

// open builds this client's own battle from the seed and the roster, and walks
// it to the first turn that needs a decision — exactly what the room does, which
// is why the cursor starts after the opening board rather than at nought.
func (m *mirror) open(start wire.Start) {
	m.t.Helper()
	if m.welcome.Seat != m.seat {
		m.t.Fatalf("%s was started before it was welcomed", m.seat)
	}
	fight, err := battle.New(m.books, start.Seed, start.Roster)
	if err != nil {
		m.t.Fatalf("%s builds its mirror of battle %d: %v", m.seat, start.Battle, err)
	}
	fight.Begin()
	_, prompt, err := fight.Replay(nil, m.limit, nil)
	if err != nil {
		m.t.Fatalf("%s opens battle %d: %v", m.seat, start.Battle, err)
	}
	m.fight, m.prompt, m.side = fight, prompt, start.Side
	m.cursor, m.events = fight.Recorded(), nil
	// ⚠️ The opening's own turns are counted from the whole record and not from
	// the cursor. The cursor deliberately starts *after* the opening, because the
	// first digest exchanged covers the first decision rather than the first
	// decision plus the board — so a client counting only what arrives on a
	// wire.Turn would sit one turn behind the cap for the whole battle.
	opening, _ := fight.Since(0)
	m.turns, m.capped = 0, false
	m.count(opening)
	m.starts = append(m.starts, start)
}

// count adds a run of events' turns to this client's own tally and stops it at
// the cap the room named.
func (m *mirror) count(events []battle.Event) {
	for _, event := range events {
		if event.Kind == battle.TurnBegan {
			m.turns++
		}
	}
	if m.welcome.TurnCap > 0 && m.turns > m.welcome.TurnCap {
		m.capped = true
	}
}

// apply takes one decision and checks the digest.
func (m *mirror) apply(turn wire.Turn) {
	m.t.Helper()
	if m.fight == nil {
		m.t.Fatalf("%s was sent a turn with no battle open", m.seat)
	}
	_, prompt, err := m.fight.Replay(battle.Script{turn.Decision}, m.limit, nil)
	if err != nil {
		m.t.Fatalf("%s applies %q's turn: %v", m.seat, turn.Decision.Unit, err)
	}
	m.prompt = prompt
	events, next := m.fight.Since(m.cursor)
	m.cursor = next
	m.events = append(m.events, events...)
	m.count(events)
	digest, err := wire.DigestEvents(events)
	if err != nil {
		m.t.Fatalf("%s digests %q's turn: %v", m.seat, turn.Decision.Unit, err)
	}
	if digest != turn.Events {
		m.t.Fatalf("%s diverged on %q's turn %d: the room's events digest %s, its own %s",
			m.seat, turn.Decision.Unit, turn.Decision.Turn, turn.Events.Short(), digest.Short())
	}
	m.compared++
}

// answer is what this client sends when the room asks it: the rating's choice,
// read off its own mirror of the battle, which is what makes a whole match play
// out deterministically with nobody typing.
func (m *mirror) answer() wire.Body {
	m.t.Helper()
	if m.prompt == nil {
		m.t.Fatalf("%s was asked to act with no turn open", m.seat)
	}
	// The other half of the cap being on the welcome: the room stops asking on
	// the turn this client stops at, so being asked past it is a divergence
	// rather than a turn to take. The room checks the cap after the skipped test
	// and never leaves a capped turn open, so this is reachable only if the two
	// arithmetics disagree.
	if m.capped {
		m.t.Fatalf("%s was asked to act on turn %d, past the %d-turn cap it was told about",
			m.seat, m.turns, m.welcome.TurnCap)
	}
	unit, known := m.fight.Unit(m.prompt.Unit)
	if !known {
		m.t.Fatalf("%s was offered %q, which is not in its mirror", m.seat, m.prompt.Unit)
	}
	// The sharpest thing this fixture asserts, and it is asserted on every turn
	// of every battle: the seat the room is waiting on is the seat whose *own*
	// battle says it is that side's turn. Two independent readings of whose turn
	// it is, one from the room's seat map and one from the client's engine.
	if unit.Side != m.side {
		m.t.Fatalf("the room asked %s to act for %q, which its mirror has on the %s side while it plays %s",
			m.seat, unit.ID, unit.Side, m.side)
	}
	choice, ok := m.fight.Suggest(m.prompt)
	if !ok {
		return wire.Pass{}
	}
	return wire.Act{Skill: choice.Skill, Aim: hex.At(choice.Aim)}
}

// table is the two clients of a room, so a test can hand a message to the seat
// it is addressed to without a map whose iteration would reach the roster.
type table struct {
	host, guest *mirror
}

func newTable(t *testing.T, dependencies room.Deps, limit int) *table {
	return &table{
		host:  newMirror(t, wire.SeatHost, dependencies, limit),
		guest: newMirror(t, wire.SeatGuest, dependencies, limit),
	}
}

func (c *table) at(seat wire.Seat) *mirror {
	switch seat {
	case wire.SeatHost:
		return c.host
	case wire.SeatGuest:
		return c.guest
	}
	return nil
}

// deliver hands every outbound message to the client it names.
func (c *table) deliver(t *testing.T, out []room.Outbound) {
	t.Helper()
	for _, message := range out {
		client := c.at(message.To)
		if client == nil {
			t.Fatalf("the room addressed a %s to %q, which is not a seat", message.Body.Kind(), message.To)
		}
		client.receive(message.Body)
	}
}
