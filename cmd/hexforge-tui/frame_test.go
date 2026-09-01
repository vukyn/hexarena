package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/tui"
)

// What frame does to a line too long for the window, and what it says about it.
//
// The subject of every test here is one row of one real screen — frame's own
// header, which names the library directory and is therefore the one line on
// every screen whose length is a fact about somebody's filesystem rather than
// about this program. A biography, an art path and a refusal sentence are the
// same shape of risk; the header is the one that can be made over-long by
// arrangement rather than by hoping the fixture data stays long enough, which is
// what keeps these from going vacuous the next time the cast is edited.

// aLibraryFiledDeep is the scratch data copied under a path long enough that the
// header naming it cannot fit the window.
//
// Built by nesting rather than by writing a long name down, for the reason
// aPathPartLongerThanTheFloor repeats a stem: what has to be true is a
// *relationship* to the floor, and a literal that happened to clear a floor of
// 80 is a literal that quietly stopped clearing one of 120.
func aLibraryFiledDeep(t *testing.T, past int) string {
	t.Helper()
	source := scratchData(t)
	target := t.TempDir()
	for lipgloss.Width(target) < past {
		target = filepath.Join(target, "a-folder-nested-deeper-than-anybody-would")
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create %s: %v", target, err)
	}
	copyTree(t, source, target)
	return target
}

// headerOf is frame's own first row, as the screen draws it.
func headerOf(m model) string {
	return strings.Split(m.screenContent(), "\n")[0]
}

// inColour is the model with a palette that writes escape codes.
//
// start sets NO_COLOR for every test in this package, which is what lets an
// assertion look for a word rather than for a word wrapped in escapes — so a
// test about what a cut does to a *styled* line has to undo that deliberately,
// and rebuild the palette afterwards, since newModel already asked.
func inColour(t *testing.T, m model) model {
	t.Helper()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	m.style = newPalette()
	if !strings.Contains(m.style.Title.Render("x"), "\x1b") {
		t.Fatal("the palette still writes plain text, so nothing below measures a styled line")
	}
	return m
}

// sgr matches one complete select-graphic-rendition sequence.
var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// aPictureRow reports whether a row of the preview is drawn art rather than
// wording.
//
// Told apart by its alphabet rather than by its position: the picture is the
// only thing on that screen made of nothing but the ramp and the half blocks,
// while every other row carries letters. Counting from the bottom instead would
// be a second copy of previewChrome's arithmetic, which is exactly the sort of
// duplicate that comment records having got wrong once already.
func aPictureRow(row string) bool {
	plain := ansi.Strip(row)
	if strings.TrimSpace(plain) == "" {
		return false
	}
	for _, letter := range plain {
		if !strings.ContainsRune(draw.Ramp, letter) && !strings.ContainsRune("▀▄", letter) {
			return false
		}
	}
	return true
}

// TestACutLineSaysItWasCut is the defect itself: frame used to take the tail off
// a line and leave nothing on the screen saying it had.
//
// Asked of the header rather than of a sentence, because the header can be made
// over-long by where the data directory is rather than by what the fixture
// happens to contain — and it is checked to *be* over-long before anything is
// asserted about the mark, since a header that fits would make every claim below
// true of a line nothing cut.
func TestACutLineSaysItWasCut(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := startIn(t, lang, aLibraryFiledDeep(t, minWidth))
		m.height = minHeight
		whole := m.style.Title.Render(programName) + m.style.Dim.Render("  "+m.lib.Dir())
		if lipgloss.Width(whole) <= minWidth {
			t.Fatalf("the header is %d cells against a floor of %d, so frame cuts nothing "+
				"and this test measures nothing", lipgloss.Width(whole), minWidth)
		}
		m.width = lipgloss.Width(whole) - 1
		header := headerOf(m)
		if !strings.HasSuffix(ansi.Strip(header), ellipsis) {
			t.Errorf("in %s the cut header does not say it was cut:\n%q", lang, ansi.Strip(header))
		}
		if got := lipgloss.Width(header); got != m.width {
			t.Errorf("in %s the marked header is %d cells against the %d the window has",
				lang, got, m.width)
		}
	}
}

// TestALineThatExactlyFillsTheWindowIsNotMarked is the other half and the whole
// off-by-one risk of marking at all.
//
// A mark on a line that fitted claims a tail that was never there, and pays a
// cell of real content to claim it. The two widths are one apart on purpose:
// the same header, in the window it exactly fills and in the window one column
// narrower, so the boundary is crossed rather than approached.
func TestALineThatExactlyFillsTheWindowIsNotMarked(t *testing.T) {
	for _, lang := range i18n.Langs() {
		m, _, _ := startIn(t, lang, aLibraryFiledDeep(t, minWidth))
		m.height = minHeight
		whole := m.style.Title.Render(programName) + m.style.Dim.Render("  "+m.lib.Dir())
		exact := lipgloss.Width(whole)
		if exact <= minWidth {
			t.Fatalf("the header is %d cells against a floor of %d, so the window that "+
				"exactly fits it is one the tool refuses to draw in", exact, minWidth)
		}

		m.width = exact
		if fitted := headerOf(m); fitted != whole {
			t.Errorf("in %s a header of %d cells in a window of %d came back changed:\n"+
				"  want %q\n  got  %q", lang, exact, exact, ansi.Strip(whole), ansi.Strip(fitted))
		}

		m.width = exact - 1
		if cut := headerOf(m); !strings.HasSuffix(ansi.Strip(cut), ellipsis) {
			t.Errorf("in %s one column narrower is not marked, so the boundary this test "+
				"exists for was never crossed:\n%q", lang, ansi.Strip(cut))
		}
	}
}

// TestACutStyledLineKeepsItsColourToItself is what the old clip could not do and
// MaxWidth could.
//
// Two assertions, and they catch different failures. The **cells** must be the
// ones a plain terminal would have drawn — that is what a half-cut escape breaks,
// since the surviving fragment stops being an escape and starts being visible
// letters. And the row must **close every style it opens**, which stripping
// cannot see at all: peeling the trailing reset off leaves the right cells and
// the right width, and bleeds the colour down every row under it.
func TestACutStyledLineKeepsItsColourToItself(t *testing.T) {
	dir := aLibraryFiledDeep(t, minWidth)
	for _, lang := range i18n.Langs() {
		plainModel, _, _ := startIn(t, lang, dir)
		plainModel.height = minHeight
		plainModel.width = lipgloss.Width(
			programName+"  "+plainModel.lib.Dir()) - 1
		if plainModel.width < minWidth {
			t.Fatalf("the header fits the floor, so nothing here is cut")
		}
		plain := headerOf(plainModel)

		colourModel := inColour(t, plainModel)
		colourModel.width = plainModel.width
		coloured := headerOf(colourModel)
		if !strings.Contains(coloured, "\x1b") {
			t.Fatalf("in %s the header carries no escape codes, so this measures the "+
				"plain path twice", lang)
		}
		if got := ansi.Strip(coloured); got != plain {
			t.Errorf("in %s the cut styled header draws different cells from the plain one:\n"+
				"  plain   %q\n  styled  %q", lang, plain, got)
		}
		if got, want := lipgloss.Width(coloured), colourModel.width; got != want {
			t.Errorf("in %s the cut styled header is %d cells against the %d the window has",
				lang, got, want)
		}
		if codes := sgr.FindAllString(coloured, -1); len(codes) == 0 {
			t.Errorf("in %s no complete escape sequence survived the cut, so one was cut "+
				"in half:\n%q", lang, coloured)
		} else if last := codes[len(codes)-1]; last != "\x1b[m" && last != "\x1b[0m" {
			t.Errorf("in %s the cut styled header ends on %q rather than a reset, so its "+
				"colour bleeds into the row below:\n%q", lang, last, coloured)
		}
	}
}

// TestAMarkedLineIsAsWideAsAnUnmarkedCutWouldHaveBeen is what keeps frame's row
// arithmetic out of this change.
//
// frame counts rows and pads to the window's height on the assumption that no
// line wraps, so a mark that cost a cell **past** the window would move the
// footer off the bottom — the exact failure the clipping exists to prevent. The
// comparison is against `MaxWidth`, which is what frame used before, so what is
// asserted is that the widths did not move rather than that they satisfy some
// number written down here.
func TestAMarkedLineIsAsWideAsAnUnmarkedCutWouldHaveBeen(t *testing.T) {
	styled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")).Render("hexforge") +
		lipgloss.NewStyle().Faint(true).Render("  /a/rather/long/path/to/a/data/directory")
	for _, subject := range []string{ansi.Strip(styled), styled} {
		whole := lipgloss.Width(subject)
		for room := 1; room <= whole+2; room++ {
			marked := clip(subject, room)
			unmarked := lipgloss.NewStyle().MaxWidth(room).Render(subject)
			if got, want := lipgloss.Width(marked), lipgloss.Width(unmarked); got != want {
				t.Errorf("at %d cells of a %d-cell line, the marked cut is %d wide and the "+
					"unmarked one %d", room, whole, got, want)
			}
		}
	}
}

// TestNoDrawingIsEverWideEnoughToBeMarked is the decision *not* to treat a
// picture specially, written down so the next reader does not "fix" it.
//
// Marking every line frame touches is only safe while every line frame touches
// is text: an ellipsis on the end of a sentence says a tail was taken off, and an
// ellipsis on the end of ten rows of hex art says something nobody can act on. So
// rather than teach frame which lines are drawings — it is handed a joined string
// and cannot know — the claim is that no drawing can reach the cut at all, and it
// is measured on both of them.
//
// The board is asked at the largest squad there is, since that is the most it can
// ever hold, and the art is asked at the floor, since that is the narrowest
// window either is ever drawn in.
func TestNoDrawingIsEverWideEnoughToBeMarked(t *testing.T) {
	base, _, _ := start(t, i18n.Vi)
	base.width, base.height = minWidth, 60
	fought := atABattleOf(t, base, hex.MaxTeamSize)
	if fought.play.fight == nil {
		t.Fatal("the fixture reached the battle screen without a battle, so no board is drawn")
	}
	for name, drawn := range map[string]string{
		"board":  tui.Board(fought.play.fight, fought.play.tags),
		"roster": tui.Roster(fought.play.fight, fought.play.tags),
	} {
		rows := strings.Split(strings.TrimRight(drawn, "\n"), "\n")
		if len(rows) < 2 {
			t.Fatalf("tui.%s drew %d rows, so the fixture is not drawing one", name, len(rows))
		}
		for _, row := range rows {
			if width := lipgloss.Width(row); width >= minWidth {
				t.Errorf("a %s row is %d cells against a floor of %d, so a drawing can now "+
					"reach frame's cut and carry a mark:\n%s", name, width, minWidth, row)
			}
		}
	}

	// The preview's art is the other drawing, and the only one whose width is a
	// function of the window rather than fixed. It is asked through the screen
	// rather than by repeating `usableWidth() - 4`, which is the arithmetic under
	// test.
	preview, _, _ := start(t, i18n.Vi)
	preview.width, preview.height = minWidth, 40
	preview = preview.hand(preview.browse.Subject())
	preview.screen = screenPreview
	body, _ := preview.preview.View(preview.ctx())
	painted, widest := 0, 0
	for _, row := range strings.Split(body, "\n") {
		if width := lipgloss.Width(row); width >= preview.width {
			t.Errorf("a preview row is %d cells against a window of %d, so the art can reach "+
				"frame's cut and carry a mark:\n%q", width, preview.width, ansi.Strip(row))
		}
		if !aPictureRow(row) {
			continue
		}
		painted++
		if width := lipgloss.Width(row); width > widest {
			widest = width
		}
	}
	// ⚠️ The count and the width are both asserted, and the width is the half a
	// count cannot give: a preview that had quietly stopped filling the window
	// would draw its rows, pass a count, and prove nothing about a drawing being
	// able to reach the cut.
	if painted < 4 {
		t.Fatalf("the preview drew %d picture rows, so this measured its heading and "+
			"nothing else:\n%s", painted, ansi.Strip(body))
	}
	if want := preview.usableWidth() - 2; widest != want {
		t.Errorf("the widest preview row is %d cells rather than the %d the art is sized to, "+
			"so this is not measuring a full-width drawing", widest, want)
	}
}
