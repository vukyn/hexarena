package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"path/filepath"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// playScreen is a battle fought by hand against the opponent the engine plays.
//
// It is raised from the fight for the reason the fight is raised from the
// catalogue: that is where a pairing is already chosen, and a battle wants two
// squads before it wants anything else. What the simulation answers over two
// hundred battles, this answers once — and the two are worth having side by
// side, because a rate says which squad is better and playing one says why.
//
// ⚠️ **This is the one screen holding something the model does not copy.** Every
// other screen is a value the model carries, so a field written while drawing is
// thrown away with the copy; a battle is a pointer, and a mutation reaches every
// copy of the model there is. So the battle is stepped in update and **never**
// touched in view — view reads it and draws, and that division is what keeps a
// redraw from playing a turn.
type playScreen struct {
	// seed is which battle this is. Walking it is how a player asks for another
	// arrangement of the same pairing, and it is the same number the log carries,
	// so a battle played here can be replayed by the game client.
	seed uint64
	// side is the half the player is fighting on: the home squad's, always, so
	// that the squad under the catalogue's cursor is the one being played rather
	// than the one being played against.
	side hex.Side

	fight *battle.Battle
	// roster is what the battle was built from, kept because a log records it:
	// a log carrying the resolved placement is what makes it re-runnable across
	// a data edit, and asking the battle for it afterwards would be asking for
	// the units as they are now rather than as they were placed.
	roster []battle.Roster
	tags   map[string]string
	names  map[string]string
	events []battle.Event
	// script is every decision taken, the player's and the engine's alike. It is
	// what undo shortens and what the whole battle is rebuilt from, which is why
	// a rebuild needs nothing else: a battle is a pure function of its seed and
	// the decisions taken.
	script battle.Script
	// pending is the turn waiting on the player. Nil means the battle is between
	// turns, which is where the engine's own units act.
	pending *battle.Prompt

	// option and aim are the two questions a turn asks, and aiming says which of
	// them is in front.
	option int
	aim    int
	aiming bool

	// logFollow and logOffset are where the log's frame sits, and they are two
	// fields rather than one for the one reason that decides this whole feature.
	//
	// ⚠️ **Following the tail is a state, not an offset value, because the tail
	// moves.** The reader is normally looking at the newest rows; an offset
	// storing "the newest rows" would be a number that means something different
	// every time an event arrives, so every turn taken would silently shift what
	// is under the reader. So logOffset counts rows **from the start of the
	// history** — which is exactly what the position on the heading row states —
	// and following is carried **beside** it rather than encoded into it.
	//
	// This is the rule Suggest's abandoned queue tie-break paid for: Queue.Pending
	// answers 0 for a unit it has never heard of and 0 is *soonest*, so absence
	// had to be declared rather than detected. A sentinel offset meaning "the
	// tail" would be that mistake again — and it would read as working, because
	// the sentinel is a legal offset on the turn it is written.
	//
	// logOffset is meaningless while logFollow holds, and it is clamped against
	// the history's current length wherever it is read rather than only where it
	// is written: undo rebuilds the battle from a shortened script, so the
	// history it rebuilds is shorter and an offset kept across it can point past
	// the end.
	logFollow bool
	logOffset int

	err error
	// notes are what a write left behind, held as facts rather than as a
	// sentence so ctrl+l redraws them in the other language.
	notes []forge.Note
}

// playTurnLimit is where a battle is abandoned, and it is cmd/hexarena's number
// deliberately: a battle this screen calls endless and a battle the game calls
// endless have to be the same battle.
const playTurnLimit = 4000

func newPlayScreen() playScreen {
	// Following, because the newest rows are what a battle nobody has scrolled is
	// showing, and because the alternative would be an offset into a history that
	// does not exist yet.
	return playScreen{seed: 1, side: hex.SideAlly, logFollow: true}
}

// begin builds the battle from the pairing in front and runs it up to the
// player's first decision.
func (p playScreen) begin(m model) playScreen {
	p.fight, p.tags, p.names = nil, nil, nil
	p.roster = nil
	p.events, p.script, p.pending = nil, nil, nil
	p.option, p.aim, p.aiming = 0, 0, false
	p.logFollow, p.logOffset = true, 0
	p.err, p.notes = nil, nil

	home, away, ok := m.fight.sides(m)
	if !ok {
		return p
	}
	roster, err := home.Take(hex.SideAlly, m.lib.Characters())
	if err != nil {
		p.err = err
		return p
	}
	facing, err := away.Take(hex.SideEnemy, m.lib.Characters())
	if err != nil {
		p.err = err
		return p
	}
	placed := append(roster, facing...)
	fight, err := battle.New(m.lib.Books(), p.seed, placed)
	if err != nil {
		p.err = err
		return p
	}
	fight.Begin()
	p.fight = fight
	p.roster = placed
	p.tags = tui.Tags(fight.Units())
	p.names = tui.Names(fight.Units())
	p.collect()
	return p.run()
}

// collect drains whatever the battle has recorded since it was last asked.
//
// The event log is the only contract a reader of a battle has, and this screen
// is a reader like any other: a screen that summed its own strikes would be a
// second place where what a battle did is decided.
func (p *playScreen) collect() {
	if p.fight == nil {
		return
	}
	p.events = append(p.events, p.fight.Drain()...)
}

// run takes every turn that is not the player's, and stops on the one that is.
//
// A skipped turn is stepped past rather than shown: a unit that lost its action
// to control has no decision in it, and a screen that stopped there would ask a
// question with no answers.
func (p playScreen) run() playScreen {
	if p.fight == nil {
		return p
	}
	for steps := 0; steps < playTurnLimit; steps++ {
		if p.fight.Finished() {
			p.pending = nil
			return p
		}
		prompt := p.pending
		p.pending = nil
		if prompt == nil {
			opened, err := p.fight.Advance()
			if err != nil {
				p.err = err
				return p
			}
			prompt = opened
		}
		p.collect()
		if prompt.Skipped {
			continue
		}
		unit, known := p.fight.Unit(prompt.Unit)
		if !known {
			// A turn offered to somebody who is not fighting is stepped past
			// rather than reported. It cannot happen — the queue holds units the
			// battle enlisted — and a screen is the wrong place to discover that
			// it did; the step limit above is what stops this being a loop.
			continue
		}
		if unit.Side == p.side {
			p.pending = prompt
			p.option = p.firstAvailable(prompt)
			p.aim, p.aiming = 0, false
			return p
		}
		if err := p.engineOrder(prompt); err != nil {
			p.err = err
			return p
		}
	}
	return p
}

// firstAvailable is the option the cursor starts on: the first the unit may
// actually take, rather than the first declared. A cursor that opened on a skill
// on cooldown would be one press from a refusal on most turns.
func (p playScreen) firstAvailable(prompt *battle.Prompt) int {
	for index, option := range prompt.Options {
		if option.Available() {
			return index
		}
	}
	return 0
}

// engineOrder takes the turn the engine would take, which is how the opponent
// plays and how the "let it pick" key answers.
//
// It lands in the script as a decision like the player's own, because the script
// is what the whole battle is rebuilt from: a half of it that was not written
// down would replay as a different battle.
func (p *playScreen) engineOrder(prompt *battle.Prompt) error {
	choice, acted := p.fight.Suggest(prompt)
	if !acted {
		return p.skip(prompt, battle.NoActionReason)
	}
	return p.take(prompt, choice.Skill, choice.Aim)
}

// take and skip are the two things a turn can be spent on, and they are two
// methods rather than one taking a decision so that a decision with a skill and
// no aim — which the engine would refuse and nothing here should be able to
// build — cannot be written at all.
func (p *playScreen) take(prompt *battle.Prompt, skill string, aim hex.Offset) error {
	if err := p.fight.Act(skill, aim); err != nil {
		return err
	}
	return p.record(battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Skill: skill, Aim: hex.At(aim),
	})
}

func (p *playScreen) skip(prompt *battle.Prompt, reason string) error {
	decision := battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Passed: true, Reason: reason,
	}
	if err := p.fight.Pass(decision.PassReason()); err != nil {
		return err
	}
	return p.record(decision)
}

func (p *playScreen) record(decision battle.Decision) error {
	p.script = append(p.script, decision)
	// A turn taken puts the reader back on the live tail, and this is the one
	// place every turn goes through — the player's, the engine's, the one the
	// "let it pick" key hands over and the pass. Somebody who scrolled back to
	// read what happened and then acted would otherwise be reading a frame from
	// before their own decision, which is the one moment the log is certainly
	// stale. Undo and another seed reset it too, through begin.
	p.logFollow, p.logOffset = true, 0
	p.collect()
	return nil
}

// rewind rebuilds the battle from the seed and a shortened script.
//
// It is the whole of undo, and it works because a battle is a pure function of
// its seed and the decisions taken: there is no state to unwind, only a shorter
// list to replay. That is the same property the log's --verify rests on.
func (p playScreen) rewind(m model, script battle.Script) playScreen {
	seed, side := p.seed, p.side
	fresh := newPlayScreen()
	fresh.seed, fresh.side = seed, side
	fresh = fresh.begin(m)
	if fresh.fight == nil || fresh.err != nil {
		return fresh
	}
	replayed, pending, err := fresh.fight.Replay(script, playTurnLimit, nil)
	if err != nil {
		fresh.err = err
		return fresh
	}
	fresh.script = replayed
	fresh.pending = pending
	fresh.events = fresh.fight.Drain()
	return fresh.run()
}

// undo drops the player's last decision and everything that followed it.
func (p playScreen) undo(m model) playScreen {
	cut := -1
	for index := len(p.script) - 1; index >= 0; index-- {
		unit, known := p.fight.Unit(p.script[index].Unit)
		if known && unit.Side == p.side {
			cut = index
			break
		}
	}
	if cut < 0 {
		return p
	}
	shortened := make(battle.Script, cut)
	copy(shortened, p.script[:cut])
	return p.rewind(m, shortened)
}

func (p playScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Saving is asked before the switch because it answers to more than one
	// keystroke; isSaveKey is the single declaration of which.
	if isSaveKey(message) {
		p = p.save(m)
		m.play = p
		return m, nil
	}
	switch message.String() {
	case "esc":
		if p.aiming {
			p.aiming = false
			m.play = p
			return m, nil
		}
		m.screen = screenFight
		return m, nil
	case "n":
		p.seed++
		p = p.begin(m)
	case "u":
		if p.fight != nil {
			p = p.undo(m)
		}
	// The log's own keys, and they are answered **here** rather than under the
	// guard below: the log is drawn between turns and on a battle that has
	// finished, which is exactly when reading back through it is the only thing
	// left to do, and it is drawn while aiming as well — the same reason ? works
	// there. ↑/↓ are the option list's and may not be taken, so this is the pair
	// that already scrolls in this client (the trait description and the picker),
	// rather than a second vocabulary for one idea.
	//
	// [ and ] are aliases for that pair and not a second vocabulary either: they
	// are the same one idea reached by keys every keyboard has. A compact board
	// has no PgUp and no PgDn — a laptop reaches them through a modifier and a
	// sixty-percent board through a layer, and neither is discoverable from a
	// footer naming them — so on such a keyboard the whole log below the frame
	// was as unreachable as it was before it scrolled at all. The brackets are
	// added at **every** site the pair works (here, the trait description and the
	// picker's reading pane), because one site aliased is exactly the second
	// vocabulary the paragraph above refuses. Back is [ and forward is ], which
	// is the direction ↑/↓ reads in and the order the footers print them in.
	case "pgup", "[":
		p = p.scrollLog(m, -1)
	case "pgdown", "]":
		p = p.scrollLog(m, 1)
	}
	if p.pending == nil {
		m.play = p
		return m, nil
	}
	switch message.String() {
	case "up", "k":
		p = p.move(-1)
	case "down", "j":
		p = p.move(1)
	case "?":
		// The full description of the option under the cursor, on the screen the
		// rest of this tool already raises with this key. It is free here and it
		// costs no turn: the blurb reads the option and nothing else, so raising
		// it and leaving it cannot step the battle.
		//
		// Reached with a prompt open, which is what the guard above this switch
		// buys, and it works while aiming too — the skill is chosen and the cell
		// is not, so "what does this do" is still the live question. An empty
		// option list is refused rather than indexed: a turn with nothing to take
		// is a turn with nothing to describe.
		if len(p.pending.Options) > 0 {
			m.play = p
			m.blurb.from = screenPlay
			m.blurb.scroll = 0
			m.screen = screenBlurb
			return m, nil
		}
	case "enter", "space":
		p = p.choose(m)
	case "a":
		// The engine's own answer, taken as the player's: somebody who wants to
		// see what it would do here should not have to guess it, and it lands in
		// the script as a decision like any other.
		if err := p.engineOrder(p.pending); err != nil {
			p.err = err
		} else {
			p.pending = nil
			p = p.run()
		}
	case "p":
		if err := p.skip(p.pending, ""); err != nil {
			p.err = err
		} else {
			p.pending = nil
			p = p.run()
		}
	}
	m.play = p
	return m, nil
}

// move walks whichever of the two lists is in front.
func (p playScreen) move(by int) playScreen {
	if p.aiming {
		aims := p.pending.Options[p.option].Aims
		p.aim = (p.aim + by + len(aims)) % len(aims)
		return p
	}
	// Unavailable options are stepped over rather than hidden: a player deciding
	// what to do needs to know a skill exists and is two turns away, and a cursor
	// that could rest on one would be a cursor whose enter does nothing.
	for step := 1; step <= len(p.pending.Options); step++ {
		next := (p.option + by*step + len(p.pending.Options)*len(p.pending.Options)) % len(p.pending.Options)
		if p.pending.Options[next].Available() {
			p.option = next
			return p
		}
	}
	return p
}

// choose answers whichever question is in front: which skill, and then where.
//
// A skill with one legal cell does not ask the second question, because a
// question with one answer is not a decision.
func (p playScreen) choose(m model) playScreen {
	option := p.pending.Options[clamp(p.option, 0, len(p.pending.Options)-1)]
	if !option.Available() {
		return p
	}
	if !p.aiming && len(option.Aims) > 1 {
		p.aiming, p.aim = true, 0
		return p
	}
	aim := option.Aims[0]
	if p.aiming {
		aim = option.Aims[clamp(p.aim, 0, len(option.Aims)-1)]
	}
	if err := p.take(p.pending, option.Skill, aim); err != nil {
		p.err = err
		return p
	}
	p.pending, p.aiming = nil, false
	return p.run()
}

// save writes the battle out where the game client can replay it.
//
// It may be pressed at any point rather than only at the end, and that is not a
// concession: a battle stopped halfway is a battle, its script is consistent,
// and re-running it reproduces exactly the half that was played. What a log
// records is what happened, not what finished.
func (p playScreen) save(m model) playScreen {
	if p.fight == nil {
		return p
	}
	home, away, ok := m.fight.sides(m)
	if !ok {
		return p
	}
	path, err := m.lib.SaveBattleLog(home.ID, away.ID, p.seed, battle.Log{
		Seed: p.seed, Roster: p.roster, Choices: p.script, Events: p.events,
	})
	if err != nil {
		p.err, p.notes = err, nil
		return p
	}
	p.err = nil
	// The second note is the whole reason the first one is worth having, and it
	// carries the rebuild warning itself: --verify re-runs against the copy
	// baked into the game binary, so a log written after an edit nobody rebuilt
	// will not verify, and the mismatch would read as corruption.
	//
	// It names the file relative to the data directory, because what goes after
	// --replay is a path somebody has to type and the absolute one is mostly the
	// part they are already standing in.
	relative := path
	if shortened, err := filepath.Rel(m.lib.Dir(), path); err == nil {
		relative = shortened
	}
	p.notes = []forge.Note{
		{Kind: forge.NoteWrote, ID: filepath.Base(path), Path: path},
		{Kind: forge.NoteBattleVerify, Path: relative},
	}
	return p
}

// playLogWanted is how many rendered rows the log asks the budget for.
//
// It is what the player has to read: what happened since they last chose. A
// screen that grew a line per event would push the board off the top by the
// third turn, which is why the section asks for a few rows rather than for all
// of them.
//
// ⚠️ **It is a floor of intent and no longer a ceiling**, and it was the ceiling
// that was the defect. `playFit` clamped the log's allotment to this number, so
// the body grew 20 → 42 rows between an 80x24 window and an 80x80 one and the log
// stood still at eight — a tall terminal bought the history nothing. The log now
// takes whatever rows nobody above it in the priority claimed, and this is only
// what it asks for **first**, which is what still makes "everything fits" a
// question with an answer: a window that gives the log its eight rows is a window
// with nothing missing, and one that gives it forty is the same window with room
// to spare.
//
// ⚠️ **It is not a floor in the priority**, and it cannot become one. The log is
// last precisely because it is history rather than state, so a guaranteed eight
// rows would have to be taken off the roster, the board or the order line — and
// every drop height in CLAUDE.md would move. Growing the log may only ever spend
// rows nobody else claimed.
//
// ⚠️ **Rendered rows, not events**, and it used to be events. The two are not
// the same number: tui.Line opens a turn with a blank row of its own, so one
// event arrives as two rows, and eight events measured **eleven rows** in a
// battle a few turns deep.
//
// Renamed off playLogLines because the number no longer says how many lines the
// screen keeps — it says how many it asks for.
const playLogWanted = 8

// playBodyRoom is how many rows the body may write before frame cuts it.
//
// frame gives the whole screen m.height - 2 rows, spends the first two on the
// header and the blank under it, and puts the footer below whatever padding is
// left — so the body's own purse is four rows short of the window. Reading it
// off frame's arithmetic rather than writing the number down is the point: a
// second copy of "how many rows are there" is how a screen comes to disagree
// with the frame drawn around it.
func playBodyRoom(height int) int { return height - 4 }

// The battle screen budgets its own body, and the reason is that it cannot fit.
//
// Measured at the declared 80x24 floor, where the body's purse is twenty rows:
// the heading is one, tui.Board is a fixed ten, tui.Roster is one plus a row a
// unit, tui.Order is one, the log asks for playLogWanted, and the option list is
// one plus a row an option. A legal squad is up to hex.MaxTeamSize a side, so
// **28 rows is the floor for a 5-a-side pairing** before a single blank or log
// line — and a summon puts units on the board past the five the squad brought,
// up to the nine formation slots a side, which is board + roster = 29 on its
// own. There is no arrangement of these sections that fits twenty rows.
//
// So the deliverable was never "make it fit". frame cuts from the **bottom** and
// the option list was the last thing the body wrote, so the one thing the player
// has to see in order to act was the first thing thrown away. That is fixable at
// any content height, and it is the actual defect.
//
// The heading and the turn in front are therefore **reserved**: never dropped,
// never cut. A battle screen that cannot show the moves is not a battle screen.
// Everything else takes what is left, in this order:
//
//  1. The save's own note. It is the answer to a keystroke pressed a moment ago
//     and it names the file that was written, so it outranks the board — and it
//     is **not** reserved, because two notes wrap to as many as four rows and
//     reserving them could crowd out the option list.
//  2. tui.Roster, clipped a row at a time. It carries the health and the effects
//     a turn is decided on, and it is the one section that compresses by degrees
//     rather than all at once.
//  3. tui.Board, dropped **whole**, because ten rows of ASCII art have no half.
//     What it says is recoverable: the aim list already prints the occupant
//     beside every cell it offers (playScreen.occupant), so the question the
//     board answers is answered again where the player is pointing.
//  4. tui.Order, one row, ahead of the log.
//  5. The log, which asks for playLogWanted rendered rows and then takes every
//     row nobody above it claimed. It is last because it is history rather than
//     state, and the two-part answer is what lets a tall window buy the history
//     something without any of the four sections above losing a row.
//
// The log is also the one section a reader can move: it is a frame over the whole
// history rather than a fixed tail, and pgup/pgdown walk it — as do [ and ], the
// aliases the footer advertises, because a compact keyboard has neither page key.
// Following the tail is a state and not an offset — see the fields on playScreen
// for why that is the decision the rest of it hangs off.
//
// ⚠️ Nothing here may touch the battle. The plan is computed while drawing, and
// this screen is the one holding a pointer the model does not copy.
//
// playSizes is what each section would spend if it were drawn whole. Every one
// of them is measured off what is actually on the board rather than assumed,
// which is what makes a summoned unit cost a row the way a placed one does.
type playSizes struct {
	// tail is the turn in front: the option list, or the ending once the battle
	// is over. Reserved, so it is not a section the plan chooses about.
	tail int
	// notes is what a save left behind, already wrapped.
	notes int
	// board is tui.Board's rows, and units is tui.Roster's rows without its
	// header — the header goes with the first unit or not at all, because a
	// column heading over nothing is a row spent on nothing.
	board int
	units int
	// log is how many rendered rows the **whole history** comes to, which is
	// what the section would spend if it were drawn whole, the way every other
	// field here is. It used to be the tail capped at eight, and a section
	// reporting its cap as its size is a section that can never be given more.
	log int
}

// playPlan is how much of each section the body draws.
type playPlan struct {
	notes bool
	// board is drawn whole or not at all; roster is a count of units.
	board  bool
	roster int
	order  bool
	// log is how many of the log's rendered lines survive, counted from its end.
	log int
	// notice is the one dim line naming what is not shown and why. It is a row
	// like any other and is budgeted for before the sections are allotted.
	notice bool
}

// playFit is the whole of the arithmetic above.
//
// Two passes rather than one. The first asks whether everything fits, because a
// screen with nothing missing has nothing to say; only when something has to go
// is a row spent saying so, and the second pass allots what is left of the
// smaller purse. If that pass turns out to have nothing worth naming — a log
// tail one line shorter is the tail it always is, not something hidden — the
// first plan is kept and the log gets the row back.
func playFit(room int, sizes playSizes) playPlan {
	left := room - 1 - blockRows(sizes.tail)
	plan, whole := playTake(left, sizes)
	if whole {
		return plan
	}
	squeezed, _ := playTake(left-1, sizes)
	squeezed.notice = true
	if len(playHidden(squeezed, sizes)) == 0 {
		return plan
	}
	return squeezed
}

// playTake is the greedy walk down the priority list, and whole says whether
// every section got all of what it wanted.
func playTake(left int, sizes playSizes) (playPlan, bool) {
	var plan playPlan
	whole := true
	if sizes.notes > 0 {
		if cost := blockRows(sizes.notes); cost <= left {
			plan.notes, left = true, left-cost
		} else {
			whole = false
		}
	}
	// The roster and the board are one pane sharing one blank above them, the
	// way the game client draws them, and the roster is allotted first so the
	// roster pays for it: the blank and the column heading go with the first
	// unit and with no unit at all, because a heading over nothing is a row
	// spent on nothing.
	if sizes.units > 0 {
		if rows := left - 2; rows > 0 {
			plan.roster = min(sizes.units, rows)
			left -= 2 + plan.roster
		}
		if plan.roster < sizes.units {
			whole = false
		}
	}
	// The board goes whole or not at all — ten rows of drawing have no half —
	// and it is never drawn over a roster that is not: the picture without the
	// health is the wrong half of the pane to keep, and it is what the priority
	// already says, since the board is dropped before the roster's last row.
	if sizes.board > 0 {
		if plan.roster > 0 && sizes.board <= left {
			plan.board, left = true, left-sizes.board
		} else {
			whole = false
		}
	}
	if cost := blockRows(1); cost <= left {
		plan.order, left = true, left-cost
	} else {
		whole = false
	}
	// The log asks for playLogWanted rows and is answered in two parts, which is
	// what lets it grow without moving anything above it. First its own ask, so
	// that "everything fits" still means something: a window that gives it those
	// rows has nothing missing. Then whatever nobody claimed, because a tall
	// terminal ought to buy the history something and the log is the only section
	// on this screen with more to show than it is ever given.
	if sizes.log > 0 {
		wanted := min(sizes.log, playLogWanted)
		if rows := left - 1; rows > 0 {
			plan.log = min(wanted, rows)
			left -= 1 + plan.log
		}
		if plan.log < wanted {
			whole = false
		}
		// The surplus, and only the surplus: the log is last in the priority, so
		// every row still in hand here is a row nobody above it wanted. It cannot
		// take more rows than the history has, or the frame would be padded with
		// nothing.
		if plan.log > 0 && left > 0 {
			spare := min(left, sizes.log-plan.log)
			plan.log += spare
			left -= spare
		}
	}
	return plan, whole
}

// blockRows is what a section of n rows costs: the rows, plus the blank row that
// separates it from whatever is above. A section of nothing costs nothing.
//
// Named in full rather than `block` for the reason drawnRows is: another screen
// in this package already uses that word for a local.
func blockRows(rows int) int {
	if rows <= 0 {
		return 0
	}
	return rows + 1
}

// playHidden is what the notice names, in the order the screen would have drawn
// the sections it is talking about.
//
// The keys rather than the sentence, because the line is composed in one place
// and because the count is what decides whether there is a line at all. A log
// that came back a row or two shorter is **not** in here: the log is a tail by
// design, so a shorter tail is the section working rather than a section
// missing, and naming it would put a notice on nearly every window.
func playHidden(plan playPlan, sizes playSizes) []i18n.Key {
	var hidden []i18n.Key
	if sizes.board > 0 && !plan.board {
		hidden = append(hidden, i18n.PlayHiddenBoard)
	}
	if sizes.units-plan.roster > 0 {
		hidden = append(hidden, i18n.PlayHiddenUnits)
	}
	if !plan.order {
		hidden = append(hidden, i18n.PlayHiddenOrder)
	}
	if sizes.log > 0 && plan.log == 0 {
		hidden = append(hidden, i18n.PlayHiddenLog)
	}
	if sizes.notes > 0 && !plan.notes {
		hidden = append(hidden, i18n.PlayHiddenNote)
	}
	return hidden
}

// hiddenSeparator is what the notice's list is joined with. Punctuation rather
// than a wording: both languages point a list with a comma, and the two ASCII
// cells are the same in either.
const hiddenSeparator = ", "

// notice is the one line saying what is not shown and why.
func (p playScreen) notice(m model, plan playPlan, sizes playSizes) string {
	hidden := playHidden(plan, sizes)
	parts := make([]string, 0, len(hidden))
	for _, key := range hidden {
		// The unit count is the only entry carrying a number, and English needs
		// the singular where Vietnamese does not — hence the second key rather
		// than a plural rule.
		if key != i18n.PlayHiddenUnits {
			parts = append(parts, m.text(key))
			continue
		}
		if left := sizes.units - plan.roster; left == 1 {
			parts = append(parts, m.text(i18n.PlayHiddenUnitsOne))
		} else {
			parts = append(parts, m.text(i18n.PlayHiddenUnits, left))
		}
	}
	return m.style.dim.Render(
		m.text(i18n.PlayHidden, strings.Join(parts, hiddenSeparator)))
}

// playDrawn is every section of this screen drawn whole, before the budget says
// how much of each survives.
//
// It exists so that there is **one** reading of how many rows the log has and how
// many of them the window leaves it: the view that draws the frame and the keys
// that move it both ask this, and a key that scrolled by a different number of
// rows than the screen shows would step over lines nobody ever saw.
//
// ⚠️ Nothing here touches the battle. It is read while drawing and it is read on
// a keystroke, and this is the one screen holding a pointer the model does not
// copy.
type playDrawn struct {
	// tail is the turn in front, and over says the battle has finished — which
	// the footer needs and the sizes do not.
	tail []string
	over bool

	board  []string
	roster []string
	order  string
	// log is the **whole history**, rendered. The frame is a window into it.
	log   []string
	notes []string
}

// drawings measures every section against the board as it stands.
func (p playScreen) drawings(m model) playDrawn {
	var drawn playDrawn
	// The turn in front, read before anything else because it is what the rest of
	// the screen is budgeted around. A finished battle first, because its ending
	// is the answer to the question a prompt would have asked; then the prompt.
	// With neither — between turns, where the engine's own units act — there is no
	// question on the screen and nothing to reserve room for.
	switch {
	case p.fight.Finished():
		drawn.tail = []string{m.style.emphasis.Render(p.ending(m))}
		drawn.over = true
	case p.pending != nil:
		drawn.tail = drawnRows(p.choices(m))
	}
	drawn.board = drawnRows(tui.Board(p.fight, p.tags))
	drawn.roster = drawnRows(tui.Roster(p.fight, p.tags))
	drawn.order = m.style.dim.Render(tui.Order(p.fight.Queue(), p.tags, 6))
	drawn.log = p.logRows(m)
	drawn.notes = p.wrote(m)
	return drawn
}

// sizes is what each section would spend if it were drawn whole.
func (d playDrawn) sizes() playSizes {
	return playSizes{
		tail:  len(d.tail),
		notes: len(d.notes),
		board: len(d.board),
		// The header is not a unit, and tui.Roster always draws one.
		units: max(len(d.roster)-1, 0),
		log:   len(d.log),
	}
}

func (p playScreen) view(m model) (string, string) {
	footer := m.text(i18n.PlayFooter, saveKeyLabel())
	if p.aiming {
		footer = m.text(i18n.PlayAimFooter)
	}
	if p.err != nil {
		return p.heading(m, "") + "\n\n  " + m.style.bad.Render(m.lang.Error(p.err)), footer
	}
	if p.fight == nil {
		return p.heading(m, "") + "\n\n  " + m.text(i18n.SquadsEmpty), footer
	}

	drawn := p.drawings(m)
	if drawn.over {
		footer = m.text(i18n.PlayOverFooter, saveKeyLabel())
	}
	sizes := drawn.sizes()
	plan := playFit(playBodyRoom(m.height), sizes)
	log := p.logFrame(drawn.log, plan.log)

	body := []string{p.heading(m, p.logPosition(m, len(drawn.log), plan.log))}
	if plan.notice {
		body = append(body, p.notice(m, plan, sizes))
	}
	if plan.board || plan.roster > 0 {
		body = append(body, "")
		if plan.board {
			body = append(body, drawn.board...)
		}
		if plan.roster > 0 {
			body = append(body, drawn.roster[:plan.roster+1]...)
		}
	}
	if plan.order {
		body = append(body, "", drawn.order)
	}
	if len(log) > 0 {
		body = append(body, "")
		body = append(body, log...)
	}
	if len(drawn.tail) > 0 {
		body = append(body, "")
		body = append(body, drawn.tail...)
	}
	if plan.notes {
		body = append(body, "")
		body = append(body, drawn.notes...)
	}
	return strings.Join(body, "\n"), footer
}

// heading is the screen's title row, and the log's position in the history when
// there is one.
//
// ⚠️ **The position goes here rather than on a row of its own.** A row of its own
// would cost what the budget below spent a whole feature proving this screen has
// not got, and the title is about seventeen cells of the seventy-nine there are.
func (p playScreen) heading(m model, position string) string {
	row := m.style.heading.Render(m.text(i18n.PlayHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.PlaySeed, p.seed))
	if position == "" {
		return row
	}
	return row + "  " + m.style.dim.Render(position)
}

// logPosition is where the frame sits in the whole history, and nothing when the
// whole of it is on screen.
//
// ⚠️ **Shown whenever rows are hidden, not only while scrolled back.** Half of
// the defect this answers is that nothing on the screen said a history existed:
// eight rows of three hundred were drawn and the other two hundred and ninety-two
// were unreachable by any means. A reader who cannot see that there are three
// hundred rows will not go looking for the key that reaches them.
//
// Nothing is said when the log is not drawn at all — the notice under the heading
// already names it as a section the window is too short for, and a range for a
// frame nobody can see would be a position in a thing that is not there.
func (p playScreen) logPosition(m model, total, room int) string {
	if room <= 0 || total <= room {
		return ""
	}
	start := p.logStart(total, room)
	return m.text(i18n.PlayLogRange, start+1, start+room, total)
}

// scrollLog moves the log's frame by whole pages, and does nothing at all when
// the history already fits the frame.
//
// A page rather than a row, because the history runs to hundreds of rows and a
// key that had to be pressed two hundred times to reach the opening board is a
// key nobody presses twice.
//
// ⚠️ **Reaching the bottom asks to follow again**, and that is not the sentinel
// the field comments refuse. A reader who scrolls down to the newest row is
// saying they want the newest row, which is a state; storing the number that
// happens to be the newest row today is the thing that goes wrong the moment the
// next event arrives. So the offset goes back to nothing there — nought is also a
// perfectly ordinary offset, meaning the top of the history, and the flag beside
// it is what tells the two apart. That is the whole argument for two fields.
func (p playScreen) scrollLog(m model, pages int) playScreen {
	if p.fight == nil || p.err != nil {
		return p
	}
	drawn := p.drawings(m)
	room := playFit(playBodyRoom(m.height), drawn.sizes()).log
	total := len(drawn.log)
	if room <= 0 || total <= room {
		// Nothing above the frame, so nothing to scroll to.
		return p
	}
	tail := total - room
	offset := clamp(p.logStart(total, room)+pages*room, 0, tail)
	if offset == tail {
		p.logFollow, p.logOffset = true, 0
		return p
	}
	p.logFollow, p.logOffset = false, offset
	return p
}

// drawnRows splits a drawing into the rows it occupies, dropping the empty one a
// trailing newline leaves behind.
//
// That last empty string is the miscount (*pickState).room had to be corrected
// for: frame splits the body on newlines, so a section ending in one is a
// section a row longer than it looks.
//
// Named for what it returns rather than `rows`, which half the screens in this
// package already use as a local: a package function shadowed in most of the
// files that could call it is one nobody reaches for.
func drawnRows(drawn string) []string {
	trimmed := strings.TrimRight(drawn, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// logRows is the whole history, rendered.
//
// ⚠️ **Every event, and it used to be the last few.** The old reading walked from
// the end and stopped at the budget, which meant the rows past the frame were not
// merely off screen — they did not exist, so no key could have reached them and
// the frame had nothing to be a window into. p.events holds every event a battle
// has emitted (collect appends and never trims), so the history was always there
// and it was the view that threw it away.
//
// Counted in **rendered rows** and not in events, because tui.Line opens a turn
// with a blank row and one event therefore arrives as two rows. Each event is
// rendered exactly as it was before there was a budget — the indent goes on the
// front of whatever tui.Line returned, once, so a turn's blank row still reads as
// a blank row — and the rows are then counted rather than the events.
//
// Nothing here reads the battle: the event log is the only contract a reader of
// one has.
func (p playScreen) logRows(m model) []string {
	var lines []string
	for _, event := range p.events {
		line := tui.Line(event, p.tags)
		if line == "" {
			continue
		}
		lines = append(lines, drawnRows("  "+m.style.dim.Render(line))...)
	}
	return lines
}

// logStart is the first row of the history the frame shows.
//
// ⚠️ **Clamped against the total every time it is read**, not only where the
// offset is written. Undo is a shorter script replayed, so the history is rebuilt
// shorter than the one the offset was taken in, and an offset carried across it
// points past the end.
func (p playScreen) logStart(total, room int) int {
	if room <= 0 {
		return 0
	}
	tail := max(total-room, 0)
	if p.logFollow {
		return tail
	}
	return clamp(p.logOffset, 0, tail)
}

// logFrame is the rows of the history that are on screen: the tail while the
// reader is following it, and whichever page they scrolled back to otherwise.
func (p playScreen) logFrame(rows []string, room int) []string {
	if room <= 0 {
		return nil
	}
	if len(rows) <= room {
		return rows
	}
	start := p.logStart(len(rows), room)
	return rows[start : start+room]
}

// choices is the turn in front: whose it is, what they may do, and where it may
// be pointed once a skill is picked.
func (p playScreen) choices(m model) string {
	unit, known := p.fight.Unit(p.pending.Unit)
	if !known {
		return ""
	}
	var out strings.Builder
	out.WriteString(m.style.label.Render(m.text(i18n.PlayYourTurn,
		p.tags[unit.ID], unit.Name, p.pending.Turn)) + "\n")
	// The id column is measured over the options this turn offers rather than
	// fixed, for the reason menuLabelWidth and every detail pane measure theirs.
	// Over the options and not over the book: the widest id in the game is
	// thirteen cells and this unit may be bringing four short ones, so a
	// book-wide column would spend the summary's room on a skill nobody here can
	// cast.
	width := p.optionWidth()
	// Clipped to the floor rather than to the window in hand, which is what the
	// caution line and the trait sentences do and for the same reason: measuring
	// the real terminal gives one line two shapes and leaves the width sweep
	// nothing to hold. Below the floor no screen is drawn at all, so the floor is
	// also the narrowest room this row ever really has.
	clip := lipgloss.NewStyle().MaxWidth(max(minWidth-1-markerWidth-width-optionGap, 0))
	for index, option := range p.pending.Options {
		marker := "  "
		// ⚠️ An unavailable option keeps its reason and **drops its summary**.
		// The row has one slot and the two answer different questions: the reason
		// is why this cannot be cast, which is the live question the moment a
		// cursor steps over it, and what the skill does is a ? away. Do not
		// "fix" this by drawing both — the second one would be the half that got
		// clipped.
		tail := p.summarise(m, option.Skill)
		if !option.Available() {
			tail = option.Reason
		}
		line := option.Skill
		if tail != "" {
			line += strings.Repeat(" ",
				width-lipgloss.Width(option.Skill)+optionGap) + clip.Render(tail)
		}
		switch {
		case !option.Available():
			line = m.style.dim.Render(line)
		case index == p.option && !p.aiming:
			marker = "> "
			line = m.style.selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}
	if !p.aiming {
		return out.String()
	}
	option := p.pending.Options[clamp(p.option, 0, len(p.pending.Options)-1)]
	out.WriteString("\n" + m.style.label.Render(m.text(i18n.PlayAimAt, option.Skill)) + "\n")
	for index, cell := range option.Aims {
		marker := "  "
		line := cell.String()
		if held := p.occupant(cell); held != "" {
			line += "  " + held
		}
		if index == p.aim {
			marker = "> "
			line = m.style.selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}
	return out.String()
}

// The two fixed columns a row spends before its summary: the cursor marker, and
// the gap between the id column and whatever follows it.
const (
	markerWidth = 2
	optionGap   = 2
)

// optionWidth is the id column, measured over the turn's own options.
func (p playScreen) optionWidth() int {
	width := 0
	for _, option := range p.pending.Options {
		if drawn := lipgloss.Width(option.Skill); drawn > width {
			width = drawn
		}
	}
	return width
}

// summarise is what a skill does, in the one line that fits beside its id.
//
// The whole reason the list is worth reading: an id is a name, and nothing in
// "venoshock" says it is the skill that doubles into a poison. It is
// i18n.Lang.SummariseSkill rather than anything assembled here — this screen may
// hold no wording of its own, and the compact line is tied to the full
// description by a test in that package rather than by a screen's good
// intentions.
//
// A skill the book cannot find summarises as nothing rather than as an error.
// The options come out of the battle, which was built from the same library, so
// a miss is not reachable; and a row is the wrong place to report that it was.
func (p playScreen) summarise(m model, id string) string {
	declared, err := m.lib.Skills().Lookup(id)
	if err != nil {
		return ""
	}
	return m.lang.SummariseSkill(declared, m.lib.Patterns())
}

// occupant is the tag and name standing on a cell, so an aim reads as somebody
// rather than as a coordinate.
func (p playScreen) occupant(cell hex.Offset) string {
	for _, unit := range p.fight.Units() {
		if unit.Dead || unit.Cell != cell {
			continue
		}
		return p.tags[unit.ID] + " " + unit.Name
	}
	return ""
}

// ending is how the battle finished, in the words the game client uses for it.
func (p playScreen) ending(m model) string {
	switch p.fight.Outcome() {
	case battle.Victory:
		winner, _ := p.fight.Winner()
		if winner == p.side {
			return m.text(i18n.PlayWon)
		}
		return m.text(i18n.PlayLost)
	case battle.Stalemate:
		return m.text(i18n.PlayDrawn)
	default:
		return m.text(i18n.PlayEmptied)
	}
}

// wrote is the line a save leaves behind, in the shape every other write in this
// client reports itself.
// It hands back the rows rather than a block, because the budget above counts
// them: a section that reported itself as one string would have to be measured
// twice, once to place it and once to draw it.
func (p playScreen) wrote(m model) []string {
	if len(p.notes) == 0 {
		return nil
	}
	var out []string
	for index, note := range m.lang.Notes(p.notes) {
		style := m.style.dim
		if index == 0 {
			style = m.style.good
		}
		// Wrapped against minWidth rather than the window in hand, for the
		// reason the fight's caution is: measuring the real terminal would give
		// one sentence two shapes and leave the width sweep nothing to hold.
		for _, line := range wrapWords(note, minWidth-1) {
			out = append(out, style.Render(line))
		}
	}
	return out
}
