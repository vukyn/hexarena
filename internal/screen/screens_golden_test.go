package screen

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// # The moved screens as a golden, in the package that now owns them
//
// Six listings moved out of cmd/hexforge-tui into this package. What did **not**
// move with them was the only test holding their layout: measured after that
// move, widening the status category column by one cell —
// `Pad(row.Category.String(), column+1)` → `column+2` in statuses.go — left
// **every test in this package green** and was caught by
// `cmd/hexforge-tui/testdata/screens.golden` alone.
//
//	internal/screen   → ok (all green)
//	client golden     → golden "  dot          sát thương mỗi lượt"
//	                    drawn  "  dot           sát thương mỗi lượt"
//
// That is real coverage sitting in another package, and it got worse rather than
// better the moment `cmd/hexarena-tui` stood up: a screen the authoring tool
// stops drawing would lose its only layout net **in silence**. So this file is
// that net,
// here, and the package holds twelve screens now that the skill listing, the
// squad builder, the works catalogue and the played battle have all arrived.
//
// ## What is recorded
//
// Forty-one entries over those twelve screens — the six listings, the
// description screen in both of its readings, the two states nothing shipped can
// draw (a species kind nobody claims, a build that spends no trait slot), the
// five states of the picker, which is handed its list and so has no one shape,
// the skill listing with the seven states of it the client's sweep registers, the
// squad builder at each of its three depths plus the two states and the two
// pickers that go with them, the works catalogue with the add-a-work form
// over it plus the two states **neither** sweep could draw before — an empty
// catalogue and a refused write — and the played battle in the six states of it
// that share no line, two of which are also states neither sweep could draw (a
// save's own note, and a battle with no pairing to open on) — in **both**
// languages at **two** sizes: the MinWidth x MinHeight floor, where the Room
// helpers bite, and 160x60, where nothing is squeezed — **164 renders, 3851
// lines**. The entry names are the client's own, so a reader holding both diffs
// is looking at the same words.
//
// ⚠️ **Two of the works entries are states the client's fixture cannot reach at
// all**, which is the same kind of finding as the three squad entries above and
// arrived the same way — by asking what each wording is rendered by rather than
// by trusting the map. `i18n.OriginsEmpty` needs a book with no works and every
// book either fixture loads has some; `i18n.AddRefused` on this form needs a
// refusal, which the client only ever produces mid-write. Measured before they
// were added: both lines returned **nought hits in both goldens**, in both
// languages. A third, `i18n.OriginAdded`, stays unrecorded on purpose — see
// aRefusedWork for why.
//
// ⚠️ **Three of the squad entries are the same screen in a fuller state than the
// client's of the same name, and that is deliberate.** The client's fixture
// deletes `squads.json` before it loads anything, because that file is the
// author's own working document and a suite that read it would be measuring
// somebody's saved sides — so its `squads` entry is the **empty** listing, and
// its `a squad trait` is a picker over a fixture cast that learns no traits. A
// listing with no rows measures no id column and a picker with no rows measures
// no row, so what the two records hold between them is the empty shape and the
// full one. The squads here are **built as values** and never read off
// `squads.json` for the same reason the client deletes it: a golden that moved
// when an author saved a side would be recording the author rather than the
// screen.
//
// Each render carries a banner naming it, so a diff says which screen moved
// rather than which line number did. The names are **sorted** before they are
// walked: `everyMovedScreen` hands back a map, Go randomises a map range, and a
// golden built off one churns at random — the client's golden's first bug was
// exactly that, and it is the same rule `internal/core` states as "no map
// iteration in anything that reaches an output", one layer up.
//
// A screen answers with a body *and* a footer, so both are recorded. The footer
// is where the trims live — every wording squeeze in `CLAUDE.md` is a footer —
// and a golden holding the body alone would not see one.
//
// ## Two things the client's golden had to do that this one does not
//
// It drops its header line and hands `forge.Load` a **relative** directory,
// because `screenContent` draws `m.lib.Dir()` and the check screen prints the
// data directory as a body line. **Neither applies here.** Measured: no file in
// this package calls `.Dir()`, and `check` did not move — so the books are
// loaded **straight** from `shippedDataDir`, with no temp copy anywhere near the
// bytes. Every screen recorded here is drawn read-only, and nothing in this file
// writes — the two screens that *can* write, the skill form and the add-a-work
// form, are recorded with their fields as an author would have typed them and
// their save keys never pressed.
//
// ⚠️ The cast browser's art row is the one line that names a **file**, and it
// names a relative one: `cast.Character.StageArt` is a path out of the data
// files, and whether it is on disk is a yes or a no rather than a directory. So
// the golden records `assets/…` and never where this checkout happens to sit —
// which is exactly the claim `noAbsolutePath` below is here to keep true.
//
// ⚠️ **`noAbsolutePath` asserts that anyway.** A property that holds by
// construction today is one a later change breaks quietly, and the cost of
// finding out from a committed golden naming somebody's home directory is higher
// than the cost of the walk.
//
// ⚠️ **This is not a strict superset of the client's golden and may not replace
// it.** That one records the same screens *as the application draws them* —
// through `model.enter`, inside `frame`, with the header, the blank and the
// footer composed into one window and every line clipped to it. Measured both
// ways: a column width inside a moved `View` reddens **both**; widening
// `Ellipsis`, which only `frame`'s clip can reach on these screens, reddens the
// client's alone. Two goldens over one set of screens, measuring the drawing and
// the framing of it.
func TestEveryMovedScreenDrawsWhatTheGoldenHolds(t *testing.T) {
	path := filepath.Join("testdata", "screens.golden")
	got := everyMovedScreenDrawn(t)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s: %d renders, %d lines",
			path, strings.Count(got, bannerMark), strings.Count(got, "\n"))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: make golden): %v", err)
	}
	if got == string(want) {
		return
	}
	t.Errorf("the moved screens differ from %s; accept with `make golden` and read the diff\n%s",
		path, firstDifference(string(want), got))
}

// goldenSizes is the two windows every screen is recorded at.
//
// The floor is where the layout argument lives: every Room helper in this package
// spends `c.Height - 4` and then reserves what its panes need, so that is where a
// row budgeted wrong shows up. The roomy one is the same screens with nothing
// taken away, which is what makes a diff at the floor readable — a row that moved
// in both is a row that moved, and a row that moved only at the floor is a budget
// moving.
var goldenSizes = [][2]int{
	{MinWidth, MinHeight},
	{160, 60},
}

const (
	// bannerMark opens the line that names a render. Counting it counts renders.
	bannerMark = "===== "
	// footerMark separates a body from the footer under it. A screen hands back
	// the two apart, so the record has to say where one ends — and it is written
	// on a line of its own **after** the body verbatim, never after a trim, so a
	// body ending in a newline is a visible blank line here rather than a
	// difference the golden cannot show. A trailing empty row is a row the
	// client's frame budgets against; miscounting one is what once truncated the
	// picker.
	footerMark = "----- footer -----"
)

// everyMovedScreenDrawn is the whole golden: every screen, both languages, both
// sizes.
func everyMovedScreenDrawn(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	for _, lang := range i18n.Langs() {
		c, lib := startOverTheShippedBooks(t, lang)
		screens := everyMovedScreen(t, c, lib)
		names := slices.Sorted(maps.Keys(screens))
		for _, size := range goldenSizes {
			sized := c
			sized.Width, sized.Height = size[0], size[1]
			for _, name := range names {
				body, footer := screens[name].View(sized)
				fmt.Fprintf(&out, "%s%s | %dx%d | %s %s\n",
					bannerMark, lang, size[0], size[1], name, strings.TrimSpace(bannerMark))
				out.WriteString(body)
				out.WriteString("\n" + footerMark + "\n")
				out.WriteString(footer + "\n")
			}
		}
	}
	body := out.String()
	noAbsolutePath(t, body)
	return body
}

// drawable is the one thing every screen in this package has in common and the
// only thing this file needs of them. Six concrete types with six different
// constructors go into one map through it.
type drawable interface {
	View(Context) (string, string)
}

// everyMovedScreen is the thirty-five entries, named as cmd/hexforge-tui's
// `everyScreen` names them — and, for the four works entries, named in that
// convention even though two of them are states the client has no entry for.
//
// ⚠️ **Every hand-built state asserts that it drew the line it exists for.**
// A registered state that renders nothing passes every sweep over it, and this
// repository has shipped that fixture more than once — a screen entered at its
// early return, a battle already finished. Two of them are not reachable from
// the shipped books at all (every kind is claimed, every build spends its trait
// slot), so there is nothing else to notice if the construction stops working;
// the two blurbs are reachable, and are checked for the same reason — a subject
// built here with the wrong id draws the describer's "nothing to describe" arm,
// which is a perfectly well-formed screen that measures none of the code the
// entry was added for.
//
// ⚠️ **The blurb answers three subject kinds and gets two entries.** A listed
// skill and a battle option are **one** kind (see Subject.SkillSubject: same id,
// same paragraph, same footer — only At and Of differ), so a third entry would
// record the same render under a second name; and the third kind, NoSubject, is
// the arm a raise cannot reach, which a client's applier is what proves.
//
// ⚠️ **The art preview is the thirteenth screen and the one entry here that
// records a drawing rather than a sentence** — see theArtPreview for what a
// plain-text record of it can and cannot hold.
func everyMovedScreen(t *testing.T, c Context, lib *forge.Library) map[string]drawable {
	t.Helper()
	species := NewSpeciesScreen(lib)
	unclaimed := withNobodyClaiming(species)
	if drawn, _ := unclaimed.View(c); !strings.Contains(drawn, c.Text(i18n.SpeciesNobodyIs)) {
		t.Fatalf("the unclaimed-kind state draws no line saying so, so the golden records "+
			"an ordinary species screen twice:\n%s", drawn)
	}
	builds := NewBuildsScreen(lib)
	traitless := withNoTraitTaken(t, builds)
	if drawn, _ := traitless.View(c); !strings.Contains(drawn, c.Text(i18n.BuildsNoTrait)) {
		t.Fatalf("the traitless-build state draws no row saying so, so the golden records "+
			"an ordinary build catalogue twice:\n%s", drawn)
	}
	listing := NewSkillsScreen(c)
	works := NewOriginsScreen(c)
	catalogue := squadCatalogue(t, c, lib)
	member := squadMember(t, c, catalogue)
	return map[string]drawable{
		// The cast browser as a raise leaves it: the first row, at the level
		// cap, with no origin filter. That is the state the client's own sweep
		// registers under this name, and a cursor moved here would be this
		// record answering a different question from that one.
		"allowlist picker": allowlistPicker(t, c, lib),
		"browse":           NewBrowseScreen(lib),
		"builds":           builds,
		"chart":            ChartScreen{},
		"elements":         ElementsScreen{},
		"filtered picker":  filteredPicker(t, c, lib),
		"kit picker":       kitPicker(t, c, lib),
		"reading a skill":  readingPicker(t, c, lib),
		"skill blurb":      skillBlurb(t, c, lib),
		"species":          species,
		"status picker":    statusPicker(t, c, lib),
		"statuses":         NewStatusesScreen(lib),
		"the art preview":  theArtPreview(t, c, lib),
		"trait blurb":      traitBlurb(t, c, lib),
		// ⚠️ **The three read-only views over the one shipped character whose
		// evolution line forks**, which is the case every entry above walks past.
		// See theForkedRaise for what they hold that the linear ones cannot.
		"a forked cast row":    theForkedCastRow(t, c, lib),
		"a forked art preview": theForkedArtPreview(t, c, lib),
		"a forked trait blurb": theForkedTraitBlurb(t, c, lib),
		"traitless build":      traitless,
		"traits":               NewPassivesScreen(lib),
		"unclaimed kind":       unclaimed,
		// The skill listing and the seven states of it the client's own sweep
		// registers, under the same names, for the reason every other entry
		// here carries the client's name: a reader holding both diffs is
		// looking at the same words.
		"skills":                  listing,
		"add a skill":             addingASkill(t, c, listing),
		"edit a skill":            editingASkill(t, c, listing),
		"edited a skill":          anEditedSkill(t, c, lib, listing),
		"filtering skills":        theFilterOpen(t, c, listing),
		"filtered skills":         theFilterFinding(t, c, listing),
		"skills filtered to none": theFilterFindingNothing(t, c, listing),
		"shape diagram":           theShapeDiagram(t, c, lib, listing),
		// The squad builder at each of its three depths, the two states of a
		// member that share no line with the plain one, and the two pickers it
		// raises — under the client's own names, for the reason every other
		// entry here carries them.
		// The works catalogue and the add-a-work form over it, under the client's
		// own names. The listing is drawn off the shipped book rather than off
		// values, unlike the squads below it: origins.json is data somebody wrote
		// for the game and it ships with rows in it, so nothing has to be
		// authored to measure a row.
		"origins":                  works,
		"add a work":               addingAWork(t, c, works),
		"an empty works catalogue": anEmptyWorksCatalogue(t, c, works),
		"a refused work":           aRefusedWork(t, c, works),
		"squads":                   catalogue,
		"a squad":                  squadInHand(t, c, catalogue),
		"a squad member":           member,
		"a deep member":            deepMember(t, c, member),
		"a held-back member":       heldBackMember(t, c, member),
		// ⚠️ **The builder on the one shipped character whose line forks**, at
		// both depths, which is the case every squad entry above walks past: the
		// fixture squad is built over the cast's most-traited character and every
		// one of those lines has a single end. See theForkedMember.
		"a forked squad":  theForkedSquad(t, c, lib),
		"a forked member": theForkedMember(t, c, lib),
		"a squad kit":     squadKitPicker(t, c, member),
		"a squad trait":   squadTraitPicker(t, c, member),
		// The played battle in the six states of it that share no line, under
		// the client's own names where it has one. Each builds its own battle —
		// see the note over these builders for the pointer that makes that a
		// rule rather than tidiness.
		"a battle":                 aBattle(t, c),
		"aiming":                   aBattleAiming(t, c),
		"a battle over":            aFinishedBattle(t, c),
		"a scrolled battle log":    aScrolledBattleLog(t, c),
		"a saved battle":           aSavedBattle(t, c),
		"a battle with no pairing": aBattleWithNoPairing(t, c),
		// The two states live mode adds, which differ from every entry above in
		// a footer and in a body line — exactly the class this golden sees. It
		// is what caught #205 (a status column a cell wider, whole package
		// green) and #222 (a squad id column, whole client suite green), and a
		// live battle is drawn by no other golden in the repository: the lobby
		// screens live in cmd/hexarena-tui, so this is the only record of the
		// PvP drawing itself.
		"a live battle":         aLiveBattle(t, c),
		"a live battle waiting": aLiveBattleWaiting(t, c),
	}
}

// aLiveBattle is the turn in front on a battle **somebody else is driving**: the
// same board, the same roster and the same option list, under a footer that
// names neither u, n nor the save key because a live screen answers to none of
// them.
//
// Its own battle, like every other entry here — a live screen holds the same
// *battle.Battle pointer a local one does, so sharing one would step them all.
func aLiveBattle(t *testing.T, c Context) PlayScreen {
	t.Helper()
	local := withAFullLog(t, c, atABattleOf(t, c, 3))
	p := NewPlayScreen().Attach(c, PlayLive{
		Fight: local.Fight, Asking: local.Pending, Side: local.Side, Seed: local.Seed,
		Clock: aCountdown(PlayClockYou),
	})
	if !p.Live || p.Pending == nil {
		t.Fatal("the live battle attached to no turn, so this records a waiting screen twice")
	}
	drawn, footer := p.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PlayHeading)) {
		t.Fatalf("the live battle draws no heading of its own:\n%s", drawn)
	}
	if footer != c.Text(i18n.PlayLiveFooter) {
		t.Fatalf("the live battle draws the local footer, so this records an ordinary "+
			"battle twice:\n%s", footer)
	}
	return p
}

// aLiveBattleWaiting is the same screen with the turn on the other side of the
// wire, which is the one line live mode adds to the drawing.
//
// ⚠️ It is a separate entry rather than a variation because the tail is the
// section the whole budget is arranged around: where a local battle between
// turns draws nothing there and resolves in microseconds, this draws a row and
// can hold it for a whole allowance.
func aLiveBattleWaiting(t *testing.T, c Context) PlayScreen {
	t.Helper()
	local := withAFullLog(t, c, atABattleOf(t, c, 3))
	p := NewPlayScreen().Attach(c, PlayLive{
		Fight: local.Fight, Side: local.Side, Seed: local.Seed,
		Clock: aCountdown(PlayClockThem),
	})
	if p.Pending != nil {
		t.Fatal("the waiting live battle still holds a turn of its own")
	}
	drawn, _ := p.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PlayLiveWaiting)) {
		t.Fatalf("the waiting live battle says nothing about waiting:\n%s", drawn)
	}
	return p
}

// aCountdown is a clock part way through one side's turn, as the client hands
// one over.
//
// ⚠️ **The two numbers differ**, which is what makes the golden able to see them
// swapped: a fixture with the same figure on both sides would record a screen
// that had put the other player's clock where the reader's own goes and read
// exactly the same. The seat on turn is the one that has spent some of its
// allowance; the other has the whole of it, because the allowance is per prompt.
func aCountdown(waiting PlayClockSeat) PlayClock {
	const whole, spent = 90, 72
	if waiting == PlayClockThem {
		return PlayClock{Waiting: waiting, Yours: whole, Theirs: spent}
	}
	return PlayClock{Waiting: waiting, Yours: spent, Theirs: whole}
}

// # The played battle's six entries
//
// The twelfth screen in this package and the one a game client needs most. What
// is recorded is the six paths through View that share no line: the turn in
// front, the cell being chosen, the ending, the log scrolled back off its tail,
// the note a write leaves, and the screen with no pairing to open a battle on.
//
// ⚠️ **Every one of them builds its own battle, and that is a rule rather than
// tidiness.** PlayScreen holds a `*battle.Battle`, the one thing on a screen in
// this package that a copy does not copy — so a fixture handing several states
// one battle would step them all, and the state driven furthest would decide
// what the others drew. The client's own sweep shipped exactly that: three
// states shared a battle, the finished one played it out, and both battle
// footers were over the window for a release with every width test passing.
//
// ⚠️ **`a battle`, `aiming` and `a battle over` are three a side where the
// client's of the same name are one**, and that is the squad entries' asymmetry
// again: the client's fixture squad has a single member, so its board draws one
// roster row and never reaches the budget. Three is what makes the floor render
// a squeeze — the notice, a dropped board, a log with no rows — which is the
// half of this screen the arithmetic is about. The client's `a squeezed battle`
// pair has no counterpart here for the opposite reason: this golden sets the
// window itself, so every entry's floor render *is* the squeezed one.
//
// ⚠️ **`a saved battle` is a state NEITHER golden could draw before.** Measured
// over both, in both languages: `i18n.NoteWrote` and `i18n.NoteBattleVerify` on
// this screen came back at **nought hits in 8,201 and 3,098 lines**. The client
// cannot reach it — its fixture writes into a temp directory, so the note names
// a path whose length is not stable between two runs on one machine, which is
// why `everyScreen` leaves the state out and says so. Here the note is a
// **value** carrying a relative path, which is also what a real write produces
// in this package's own directory, so `noAbsolutePath` holds and the rows are
// recorded.
//
// ⚠️ **`a battle with no pairing` is the other one.** The client's fight guards
// its `p` on the catalogue holding a squad, so nothing there can open a battle
// on nothing; `Open` refuses two empty squads and says so, and this is the only
// record of that arm.

// aBattle is the turn in front: the board, the roster, the order line, the log
// and the option list under them.
func aBattle(t *testing.T, c Context) PlayScreen {
	t.Helper()
	p := withAFullLog(t, c, atABattleOf(t, c, 3))
	if drawn, _ := p.View(c); !strings.Contains(drawn, c.Text(i18n.PlayHeading)) {
		t.Fatalf("the battle draws no heading of its own:\n%s", drawn)
	}
	return p
}

// aBattleAiming is the second question: the same board with the cells the chosen
// skill may be pointed at under the option list.
//
// ⚠️ Its own battle, and the aim list is asserted rather than assumed: an option
// with one cell never opens the question, so a state built on the wrong row
// records an ordinary battle screen twice.
func aBattleAiming(t *testing.T, c Context) PlayScreen {
	t.Helper()
	p := withAFullLog(t, c, atABattleOf(t, c, 3))
	p.Aiming = true
	option := p.Pending.Options[p.Option]
	if len(option.Aims) == 0 {
		t.Fatalf("the option %q has nowhere to point, so the aim list is empty",
			option.Skill)
	}
	drawn, _ := p.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PlayAimAt, option.Skill)) {
		t.Fatalf("the aiming battle draws no aim list:\n%s", drawn)
	}
	return p
}

// aFinishedBattle is the ending, which replaces the option list and takes the
// over footer with it.
func aFinishedBattle(t *testing.T, c Context) PlayScreen {
	t.Helper()
	p := playedOut(t, c, atABattleOf(t, c, 3))
	drawn, _ := p.View(c)
	said := false
	for _, ending := range []i18n.Key{
		i18n.PlayWon, i18n.PlayLost, i18n.PlayDrawn, i18n.PlayEmptied,
	} {
		if strings.Contains(drawn, c.Text(ending)) {
			said = true
		}
	}
	if !said {
		t.Fatalf("the finished battle says nothing about how it ended:\n%s", drawn)
	}
	return p
}

// aScrolledBattleLog is the log walked back off its own tail, which is the one
// state that draws the position on the heading row.
//
// ⚠️ It has to be **built twice over**: a battle a few turns old has a history
// that fits its frame, so nothing is hidden and the position is correctly not
// drawn — and the floor this golden renders at gives a three-a-side board no log
// row at all. So the history is played out past any frame, and the frame is
// walked back with the key a reader would press, in a window the log has rows
// in. The position then draws at the roomy size and the notice names the log at
// the floor, which is both halves of the record.
func aScrolledBattleLog(t *testing.T, c Context) PlayScreen {
	t.Helper()
	tall := inAWindow(c, longLogHeight)
	p := withALongLog(t, tall, atABattleOf(t, tall, 3), longLogRows)
	for range 4 {
		p = playing(t, tall, p, "pgup")
	}
	if p.LogFollow || p.LogOffset == 0 {
		t.Fatalf("the log did not scroll back (following %v at %d), so the position "+
			"is drawn by nothing here", p.LogFollow, p.LogOffset)
	}
	roomy := c
	roomy.Width, roomy.Height = goldenSizes[len(goldenSizes)-1][0], goldenSizes[len(goldenSizes)-1][1]
	history, room, _ := theLogFrame(roomy, p)
	if room <= 0 || len(history) <= room {
		t.Fatalf("the roomy window holds the whole %d-row history in %d rows, so no "+
			"position is drawn", len(history), room)
	}
	return p
}

// aSavedBattle is the pair of notes a write leaves behind, drawn under the
// option list.
//
// ⚠️ The notes are a **value** and nothing here writes: the path is relative,
// which is what a save in this package's own data directory really produces and
// what keeps noAbsolutePath true. It is the one state neither golden held — see
// the note above for the measurement.
func aSavedBattle(t *testing.T, c Context) PlayScreen {
	t.Helper()
	p := withAFullLog(t, c, atABattleOf(t, c, 3))
	p.Notes = aSaveNote()
	rows := p.Wrote(c)
	if len(rows) == 0 {
		t.Fatal("the fixture note renders no row")
	}
	if drawn, _ := p.View(c); !strings.Contains(drawn, rows[0]) {
		t.Fatalf("the saved battle does not draw its own note:\n%s", drawn)
	}
	return p
}

// aBattleWithNoPairing is the screen a client with an empty catalogue opens: no
// battle, and the line saying nothing has been built.
func aBattleWithNoPairing(t *testing.T, c Context) PlayScreen {
	t.Helper()
	p := NewPlayScreen().Open(c, placement.Squad{}, placement.Squad{})
	if p.Fight != nil || p.Err != nil {
		t.Fatalf("two empty squads opened a battle (%v)", p.Err)
	}
	if drawn, _ := p.View(c); !strings.Contains(drawn, c.Text(i18n.SquadsEmpty)) {
		t.Fatalf("the unpaired battle says nothing about a side having to be built:\n%s", drawn)
	}
	return p
}

// addingAWork is the add-a-work form over the catalogue: five rows, the medium
// chooser among them, and the hint under them.
//
// Driven with the key an author would press rather than by writing the flag,
// which is the rule every state in this file follows — and it asserts it drew
// the form's own heading, because a flag set on a screen that had stopped
// reading it renders the listing again and passes every sweep over it.
func addingAWork(t *testing.T, c Context, works OriginsScreen) OriginsScreen {
	t.Helper()
	form := onOrigins(t, c, works, "a")
	if !form.Adding {
		t.Fatal("a did not open the add-a-work form")
	}
	if drawn, _ := form.View(c); !strings.Contains(drawn, c.Text(i18n.OriginFormHeading)) {
		t.Fatalf("the add-a-work form draws no heading of its own:\n%s", drawn)
	}
	return form
}

// anEmptyWorksCatalogue is the listing with nothing on it, which is a state a
// reader can genuinely arrive at: a fresh data directory has no origins.json,
// and forge.Load takes a missing file as an empty catalogue.
//
// ⚠️ It is here because that line had **never been drawn by anything**. Measured:
// neither this golden nor the client's held `i18n.OriginsEmpty` in either
// language, because every book either fixture loads has works in it — so the one
// sentence a first-run author sees had never been measured against the floor.
// Built by emptying the fields for the reason withNoTraitTaken empties a build's
// trait slot: nothing shipped need be deleted for a wording to be recorded.
func anEmptyWorksCatalogue(t *testing.T, c Context, works OriginsScreen) OriginsScreen {
	t.Helper()
	works.Origins, works.Counts = nil, nil
	drawn, _ := works.View(c)
	if !strings.Contains(drawn, c.Text(i18n.OriginsEmpty)) {
		t.Fatalf("the empty catalogue draws no line saying so, so the golden records an "+
			"ordinary works listing twice:\n%s", drawn)
	}
	return works
}

// aRefusedWork is the add-a-work form with the catalogue's own refusal under it:
// an id somebody has already used.
//
// The refusal is a **value** rather than the result of a write, which is the rule
// this file holds throughout — `Err` is a field, and the error is the one
// internal/forge would have handed back. It is the duplicate-id refusal rather
// than a bad year because that one is worded in internal/i18n
// (`ErrorOriginTaken`) in both languages, so the record measures two spellings; a
// raw Go error string would be the same English line twice and would measure
// nothing about the pair.
//
// ⚠️ It is here because how this line **draws** had never been recorded. The
// client drives it behaviourally — TestAddingAWorkWritesTheCatalog presses the
// same id twice and looks for the refusal in `screenContent` — but a
// strings.Contains is not a layout, and neither golden held `i18n.AddRefused` on
// this form in either language.
//
// ⚠️ **The line the form draws after a SUCCESSFUL write, `i18n.OriginAdded`, is
// deliberately NOT here and must not be added.** It names the file it wrote,
// through `Lib.OriginsPath()`, and `noAbsolutePath` below walks the whole
// recorded body — so an entry holding it would either record wherever this
// checkout happens to sit or would need the relative-directory trick the
// client's golden takes, which is the one thing this record's own doc comment
// says it does not do. That state stays a gap, stated rather than closed.
func aRefusedWork(t *testing.T, c Context, works OriginsScreen) OriginsScreen {
	t.Helper()
	taken := c.Lib.Origins().All()
	if len(taken) == 0 {
		t.Fatal("the shipped book holds no works, so no id can already be taken")
	}
	// Its own ResetForm rather than addingAWork's, because a form's Inputs slice is
	// SHARED between copies of the screen — the caveat OriginsScreen.Inputs carries
	// — and two golden entries writing through one slice would be two records of
	// whichever wrote last.
	form := onOrigins(t, c, works, "a")
	if !form.Adding {
		t.Fatal("a did not open the add-a-work form")
	}
	form.Err = &forge.OriginTakenError{ID: taken[0].ID}
	drawn, _ := form.View(c)
	if !strings.Contains(drawn, c.Text(i18n.AddRefused, c.Lang.Error(form.Err))) {
		t.Fatalf("the refused form draws no refusal, so the golden records an ordinary "+
			"add-a-work form twice:\n%s", drawn)
	}
	if !strings.Contains(drawn, taken[0].ID) {
		t.Fatalf("the refusal does not name the work it is about (%q):\n%s", taken[0].ID, drawn)
	}
	return form
}

// # The squad builder's seven entries, and why every one of them is a value
//
// The builder is the tenth screen in this package and the only one that **writes
// the author's own file**. Nothing here writes: `Saved` is a field, `Open` takes
// its baseline off whatever it is handed, and a picker is a literal the screen
// builds — so every state below is arranged the way `anEditedSkill` arranges its
// reported edit, as data rather than as the result of an edit.
//
// ⚠️ Each asserts it drew the line it exists for, as every hand-built state in
// this file does. A catalogue with no rows, a member on a character the book has
// lost and a picker over an empty learnset all render perfectly good screens that
// measure none of the code the entry was added for.

// aSquadOverTheShippedCast is one squad, built around whichever character the
// shipped book gives the most traits.
//
// The most traits, so the trait picker below has rows to draw: which character
// that is is a fact about the book, and naming one would tie this record to
// content an author is free to change.
func aSquadOverTheShippedCast(t *testing.T, lib *forge.Library) placement.Squad {
	t.Helper()
	characters := lib.Characters().All()
	if len(characters) == 0 {
		t.Fatal("the shipped book holds no characters, so no squad can be built")
	}
	found, most := 0, -1
	for index, character := range characters {
		held := len(character.PassivesAt(progression.LevelCap, progression.Furthest))
		if held > most {
			found, most = index, held
		}
	}
	character := characters[found]
	unit := placement.Placement{
		ID:        "mot",
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      hex.Offset{Col: hex.FormationCols - 1, Row: 1},
	}
	kit := character.SkillsAt(unit.Level, progression.Furthest)
	if len(kit) > cast.SkillSlots {
		kit = kit[:cast.SkillSlots]
	}
	unit.Skills = kit
	if traits := character.PassivesAt(unit.Level, progression.Furthest); len(traits) > 0 {
		unit.Passives = traits[:cast.TraitSlots]
	}
	return placement.Squad{
		ID: "do-thu", Name: "đội thử", Units: []placement.Placement{unit},
	}
}

// squadCatalogue is the listing with two squads on it, which is the state that
// measures the id column: it is sized over the rows **and** the header, so a
// listing with nothing in it draws neither.
//
// Two rather than one because the second is wider than the first, so a column
// measured off one row alone would come back a different number.
func squadCatalogue(t *testing.T, c Context, lib *forge.Library) SquadsScreen {
	t.Helper()
	s := NewSquadsScreen(c)
	first := aSquadOverTheShippedCast(t, lib)
	second := first.Clone()
	second.ID, second.Name = "do-thu-hai", "đội thử hai"
	s.Saved = []placement.Squad{first, second}
	drawn, _ := s.View(c)
	if strings.Contains(drawn, c.Text(i18n.SquadsEmpty)) {
		t.Fatalf("the catalogue records the empty listing, which the client's golden of "+
			"this name already holds:\n%s", drawn)
	}
	if !strings.Contains(drawn, second.ID) {
		t.Fatalf("the catalogue does not list %q, so its id column is measured over one "+
			"row:\n%s", second.ID, drawn)
	}
	return s
}

// squadInHand is the first of those squads taken up for editing: the id, the
// name, the member rows and the formation under them.
func squadInHand(t *testing.T, c Context, catalogue SquadsScreen) SquadsScreen {
	t.Helper()
	s := catalogue.Open(catalogue.Saved[0])
	if s.Mode != SquadEdit {
		t.Fatalf("opening a saved squad landed in %v", s.Mode)
	}
	if drawn, _ := s.View(c); !strings.Contains(drawn, c.Text(i18n.SquadAddMember)) {
		t.Fatalf("the squad under edit draws no row to add a member:\n%s", drawn)
	}
	return s
}

// squadMember is that squad's one member opened, which is the six-row form and
// the live formation beside it.
func squadMember(t *testing.T, c Context, catalogue SquadsScreen) SquadsScreen {
	t.Helper()
	s := squadInHand(t, c, catalogue).EditUnit(0)
	if s.Mode != SquadUnit {
		t.Fatalf("opening a member landed in %v", s.Mode)
	}
	if drawn, _ := s.View(c); !strings.Contains(drawn, c.Text(i18n.SquadFieldSlot)) {
		t.Fatalf("the member draws no slot row:\n%s", drawn)
	}
	return s
}

// deepMember is the same member standing in the rank whose name is longest.
//
// The rank is looked up rather than named, for the reason skillBlurb takes the
// widest reading: which of the three words is longest is a fact about a language,
// so naming one would record English and skip Vietnamese. The cell is written on
// the unit under edit rather than committed, which is also the state the live
// grid is for — the picture follows the arrows.
func deepMember(t *testing.T, c Context, member SquadsScreen) SquadsScreen {
	t.Helper()
	found, most := member.Unit.Slot, 0
	for _, slot := range FormationSlots() {
		if width := lipgloss.Width(RankLabel(c, slot)); width > most {
			found, most = slot, width
		}
	}
	member.Unit.Slot = found
	if drawn, _ := member.View(c); !strings.Contains(drawn, RankLabel(c, found)) {
		t.Fatalf("the deep member does not say it stands in %q:\n%s",
			RankLabel(c, found), drawn)
	}
	return member
}

// heldBackMember is the same member holding a character the cast has taken out
// of the builder's list, which is the one state that draws the held-back line.
//
// It edits the screen's own copy of the cast rather than the book, which is what
// withNoTraitTaken does and for the same reason: nothing shipped need be hidden
// for the wording to be recorded. The slice is copied because every state here
// shares one backing array.
func heldBackMember(t *testing.T, c Context, member SquadsScreen) SquadsScreen {
	t.Helper()
	characters := append([]cast.Character(nil), member.Characters...)
	held := false
	for index := range characters {
		if characters[index].ID == member.Unit.Character {
			characters[index].Hidden, held = true, true
		}
	}
	if !held {
		t.Fatalf("the fixture member names %q, which is not in the cast the screen holds",
			member.Unit.Character)
	}
	member.Characters = characters
	if drawn, _ := member.View(c); !strings.Contains(drawn, c.Text(i18n.SquadHeldBack)) {
		t.Fatalf("the held-back member draws no line saying so, so this records an "+
			"ordinary member twice:\n%s", drawn)
	}
	return member
}

// theForkedMember is the member form on the shipped character whose evolution
// line forks, at a level the fork is open at, naming no arm — and theForkedSquad
// is the squad behind it, whose member row draws the same fact in a row rather
// than in a field.
//
// ⚠️ **Every squad entry above these is a line that does not fork, and that is
// what let the defect ship.** aSquadOverTheShippedCast picks the character with
// the most traits and fields it at the cap with an empty stage, so the form row
// read "furthest" and was right. On a fork "furthest" names neither end, and the
// two loadout lists silently drop every arm-gated skill and trait — neither of
// which any record in this file could see.
//
// Two entries because the two depths draw it differently: the member has the
// chooser value and the line naming the arms under the fields, and the squad has
// only the member's row, which is clipped to the window and reads an empty stage
// through a *different* lookup — a member that is not the one under edit.
//
// The state is built by aForkedMember in squadfork_test.go rather than here, for
// the reason every other builder is shared: it is the same value those tests
// assert against, so a record and a test cannot come to be about two different
// screens.
func theForkedMember(t *testing.T, c Context, lib *forge.Library) SquadsScreen {
	t.Helper()
	s := aForkedMember(t, c, lib)
	drawn, _ := s.View(c)
	if !strings.Contains(drawn, c.Text(i18n.SquadForkUnnamed)) {
		t.Fatalf("the forked member draws no fork in its form row, so this entry "+
			"records an ordinary member twice:\n%s", drawn)
	}
	for _, line := range forkNote(c, s.unnamedArms(s.Unit)) {
		if !strings.Contains(drawn, line) {
			t.Fatalf("the forked member does not draw %q, so the line this entry "+
				"exists for is recorded by nothing:\n%s", line, drawn)
		}
	}
	return s
}

func theForkedSquad(t *testing.T, c Context, lib *forge.Library) SquadsScreen {
	t.Helper()
	s := aForkedMember(t, c, lib).Commit()
	s.Mode = SquadEdit
	if drawn, _ := s.View(c); !strings.Contains(drawn, c.Text(i18n.SquadForkUnnamed)) {
		t.Fatalf("the forked squad's member row draws no fork, so this entry records "+
			"an ordinary squad twice:\n%s", drawn)
	}
	return s
}

// squadKitPicker and squadTraitPicker are the two lists the member raises,
// **raised** rather than built: they are the one pair of picker entries in this
// file that a screen here composes, since the builder is the only raiser that
// has moved.
func squadKitPicker(t *testing.T, c Context, member SquadsScreen) *PickState {
	t.Helper()
	raised := member.OpenSkills()
	if raised == nil {
		t.Fatal("the member names no character, so its kit list will not open")
	}
	picker := raised.Raise()
	if len(picker.Options) == 0 {
		t.Fatal("the kit picker holds no rows, so it records an empty list")
	}
	return picker
}

func squadTraitPicker(t *testing.T, c Context, member SquadsScreen) *PickState {
	t.Helper()
	raised := member.OpenPassives()
	if raised == nil {
		t.Fatal("the member names no character, so its trait list will not open")
	}
	picker := raised.Raise()
	if len(picker.Options) == 0 {
		t.Fatal("the trait picker holds no rows; the fixture picks the character with " +
			"the most traits, so the shipped book has none at all")
	}
	return picker
}

// # The skill listing's seven states, and what each is here for
//
// The listing is the ninth screen in this package and the busiest: it is a
// listing, a form over that listing, a diagram over that form, and a typed
// filter that is a mode of its own. The states below are the paths through View
// that share no line with each other, which is the same rule the picker's five
// were chosen under.
//
// ⚠️ **The three filter states are driven with the keys an author would press**,
// exactly as the client's fixture drives them, because the query is what decides
// which rows there are — a hand-set field would record this test's idea of the
// filter rather than the one `/` opens. The client's fixture has twice measured a
// screen's early exit instead of the screen; a driven state cannot do that.
//
// ⚠️ Each asserts it drew the line it exists for. A registered state that renders
// nothing passes every sweep over it, and two of these (an empty result, a
// reported edit) draw a *different* line rather than no line, so the failure is
// silent in both directions.

// addingASkill is the form over an empty draft: every field at its default,
// which is the state `a` reaches.
func addingASkill(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	form := pressOn(t, c, listing, "a")
	if !form.Adding || form.Editing != "" {
		t.Fatalf("a did not open the new-skill form (adding %v, editing %q)",
			form.Adding, form.Editing)
	}
	if drawn, _ := form.View(c); !strings.Contains(drawn, c.Text(i18n.SkillFormHeading)) {
		t.Fatalf("the new-skill form draws no heading of its own:\n%s", drawn)
	}
	return form
}

// editingASkill is the same form over a skill that already exists, which is the
// widest it ever draws: every field prefilled from the book rather than empty.
func editingASkill(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	if len(listing.Skills) == 0 {
		t.Fatal("the shipped book holds no skills to edit")
	}
	form := listing.Prefill(c, listing.Skills[0])
	if form.Editing != listing.Skills[0].ID {
		t.Fatalf("the form opened over %q, want %q", form.Editing, listing.Skills[0].ID)
	}
	if drawn, _ := form.View(c); !strings.Contains(drawn, c.Text(i18n.SkillFormEditHeading)) {
		t.Fatalf("the edit form draws the new-skill heading:\n%s", drawn)
	}
	return form
}

// anEditedSkill is the listing as an edit leaves it: two lines rather than one,
// the second of which is the damage before and after.
//
// The change is built by hand rather than written, because nothing in this
// package writes: the books are loaded straight out of internal/seed/data and a
// golden that edited them would edit the repository. What it needs is a
// forge.SkillChange, which is four values and no file.
func anEditedSkill(t *testing.T, c Context, lib *forge.Library, listing SkillsScreen) SkillsScreen {
	t.Helper()
	if len(listing.Skills) == 0 {
		t.Fatal("the shipped book holds no skills to edit")
	}
	before := listing.Skills[0]
	after := before
	after.Power = before.Power*2 + 1000
	reported := listing
	reported.Edited = &forge.SkillChange{
		Before: before, After: after,
		BeforeDamage: lib.PreviewDamage(before),
		AfterDamage:  lib.PreviewDamage(after),
	}
	if !reported.Edited.MovesDamage() {
		t.Fatal("the fixture edit moves no damage, so the second line is drawn by nothing")
	}
	drawn, _ := reported.View(c)
	if want := c.Text(i18n.SkillEdited, after.ID, lib.SkillsPath()); !strings.Contains(drawn, want) {
		t.Fatalf("the listing does not report the edit %q:\n%s", want, drawn)
	}
	return reported
}

// theFilterOpen is the field just opened with nothing in it, which says what to
// type rather than how much of the book is left.
func theFilterOpen(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	opened := pressOn(t, c, listing, "/")
	if !opened.Filtering {
		t.Fatal("/ did not open the filter, so its three states are drawn by nothing here")
	}
	if drawn, _ := opened.View(c); !strings.Contains(drawn, c.Text(i18n.SkillsFilterPrompt)) {
		t.Fatalf("the opened filter says nothing about what to type:\n%s", drawn)
	}
	return opened
}

// theFilterFinding is a query that has found several rows: the row above says
// how much of the book is left, and the listing under it is narrowed.
//
// ⚠️ The fixture's own discrimination is asserted, which no assertion downstream
// can see: a query that has quietly stopped matching turns this into a second
// copy of the state below, and both would still render.
func theFilterFinding(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	found := typeInto(t, c, theFilterOpen(t, c, listing), someSkillQuery)
	rows, all := len(found.Rows()), len(found.Skills)
	if rows < 2 || rows >= all {
		t.Fatalf("the query %q finds %d of %d skills, so the filtered listing is not "+
			"a narrowed one", someSkillQuery, rows, all)
	}
	return found
}

// theFilterFindingNothing is a query nothing answers to, which says so where the
// rows would have been and draws no column header over them.
func theFilterFindingNothing(t *testing.T, c Context, listing SkillsScreen) SkillsScreen {
	t.Helper()
	nothing := typeInto(t, c, theFilterOpen(t, c, listing), noSkillQuery)
	if found := len(nothing.Rows()); found != 0 {
		t.Fatalf("the query %q finds %d skills, so the empty result is drawn by nothing",
			noSkillQuery, found)
	}
	if drawn, _ := nothing.View(c); !strings.Contains(drawn, c.Text(i18n.SkillsFilterNothing)) {
		t.Fatalf("the empty listing does not say the filter found nothing:\n%s", drawn)
	}
	return nothing
}

// theShapeDiagram is the board with a shape's coverage marked on it, which is a
// state of the form rather than a screen of its own.
//
// The shape is the one that catches the most cells, for the reason kitPicker
// takes the most-refused carrier: a golden of a layout is for the case that
// spends the most of it, and a single-cell shape draws a board with one mark.
func theShapeDiagram(t *testing.T, c Context, lib *forge.Library, listing SkillsScreen) SkillsScreen {
	t.Helper()
	shapes := lib.PatternNames()
	if len(shapes) == 0 {
		t.Fatal("the shipped book declares no shapes")
	}
	widest, most := shapes[0], -1
	for _, name := range shapes {
		coverage, err := lib.ShapeCoverage(name, defaultSkillTarget)
		if err != nil {
			t.Fatalf("coverage of %s: %v", name, err)
		}
		if coverage.Covered() > most {
			widest, most = name, coverage.Covered()
		}
	}
	diagram := addingASkill(t, c, listing)
	diagram.Field = SkillFieldShape
	diagram.ShapeIndex = IndexOf(shapes, widest)
	diagram.ShapeDrawn = true
	drawn, _ := diagram.View(c)
	if !strings.Contains(drawn, shapeSplashMark) {
		t.Fatalf("the diagram for %s marks no splash cell, so it records a board with "+
			"one mark on it:\n%s", widest, drawn)
	}
	return diagram
}

// # The five picker states, and why they are built here rather than raised
//
// The multi-select is the eighth screen in this package and the first with no
// listing of its own: it is handed a list, so a picker is a *state* rather than
// a screen with one shape, and which state is recorded is a decision.
//
// ⚠️ **They are hand-built and the client's are raised, and the names are the
// same on purpose.** The three screens that raise a picker — the character form,
// the skill form, the squad builder — are all still in cmd/hexforge-tui, so a
// state reached through one of them would be a state this package cannot make.
// What is recorded here is therefore the *drawing* under a list that says what
// this entry is for, and the client's golden of the same name is the same screen
// as its own raiser leaves it. Two records of one screen, which is the whole
// arrangement of this file.
//
// The five are the paths through View that share no line with each other: rows
// carrying a refusal and a detail column (the kit), rows with a filter line over
// them (the allowlist), that filter narrowed (the filtered one), a field and its
// percentage under the list (the statuses), and the reading pane, which replaces
// the list outright.
//
// ⚠️ Each asserts it drew the line it exists for, as every hand-built state in
// this file does. A picker built with the wrong options still renders a
// perfectly good screen — an empty list, an unmarked column, a filter that hides
// nothing — and every sweep over it would pass.

// kitPicker is the character form's kit picker over the shipped book, for the
// carrier that refuses the most of it.
//
// The most refused rather than the first character, for the reason skillBlurb
// takes the widest reading: what a golden of a layout is for is the case that
// spends the most of it, and here that is a list drawing both sorts of row. A
// carrier answering nothing restricts nothing, so the empty one draws a list
// with no marks on it at all.
func kitPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	characters := lib.Characters().All()
	if len(characters) == 0 {
		t.Fatal("the shipped book holds no characters to build a carrier from")
	}
	found, most := forge.Carrier{}, -1
	for _, character := range characters {
		who := forge.Carrier{
			ID: character.ID, Archetype: character.Archetype,
			Affinity: character.Element, HasAffinity: true,
			Species: character.Species, Origin: character.Origin,
		}
		refused := 0
		for _, option := range KitOptions(lib, who) {
			if option.Refusal != nil {
				refused++
			}
		}
		if refused > most {
			found, most = who, refused
		}
	}
	picker := (&PickState{
		Title: i18n.PickerKitTitle, Kind: PickSkills, Options: KitOptions(lib, found),
	}).Raise()
	offered := len(picker.Options) - most
	if most == 0 || offered == 0 {
		t.Fatalf("the kit picker refuses %d of %d rows, so it records a list with "+
			"only one sort of row on it", most, len(picker.Options))
	}
	if drawn, _ := picker.View(c); !strings.Contains(drawn, "!") {
		t.Fatalf("no row of the kit picker carries the unavailable mark:\n%s", drawn)
	}
	return picker
}

// allowlistPicker is a restriction's character list: no slots, no refusals, and
// the one picker with a filter over it.
func allowlistPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	picker := (&PickState{
		Title: i18n.PickerCharactersTitle, Hint: i18n.PickerAllowlistHint,
		Footer: i18n.PickerFilterFooter, Kind: PickCharacters,
		Options: CharacterOptions(lib), Groups: lib.OriginIDs(),
	}).Raise()
	if len(picker.Groups) == 0 {
		t.Fatal("the allowlist picker carries no works, so it draws no filter line")
	}
	drawn, _ := picker.View(c)
	showing := c.Text(i18n.PickerShowing, picker.groupName(c), len(picker.Visible()),
		len(picker.Options))
	if !strings.Contains(drawn, showing) {
		t.Fatalf("the allowlist picker draws no filter line %q:\n%s", showing, drawn)
	}
	return picker
}

// filteredPicker is that same list narrowed to one work, which is a line the
// unfiltered one does not draw and a row count the unfiltered one does not have.
//
// It is a second picker rather than the one above with the filter stepped on,
// because NextFilter mutates in place: sharing one would put the entry beside
// this into a state it is not here to measure.
func filteredPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	picker := allowlistPicker(t, c, lib)
	picker.NextFilter()
	group := picker.Group()
	if group == "" {
		t.Fatal("stepping the filter left it on everything, so this records the " +
			"unfiltered list twice")
	}
	if len(picker.Visible()) == len(picker.Options) {
		t.Fatalf("the %q filter hides nothing of the %d rows", group, len(picker.Options))
	}
	if drawn, _ := picker.View(c); !strings.Contains(drawn, group) {
		t.Fatalf("the filtered picker does not name %q:\n%s", group, drawn)
	}
	return picker
}

// statusPicker is what a skill inflicts: the one picker of the five carrying a
// field, and the only entry here whose body has a row under the list.
//
// One status is chosen so the answer line under the list is the ids rather than
// the nothing-chosen wording, and the field is left empty so the placeholder and
// its percentage reading are what is recorded — which is the state a raise
// leaves it in, and the one the reading has to be right for.
func statusPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	statuses := StatusOptions(lib)
	if len(statuses) == 0 {
		t.Fatal("the shipped book declares no statuses")
	}
	picker := (&PickState{
		Title: i18n.PickerStatusesTitle, Hint: i18n.PickerStatusHint,
		Footer: i18n.PickerStatusFooter, Kind: PickStatuses,
		Options: statuses, Chosen: []string{statuses[0].ID},
		Typed: NumberField(c.Style.Plain, forge.DefaultApplicationChance),
		Label: i18n.PickerChance,
	}).Raise()
	drawn, _ := picker.View(c)
	if !strings.Contains(drawn, c.Text(i18n.PickerChance)) {
		t.Fatalf("the status picker does not name its chance field:\n%s", drawn)
	}
	if !strings.Contains(drawn, statuses[0].ID) {
		t.Fatalf("the status picker does not draw %q as chosen:\n%s", statuses[0].ID, drawn)
	}
	return picker
}

// readingPicker is the kit picker with the description of the row under the
// cursor in front of the list, which is the picker's other state and shares no
// line with the list.
//
// It is the kit rather than the traits because the shipped skill book is the
// list this pane is most often reached from, and the row is walked to the
// longest reading for the reason skillBlurb is: this is the entry that records
// the line saying there is more to read.
func readingPicker(t *testing.T, c Context, lib *forge.Library) *PickState {
	t.Helper()
	picker := kitPicker(t, c, lib)
	longest, most := 0, 0
	for index, option := range picker.Visible() {
		declared, err := lib.Skills().Lookup(option.ID)
		if err != nil {
			continue
		}
		if lines := len(SkillLines(c, declared)); lines > most {
			longest, most = index, lines
		}
	}
	picker.Cursor, picker.Reading = longest, true
	id := picker.Visible()[longest].ID
	if drawn, _ := picker.View(c); !strings.Contains(drawn, id) {
		t.Fatalf("the reading pane does not name %q, so it records the "+
			"nothing-to-pick arm:\n%s", id, drawn)
	}
	return picker
}

// skillBlurb is the description screen over the shipped skill whose reading
// takes the most lines.
//
// The widest reading rather than the first row, for the reason the client's
// fixture picks its widest element and its widest trait row: what a golden of a
// layout is for is the case that spends the most of it. Measured in **one**
// language whatever the reader's is, so both records describe the same skill —
// which skill is the widest is a fact about the book, and a record that changed
// subject between the two halves of the file would be two records.
//
// The position is where the skill sits in the book, exactly as the listing that
// raises this counts it: the raiser hands over At and Of, and Of at nought is how
// it says there was nothing to describe.
func skillBlurb(t *testing.T, c Context, lib *forge.Library) BlurbScreen {
	t.Helper()
	skills := lib.Skills().Skills()
	if len(skills) == 0 {
		t.Fatal("the shipped book holds no skills, so there is nothing to describe")
	}
	found, most := 0, 0
	for index, declared := range skills {
		lines := len(strings.Split(i18n.Vi.Describe(declared, lib.Patterns()), "\n"))
		if lines > most {
			found, most = index, lines
		}
	}
	blurb := BlurbScreen{Subject: Subject{
		Kind: SkillSubject, ID: skills[found].ID, At: found + 1, Of: len(skills),
	}}
	if drawn, _ := blurb.View(c); !strings.Contains(drawn, skills[found].ID) {
		t.Fatalf("the skill blurb does not name %q, so the golden records the "+
			"describer's nothing-to-describe arm:\n%s", skills[found].ID, drawn)
	}
	return blurb
}

// traitBlurb is the description screen over the shipped character carrying the
// most traits at the level cap.
//
// The most traits, because that is the state the screen's scroll exists for: five
// at the cap wrap past a 120x24 window, so this is the entry that records the
// line saying there is more to read. It is recorded **unscrolled**, which is
// where a raise leaves it.
func traitBlurb(t *testing.T, c Context, lib *forge.Library) BlurbScreen {
	t.Helper()
	characters := lib.Characters().All()
	found, most := 0, 0
	for index, character := range characters {
		held := len(lib.KitPassives(
			character.PassivesAt(progression.LevelCap, progression.Furthest)))
		if held > most {
			found, most = index, held
		}
	}
	if most < 2 {
		t.Fatalf("the widest shipped character carries %d traits at the cap, so this "+
			"entry records the carries-nothing line rather than the sentences", most)
	}
	blurb := BlurbScreen{Subject: Subject{
		Kind: CharacterSubject, ID: characters[found].ID, Level: progression.LevelCap,
		At: found + 1, Of: len(characters),
	}}
	held := lib.KitPassives(
		characters[found].PassivesAt(progression.LevelCap, progression.Furthest))
	drawn, _ := blurb.View(c)
	if !strings.Contains(drawn, held[0].ID) {
		t.Fatalf("the trait blurb does not name %q, the first trait %s carries at the "+
			"cap:\n%s", held[0].ID, characters[found].ID, drawn)
	}
	return blurb
}

// # The art preview's entry, and the two halves of it a golden cannot hold
//
// theArtPreview is the picture the cast browser raises with `p`, over the first
// row of the shipped cast at the level cap — which is where a raise leaves it,
// and the same state both clients register under this name.
//
// ## Why the record is monochrome, and why that is not a choice
//
// Goldens here are taken under NO_COLOR, so `Palette.Plain` is true and the
// screen draws `rampCell`: one character a cell out of `Ramp`, and no escape
// codes. That is also the only affordable record there is. Measured, one
// character in one language: the plain drawing is **19 lines / 2.0 KB** at the
// floor and **55 lines / 8.4 KB** at 160x60, while the *same* lines coloured are
// **14 KB** and **128 KB**, because every cell carries its own truecolor
// sequence. Two languages x two sizes is therefore about **21 KB** of record
// here and would be about **284 KB** if the palette ever came back coloured —
// which is why `Plain` is asserted below rather than assumed.
//
// ## Both windows, deliberately
//
// The floor is where `previewChrome` bites: `Height - 8` leaves 16 rows, so the
// art is rasterised into a 32-pixel-tall box and the drawing is the squeezed one.
// The roomy size is the same picture with nothing taken away. A ramp character
// that moved in both is the rasteriser or the weights moving; one that moved only
// at the floor is the row budget moving, which is the same reading every other
// entry's two sizes are for.
//
// ## ⚠️ It is a same-machine record, and that is a narrower claim than the rest
// of this file makes
//
// The rasterisation **is** reproducible here — the whole shipped cast digests
// byte for byte identically across separate `go test` processes, in both
// drawings, and twice in one process off two separately loaded libraries;
// `internal/forge.rasteriseSVG` reads no clock, no map and no environment. What
// is **not** established is another architecture: `rasterx` calls `math.Sin`,
// `math.Cos`, `math.Atan2` and `math.Tan`, which is the family Go has had
// per-architecture assembly for. `math.Sqrt` is an IEEE-exact instruction and the
// shipped art is `vtracer` output — beziers and polygons — so a transcendental
// may never be reached at all, but "same bytes on every machine" is a promise
// `internal/core` makes and this drawing does not. So the assertions below are
// about **structure** — that the rows are full-width, that they are drawn out of
// the ramp and nothing else — and the exact pixel field is left to the record,
// where a diff on another machine is a finding to read rather than a broken gate.
//
// ## ⚠️ What this entry cannot see at all
//
// `blockCell` — the coloured half of the drawing, which is what a reader actually
// looks at. NO_COLOR means this record never reaches it, so swapping `▀` for `▄`
// there moves nothing in this file. That is held by
// TestEachPixelIsDrawnInItsOwnHalfOfTheCell in preview_test.go instead, and the
// weights behind the ramp by TestTheRampWeighsGreenOverRedOverBlue beside it.
//
// ## What the width sweep is told about it
//
// Nothing, and that is the honest answer rather than a gap. `CLAUDE.md` § the
// TUI width rule splits prose, which takes `MinWidth`, from data, which takes
// `UsableWidth()`. The art is neither: `picture` asks for exactly
// `UsableWidth() - 4` cells and `cellRows` writes one cell per pixel column after
// a two-space indent, so **every row is `UsableWidth() - 2` wide by
// construction** and a floor assertion over it would pass on nothing it could
// ever fail. What the screen does put in front of the width sweep is its wording —
// the heading, the art/level/stage line, the missing and unreadable lines, the
// footer — and those take `MinWidth` like every other sentence. So the clients'
// sweeps exempt the picture rows by their alphabet and measure the rest, and the
// only assertion made about the drawing's width is the one below: that it fills
// the window it was given, so the record is of a full-width drawing rather than
// of a stub.
func theArtPreview(t *testing.T, c Context, lib *forge.Library) PreviewScreen {
	t.Helper()
	if !c.Style.Plain {
		t.Fatal("the golden's palette draws colour, so this entry would record a " +
			"truecolor sequence per cell — about 128 KB for the roomy render alone")
	}
	browser := NewBrowseScreen(lib)
	browser.Level = progression.LevelCap
	subject := browser.Subject()
	if subject.Of == 0 {
		t.Fatal("the shipped cast has no rows, so there is nothing to preview")
	}
	preview := NewPreviewScreen()
	preview.Subject = subject
	drawn, _ := preview.View(c)
	if strings.Contains(drawn, c.Text(i18n.ArtMissing)) {
		t.Fatalf("%s has no art on disk, so this entry records the missing-art line "+
			"rather than a drawing:\n%s", subject.ID, drawn)
	}
	painted, widest := 0, 0
	for _, row := range strings.Split(drawn, "\n") {
		if !aRampRow(row) {
			continue
		}
		painted++
		if width := lipgloss.Width(row); width > widest {
			widest = width
		}
	}
	// ⚠️ The count and the width are both asserted, and the width is the half a
	// count cannot give: a preview that had quietly stopped filling the window
	// would draw its rows, pass a count, and record a picture in a corner.
	if painted < 4 {
		t.Fatalf("the preview drew %d rows of ramp, so this entry records its heading "+
			"and nothing else:\n%s", painted, drawn)
	}
	if want := c.UsableWidth() - 2; widest != want {
		t.Errorf("the widest drawn row is %d cells rather than the %d the art is sized "+
			"to, so this is not a record of a full-width drawing", widest, want)
	}
	return preview
}

// aRampRow reports whether a line is drawn art rather than wording.
//
// Told apart by its alphabet, which is what `Ramp` is exported for: the picture
// is the only thing on that screen made of nothing but those ten characters,
// while every other row carries letters. Counting rows from the bottom instead
// would be a second copy of `previewChrome`'s arithmetic, which is the sort of
// duplicate that constant's own comment records having got wrong once already.
//
// The coloured drawing's two half blocks are not in it, and need not be: this
// package's golden is plain by construction, and the client helper that has to
// answer for both — cmd/hexforge-tui's aPictureRow — names them.
func aRampRow(row string) bool {
	if strings.TrimSpace(row) == "" {
		return false
	}
	for _, letter := range row {
		if !strings.ContainsRune(Ramp, letter) {
			return false
		}
	}
	return true
}

// # The fork, and why three more entries rather than one
//
// theForkedRaise is the cast browser sitting on the one shipped character whose
// evolution line forks, at a level the fork is open at, with the arm a chooser
// opens on in front — which is the state a raise leaves, exactly as the linear
// entries above are.
//
// ⚠️ **Every entry above this one is a line that does not fork, and that is what
// let the defect ship.** progression.Line.StageAt refuses to name a form when a
// level has reached two grown ones, on purpose — handing a reader whichever arm
// the file lists last is a wrong stat line, a wrong picture and a wrong trait
// list with nothing saying so. All three read-only views asked it anyway, so the
// art preview drew that refusal in red instead of a picture and the detail pane
// ended at the row with it in. Nothing in this record could see it: the browse
// entry opens on the first row of the cast and the preview and the blurb are
// raised from there, and the first row is linear.
//
// Three entries and not one because the three views fail differently. The
// preview drew the refusal, the detail pane stopped at it, and the blurb drew
// **neither arm's traits and said nothing at all** — the quiet form of the same
// fault, and the one a screenshot would not have caught. What each records now is
// the form row, which is the whole of what "the reader chose this arm" looks like
// on screen.
//
// The level is forkLevel rather than the cap for the reason form_test.go gives:
// the cap is where every other entry sits, and a fork that only worked there
// would be a fork nothing walked into.
func theForkedRaise(t *testing.T, lib *forge.Library) BrowseScreen {
	t.Helper()
	character, level := theShippedFork(t, lib)
	return browserOn(t, lib, character.ID, level)
}

// theForkedCastRow is the detail pane over that character: the form row, and the
// picture, traits and stat line under it that the form decides.
func theForkedCastRow(t *testing.T, c Context, lib *forge.Library) BrowseScreen {
	t.Helper()
	browser := theForkedRaise(t, lib)
	drawn, _ := browser.View(c)
	if !strings.Contains(drawn, c.Text(i18n.FormChoice,
		browser.Subject().Stage, 1, len(FormArms(rowUnder(t, browser), browser.Level)))) {
		t.Fatalf("the forked cast row draws no form row, so this entry records an "+
			"ordinary detail pane twice:\n%s", drawn)
	}
	return browser
}

// theForkedArtPreview is the picture of the arm in front.
//
// It carries theArtPreview's whole argument — monochrome because a golden is
// taken under NO_COLOR, structural rather than exact because rasterx reaches for
// transcendentals and this is therefore a same-machine record — and adds one
// thing that entry cannot hold: that there is a drawing here at all. Before this
// change the same keystroke over this character produced one red line.
func theForkedArtPreview(t *testing.T, c Context, lib *forge.Library) PreviewScreen {
	t.Helper()
	if !c.Style.Plain {
		t.Fatal("the golden's palette draws colour, so this entry would record a " +
			"truecolor sequence per cell")
	}
	preview := NewPreviewScreen()
	preview.Subject = theForkedRaise(t, lib).Subject()
	drawn, _ := preview.View(c)
	if strings.Contains(drawn, c.Text(i18n.ArtMissing)) {
		t.Fatalf("%s has no art on disk for the arm in front, so this entry records "+
			"the missing-art line rather than a drawing:\n%s", preview.Subject.ID, drawn)
	}
	painted := 0
	for _, row := range strings.Split(drawn, "\n") {
		if aRampRow(row) {
			painted++
		}
	}
	if painted < 4 {
		t.Fatalf("the forked preview draws %d rows of art, which is a stub rather than "+
			"a picture:\n%s", painted, drawn)
	}
	return preview
}

// theForkedTraitBlurb is what that character carries as the arm in front.
//
// ⚠️ This is the entry that records a **silent** wrong answer having been fixed.
// The screen never refused: cast.Character.form falls back to the empty stage
// when a level names no single one, so a trait gated on either arm simply did not
// match and the reader was shown a list belonging to neither. A record of it is
// the only thing that can tell that state from a right one.
func theForkedTraitBlurb(t *testing.T, c Context, lib *forge.Library) BlurbScreen {
	t.Helper()
	blurb := BlurbScreen{Subject: theForkedRaise(t, lib).Subject()}
	drawn, _ := blurb.View(c)
	if strings.Contains(drawn, c.Text(i18n.BlurbTraitNone)) {
		t.Fatalf("%s carries no traits at level %d, so this entry records the empty "+
			"reading rather than a fork:\n%s", blurb.Subject.ID, blurb.Subject.Level, drawn)
	}
	return blurb
}

// rowUnder is the character the browser's cursor is on.
func rowUnder(t *testing.T, browser BrowseScreen) cast.Character {
	t.Helper()
	rows := browser.Rows()
	if len(rows) == 0 {
		t.Fatal("the browser lists nothing")
	}
	return rows[Clamp(browser.Cursor, 0, len(rows)-1)]
}

// startOverTheShippedBooks is a Context over internal/seed/data itself.
//
// The rest of this package's tests go through `start`, which copies the data into
// a temp directory and injects the fixture cast — they need characters of their
// own to make an assertion about. A golden needs the opposite: the cast the
// repository actually ships, so the diff is the design record, and no temp path
// anywhere near the bytes. Nothing here writes, so a direct load is safe as well
// as simpler.
//
// NO_COLOR, as every fixture in this package sets: the styles then render as
// plain text, which is the whole reason a golden of a styled screen is readable
// at all.
func startOverTheShippedBooks(t *testing.T, lang i18n.Lang) (Context, *forge.Library) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load %s: %v", shippedDataDir, err)
	}
	return Context{
		Lib: lib, Lang: lang, Style: NewPalette(plainHere()),
		Width: MinWidth, Height: MinHeight,
		// Authoring, because this golden records the screens **as the authoring
		// tool draws them** — the skill form, the add-a-work form and the squad
		// builder are three of the twelve, and none of them opens without it.
		// The read-only spellings of the three footers those screens switch are
		// recorded in cmd/hexarena-tui's golden, which is the client that draws
		// them. ⚠️ Dropping this line moves this golden by three footers and no
		// body, which is exactly loud enough.
		Authoring: true,
	}, lib
}

// noAbsolutePath refuses a golden carrying this machine in it.
//
// ⚠️ It cannot fail today and is here for that reason. The books are loaded from
// a relative directory, no screen in this package prints one, and the two states
// are built in memory — so this holds **by construction**, and a property that
// holds by construction is a property a later change breaks without saying so. A
// golden naming `/var/folders/…` cannot be committed, and one naming a directory
// whose length varies cannot be reproduced twice on the same machine.
//
// The rooted-path rule is the client's: a separator at the front and another one
// further along, which is what tells a path from the bare `/` that several
// footers name as a key.
func noAbsolutePath(t *testing.T, body string) {
	t.Helper()
	separator := string(filepath.Separator)
	temp := filepath.Clean(os.TempDir())
	for number, line := range strings.Split(body, "\n") {
		if strings.Contains(line, temp) {
			t.Fatalf("line %d of the golden names the temp directory:\n%s", number+1, line)
		}
		for _, word := range strings.Fields(line) {
			if !strings.HasPrefix(word, separator) || !strings.Contains(word[len(separator):], separator) {
				continue
			}
			t.Fatalf("line %d of the golden names a filesystem path, which cannot be "+
				"committed — find where it enters and give it a relative name:\n%s",
				number+1, line)
		}
	}
}

// firstDifference names what moved: the render it happened under, the line
// number, and the two lines.
//
// The house style for a golden here is to print the whole of what was drawn,
// which works for a table and does not for thousands of lines of screen. The
// banner is what makes a single line readable instead — a diff that says
// `vi | 120x24 | statuses` has already told the reader which screen to look at.
//
// It is the client's helper of the same name, copied rather than shared, for the
// reason `firstWords` in this package's fixture is: a test helper is not code two
// suites may drift over, and reaching across would edit the file this whole
// exercise exists to stop depending on.
func firstDifference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	banner := "(before any banner)"
	for i := range max(len(wantLines), len(gotLines)) {
		if i < len(gotLines) && strings.HasPrefix(gotLines[i], bannerMark) {
			banner = strings.TrimSpace(gotLines[i])
		}
		switch {
		case i >= len(wantLines):
			return fmt.Sprintf("the drawn screens are longer: %d lines against the golden's %d; "+
				"line %d, under %s, is %q", len(gotLines), len(wantLines), i+1, banner, gotLines[i])
		case i >= len(gotLines):
			return fmt.Sprintf("the drawn screens are shorter: %d lines against the golden's %d; "+
				"line %d, under %s, was %q", len(gotLines), len(wantLines), i+1, banner, wantLines[i])
		case wantLines[i] != gotLines[i]:
			return fmt.Sprintf("line %d, under %s:\n  golden %q\n  drawn  %q",
				i+1, banner, wantLines[i], gotLines[i])
		}
	}
	return "the lines all match, so the difference is the trailing bytes"
}
