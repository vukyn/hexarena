package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// fightScreen is forge.FightSquads drawn.
//
// Which two squads are fought is this screen's own question, so it carries both
// of them: home and away are its own indices, and neither is read off another
// screen's cursor. That is what lets it be reached from the menu — a player who
// has built a side and wants a battle should not have to find one behind the
// catalogue.
//
// The catalogue's f still seeds home from whatever was under its cursor, so
// arriving that way fights the squad that was pointed at, which is what that key
// has always meant.
type fightScreen struct {
	// home and away are indices into the catalogue, not ids: the catalogue is
	// what the choosers walk, and an id would need looking up on every keystroke
	// to find out where in it a cursor is.
	home, away int
	// seeds is how many battles each half is fought over, so a run is twice
	// this many.
	seeds int
	// fought and failed are the runs already made, keyed by the pair and the
	// seed count. Every method here has a value receiver, so a plain field
	// written while drawing would be thrown away with the copy — and a run is
	// fast per keystroke but not free, and the opponent moves under an arrow key
	// that repeats.
	fought map[string]forge.SquadReport
	failed map[string]error
}

// The battles a run is fought over, and how far it may be moved.
//
// The same ladder the spar uses, and deliberately: a hundred tells a lopsided
// pairing from an even one and is cheap enough to run while a key repeats, a
// thousand is where a percentage point starts to mean something. ⚠️ A squad
// battle is five units a side rather than one against one, so the top of this
// ladder is slower than the spar's by more than the seed count suggests.
const (
	fightSeeds    = 100
	fightMinSeeds = 10
	fightMaxSeeds = 1000
)

func newFightScreen() fightScreen {
	return fightScreen{
		seeds:  fightSeeds,
		fought: map[string]forge.SquadReport{},
		failed: map[string]error{},
	}
}

// refresh empties the cache, and is called on entering the screen.
//
// A report is derived from the whole library and the library is written to by
// the screen this one is raised from: entering the fight after editing a squad
// has to fight the squad that was edited.
func (f fightScreen) refresh() fightScreen {
	f.fought = map[string]forge.SquadReport{}
	f.failed = map[string]error{}
	if f.seeds == 0 {
		f.seeds = fightSeeds
	}
	return f
}

func (f fightScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	squads := m.squad.saved
	switch message.String() {
	case "esc", "f":
		m.screen = screenSquads
		return m, nil
	case "up", "k":
		if len(squads) > 0 {
			f.home = (f.home - 1 + len(squads)) % len(squads)
		}
	case "down", "j":
		if len(squads) > 0 {
			f.home = (f.home + 1) % len(squads)
		}
	case "left", "h":
		if len(squads) > 0 {
			f.away = (f.away - 1 + len(squads)) % len(squads)
		}
	case "right", "l":
		if len(squads) > 0 {
			f.away = (f.away + 1) % len(squads)
		}
	case "p":
		// Play the pairing by hand. What the run answers over two hundred
		// battles, one played battle answers once — and the two belong beside
		// each other, because a rate says which squad is better and playing one
		// says why.
		if len(squads) > 0 {
			m.fight = f
			return m.enter(screenPlay), nil
		}
	case "+", "=":
		f.seeds = clamp(f.seeds*2, fightMinSeeds, fightMaxSeeds)
	case "-":
		f.seeds = clamp(f.seeds/2, fightMinSeeds, fightMaxSeeds)
	}
	m.fight = f
	return m, nil
}

// sides is the two squads being fought, both read off this screen's own
// choosers.
//
// The catalogue is still where the list comes from — it is the one reading of
// what is on disk, and a second copy of that would be a second thing to keep in
// step — but which two of it are fought is settled here, so the screen is a
// whole subject on its own and can be opened by anybody.
func (f fightScreen) sides(m model) (home, away placement.Squad, ok bool) {
	squads := m.squad.saved
	if len(squads) == 0 {
		return placement.Squad{}, placement.Squad{}, false
	}
	home = squads[clamp(f.home, 0, len(squads)-1)]
	away = squads[clamp(f.away, 0, len(squads)-1)]
	return home, away, true
}

// report is the run for the pair in front, fighting it if it has not been
// fought.
func (f fightScreen) report(m model) (forge.SquadReport, error, bool) {
	home, away, ok := f.sides(m)
	if !ok {
		return forge.SquadReport{}, nil, false
	}
	key := fmt.Sprintf("%s|%s|%d", home.ID, away.ID, f.seeds)
	if done, cached := f.fought[key]; cached {
		return done, nil, true
	}
	if failure, cached := f.failed[key]; cached {
		return forge.SquadReport{}, failure, true
	}
	done, err := m.lib.FightSquads(home.ID, away.ID, f.seeds)
	if err != nil {
		f.failed[key] = err
		return forge.SquadReport{}, err, true
	}
	f.fought[key] = done
	return done, nil, true
}

func (f fightScreen) view(m model) (string, string) {
	footer := m.text(i18n.FightFooter)
	home, away, ok := f.sides(m)
	if !ok {
		// Reachable from the menu now, and it was not before: the catalogue's f
		// did nothing on an empty list, so this was the one state nobody could
		// get to. It says what has to happen next rather than borrowing the
		// catalogue's line, which offers a key this screen does not have.
		return "  " + m.text(i18n.FightNoSquads) + "\n", footer
	}
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.FightHeading)) + "  " +
		m.style.label.Render(home.ID) + "  " +
		m.style.dim.Render(m.text(i18n.FightAgainst)) + "  " +
		fmt.Sprintf("< %s >", away.ID) + "\n")
	if home.ID == away.ID {
		out.WriteString("  " + m.style.dim.Render(m.text(i18n.FightControl)) + "\n")
	}
	out.WriteString("\n")

	report, failure, _ := f.report(m)
	if failure != nil {
		out.WriteString("  " + m.style.bad.Render(m.lang.Error(failure)) + "\n")
		return out.String(), footer
	}
	total := report.Total()
	width := fightLabelWidth(m)
	out.WriteString("  " + m.style.dim.Render(m.text(i18n.FightConditions,
		report.Seeds, total.Battles())) + "\n\n")
	out.WriteString(m.labelAt(m.text(i18n.FightRate), width, "%s",
		m.style.emphasis.Render(forge.Percent(report.Rate()))))
	out.WriteString(m.labelAt(m.text(i18n.FightRecord), width, "%s",
		m.text(i18n.FightRecordLine, total.Wins, total.Losses, total.Draws)))
	// The two halves apart, because the difference between them is what the side
	// itself is worth — and on a board where a front rank shields the columns
	// behind it, that is not nothing.
	out.WriteString(m.labelAt(m.text(i18n.FightBySide), width, "%s",
		m.text(i18n.FightBySideLine,
			forge.Percent(report.AsAlly.Rate()), forge.Percent(report.AsEnemy.Rate()))))
	out.WriteString(m.labelAt(m.text(i18n.FightLength), width, "%s",
		m.text(i18n.FightLengthLine, report.Turns)))
	if total.Endless > 0 {
		out.WriteString(m.labelAt(m.text(i18n.FightEndless), width, "%s",
			m.style.bad.Render(m.text(i18n.FightEndlessLine, total.Endless))))
	}
	out.WriteString("\n" + f.caution(m))
	return strings.TrimRight(out.String(), "\n"), footer
}

// fightLabelWidth is measured rather than fixed, like every other label column
// here: the two languages word these differently and a constant would be right
// for one of them.
func fightLabelWidth(m model) int {
	width := 0
	for _, key := range []i18n.Key{
		i18n.FightRate, i18n.FightRecord, i18n.FightBySide,
		i18n.FightLength, i18n.FightEndless,
	} {
		if drawn := lipgloss.Width(m.text(key)); drawn > width {
			width = drawn
		}
	}
	return width + 3
}

// caution is the one line on this screen that is prose: the sentence that stops
// the figure above it being read as something it is not.
//
// It is wrapped against **minWidth** rather than against the window in hand, for
// the reason the art chooser measures its own room that way: measuring the real
// terminal would give the same sentence two shapes, and the width sweep would
// have nothing to hold. A wide terminal gets the same two short lines a narrow
// one does, which is what every other wording here promises.
func (f fightScreen) caution(m model) string {
	const marker = 2
	var out strings.Builder
	for _, line := range wrapWords(m.text(i18n.FightCaution), minWidth-1-marker) {
		out.WriteString("  " + m.style.dim.Render(line) + "\n")
	}
	return out.String()
}
