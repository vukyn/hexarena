package screen

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// roomyWindow is a window with room for the wide roster.
//
// It is the roomy size the goldens are recorded at, so a screen this test finds
// the wide table on is a screen the golden records the wide table for.
const roomyWindow = 160

// TestAResizePicksTheOtherRosterWithoutANewTurn is the whole reason the width
// decision sits in the draw and not in the reading.
//
// ⚠️ **A reading is taken when a TURN arrives, not on every frame.** For a live
// screen that is Attach, which is where the mirror's lock is held; on a
// ninety-second allowance the gap between two of them is most of a minute. So a
// width consulted while the reading is taken is the width the window had at the
// last turn, and a player who resized in between would go on being shown the
// table for a window they no longer have — for as long as the other player
// thinks.
//
// This drives exactly that: one Attach, then two draws at two widths with
// nothing attached in between. A test built on a fresh Attach per size cannot see
// the bug at all — it would pass with the decision taken in readBattle, which is
// the arrangement this is here to refuse.
func TestAResizePicksTheOtherRosterWithoutANewTurn(t *testing.T) {
	c, _ := start(t, i18n.En)
	if c.Width != MinWidth {
		t.Fatalf("the fixture starts at %d rather than the floor of %d", c.Width, MinWidth)
	}
	// The premise this rests on, held rather than assumed: the two sizes really
	// are on opposite sides of the threshold. If they were not, the whole test
	// below would compare a table against itself and pass.
	if tui.RosterIsWide(MinWidth) || !tui.RosterIsWide(roomyWindow) {
		t.Fatalf("the floor of %d and the roomy window of %d are not on opposite sides of "+
			"the wide table's threshold", MinWidth, roomyWindow)
	}

	fight, prompt := aBattleNobodyHereDrives(t, c, 3)
	live := NewPlayScreen().Attach(c, PlayLive{Fight: fight, Asking: prompt})

	// What the battle reads as at the moment of that one attach. Both tables,
	// taken now, because the battle is about to move underneath the screen.
	atAttachNarrow := tui.Roster(c.Lang, fight, live.Tags)
	atAttachWide := tui.RosterWide(c.Lang, fight, live.Tags, c.Style.ElementInk())
	if atAttachNarrow == atAttachWide {
		t.Fatal("the two tables draw the same thing on this fixture, so nothing here could " +
			"tell which one a draw picked")
	}

	// The room takes some turns and nothing re-attaches. That is the second half
	// of the claim: the layout the resize picks has to come out of the reading
	// rather than out of a battle this goroutine may not touch.
	for range 6 {
		prompt = steppedByTheRoom(t, fight, prompt)
		if prompt == nil {
			break
		}
	}
	if now := tui.Roster(c.Lang, fight, live.Tags); now == atAttachNarrow {
		t.Fatal("six turns moved nothing on the roster, so a drawing off the reading and a " +
			"drawing off the battle are indistinguishable here")
	}

	narrow := c
	narrow.Width = MinWidth
	roomy := c
	roomy.Width = roomyWindow

	if got := strings.Join(live.drawings(narrow).roster, "\n"); got != atAttachNarrow {
		t.Errorf("at the floor the screen draws\n%s\nand the reading it took holds\n%s",
			got, atAttachNarrow)
	}
	if got := strings.Join(live.drawings(roomy).roster, "\n"); got != atAttachWide {
		t.Errorf("at %d cells the screen draws\n%s\nand the reading it took holds\n%s",
			roomyWindow, got, atAttachWide)
	}

	// And back again, because a screen that widened once and then stuck would
	// pass everything above.
	if got := strings.Join(live.drawings(narrow).roster, "\n"); got != atAttachNarrow {
		t.Errorf("narrowed again the screen keeps the wide table:\n%s", got)
	}
}

// TestALocalBattleTakesTheWideRosterOnAWideWindowToo is the other half of the
// screen's two paths.
//
// A hot-seat screen owns its battle and reads it lazily, so it never stores a
// reading and the width question reaches it by a different route. The rule has to
// be the same on both, or the same window would draw two different tables
// depending on whether the game is a match or not.
func TestALocalBattleTakesTheWideRosterOnAWideWindowToo(t *testing.T) {
	c, _ := start(t, i18n.En)
	local := atABattleOf(t, c, 3)

	roomy := c
	roomy.Width = roomyWindow
	drawn := strings.Join(local.drawings(roomy).roster, "\n")
	if want := tui.RosterWide(c.Lang, local.Fight, local.Tags, c.Style.ElementInk()); drawn != want {
		t.Errorf("a hot-seat screen at %d cells draws\n%s\nwant\n%s", roomyWindow, drawn, want)
	}
	if narrow := strings.Join(local.drawings(c).roster, "\n"); narrow == drawn {
		t.Errorf("the floor and %d cells draw the same table, so this measured nothing:\n%s",
			roomyWindow, narrow)
	}
}

// TestTheWideRosterStillFitsTheWindowItIsOfferedIn is the bound on the far side
// of the threshold: the table is offered because there is room for it, so every
// row it draws has to be inside that room.
//
// ⚠️ **Measured at the threshold itself and not at the roomy golden window**,
// because the threshold is the only width where the claim can fail — every wider
// window has slack, so a row overflowing by a cell would be invisible at 160 and
// caught here.
func TestTheWideRosterStillFitsTheWindowItIsOfferedIn(t *testing.T) {
	for _, lang := range i18n.Langs() {
		c, _ := start(t, lang)
		local := atABattleOf(t, c, 3)
		// The narrowest window that takes the wide table, found by walking up to
		// it rather than naming it, so this cannot drift from tui's own rule.
		threshold := MinWidth
		for !tui.RosterIsWide(threshold) {
			threshold++
			if threshold > 1000 {
				t.Fatal("no window under a thousand cells takes the wide table")
			}
		}
		sized := c
		sized.Width = threshold
		for _, row := range local.drawings(sized).roster {
			if width := lipgloss.Width(row); width >= threshold {
				t.Errorf("%v: a wide roster row is %d cells in the %d-cell window that "+
					"offered it, and every line leaves one empty:\n%s",
					lang, width, threshold, row)
			}
		}
	}
}
