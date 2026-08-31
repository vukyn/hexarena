package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// chartScreen is the affinity chart drawn as the rings it was declared in.
//
// The elements listing answers "what does fire lose to" one element at a time,
// which is the question somebody has with a skill in front of them. This answers
// the other one: what is the *shape*. Eleven rows of two strengths and two
// weaknesses is twenty-two facts to hold; three rings and a pair is four, and
// the rings are what the chart was written as — see the package doc on
// internal/core/element, which draws them the same way for the same reason.
//
// A screen of its own rather than a block under the listing. Both halves want
// the height: the listing is eleven rows plus a description, the rings are five
// blocks plus their rates, and a window at the floor cannot hold the two. It is
// raised from the listing, the way the description screen is raised from the
// skills listing, because that is where a reader is when the question occurs to
// them.
//
// No cursor. Nothing here is selected — the whole point is that it is read at
// once, and a cursor would invite a keystroke that has nothing to do.
type chartScreen struct{}

func (s chartScreen) update(_ draw.Context, message tea.KeyPressMsg) (chartScreen, draw.Action) {
	switch message.String() {
	case "q":
		return s, draw.Action{Kind: draw.Quit}
	case "esc":
		// Back to whoever raised this, rather than to the menu: a reader who
		// pressed one key to arrive expects one key to undo it.
		//
		// It named the elements listing until this step, on the ground that
		// nothing else raises this screen — still true (grep screenChart: the
		// client declares it, the elements listing raises it, the dispatcher
		// draws it, and the width fixture enters it without ever pressing a key
		// in it). So Back lands in exactly the same place, and it does so
		// without a screen here knowing that listing exists.
		return s, draw.Action{Kind: draw.Back}
	}
	return s, draw.Action{}
}

// The pieces a ring is drawn out of.
//
// ASCII rather than the box-drawing characters and arrows this would obviously
// like to use. → and ⇄ and ┌ are East-Asian *ambiguous* width: a terminal that
// draws them two cells wide while the program measures them as one leaves every
// column after them one cell out — and a diagram is a picture made of columns,
// so it does not survive being one cell out the way a sentence does. This
// repository has already been bitten by exactly that with the modifier glyphs on
// a footer.
const (
	beatsMark  = " --> "
	backMark   = " <-- "
	mutualMark = " <--> "
	// ringOpen is the corner the ring starts at, and ringBack the one the return
	// line starts at, so a reader can see where the loop is entered and where it
	// comes back.
	ringOpen = ",--> "
	ringBack = "'--- "
	// chainMark is the fallback notation, and internal/core/element's package
	// doc's own: it is what a ring falls back to when no window can hold the
	// picture.
	chainMark = " > "
)

func (s chartScreen) view(m model) (string, string) {
	footer := m.text(i18n.ChartFooter)
	chart := m.lib.Chart()
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.ChartHeading)) + "  " +
		m.style.Dim.Render(m.text(i18n.ChartSubtitle)) + "\n\n")

	blocks := chartBlocks(m, chart)
	if len(blocks) == 0 {
		out.WriteString("  " + m.text(i18n.ChartEmpty) + "\n")
		return out.String(), footer
	}
	// Every element the chart names, so the colouring can pick them out of a line
	// of the picture. Word by word inside marked, which is what keeps a style
	// inside one cell run rather than spanning the rules drawn beside it.
	names := make([]string, 0, element.Count)
	for _, member := range element.All() {
		names = append(names, member.String())
	}
	for _, block := range blocks {
		out.WriteString("  " + m.style.Label.Render(block.label) + "\n")
		for _, line := range block.lines {
			out.WriteString("    " + marked(line, names, func(word string) string {
				member, err := element.Parse(word)
				if err != nil {
					return word
				}
				return m.style.Element(member).Render(word)
			}) + "\n")
		}
	}

	// The three figures the whole picture is worth, once at the foot. They are on
	// every element's own description too, but a reader looking at the shape is
	// asking what an edge is *for*, and an edge with no price on it is a line
	// between two words.
	rates := chart.Multipliers()
	out.WriteString("\n  " + m.style.Dim.Render(m.text(i18n.ChartRates,
		i18n.Share(rates.Advantage), i18n.Share(rates.Neutral),
		i18n.Share(rates.Disadvantage))))
	return out.String(), footer
}

// chartBlock is one grouping: its name, and the lines that draw it.
type chartBlock struct {
	label string
	lines []string
}

// chartBlocks is the whole chart, in the order it is read: the rings as
// declared, then the pairs, then whatever is in neither.
//
// The rings keep their authored names — "organic", "cross" — because those are
// what the author grouped by and the id of a ring is the only handle a reader
// has for one. The last two blocks are named by the program, since being a pair
// and being inert are facts about the shape rather than things somebody wrote.
//
// ⚠️ Every block is **generated from the chart**, and that is the whole reason
// the picture may be a picture at all. A hand-drawn diagram is right until the
// day an element is added to elements.json, and then it is a figure that lies
// with nothing to catch it — the same trade the derived descriptions make. Add
// an element and the ring redraws itself; what an author has to check is only
// that it still *fits*, which TestEveryRingClosesAtTheFloor does for them.
func chartBlocks(m model, chart *element.Chart) []chartBlock {
	blocks := make([]chartBlock, 0, 6)
	for _, cycle := range chart.Cycles() {
		if len(cycle.Chain) == 0 {
			continue
		}
		blocks = append(blocks, chartBlock{cycle.Name, ringLines(elementIDs(cycle.Chain), chartRoom())})
	}
	for _, pair := range chart.MutualPairs() {
		// Not a ring of two. Both members beat each other, so neither is safe
		// from the other, and a two-link loop would draw one of the arrows as
		// though it were the way round the pair was declared.
		blocks = append(blocks, chartBlock{
			m.text(i18n.ChartMutual),
			[]string{pair[0].String() + mutualMark + pair[1].String()},
		})
	}
	if inert := chart.Inert(); len(inert) > 0 {
		blocks = append(blocks, chartBlock{
			m.text(i18n.ChartInert),
			[]string{strings.Join(elementIDs(inert), " ")},
		})
	}
	return blocks
}

// elementIDs is a run of elements as the ids they are written with.
func elementIDs(members []element.Element) []string {
	out := make([]string, 0, len(members))
	for _, member := range members {
		out = append(out, member.String())
	}
	return out
}

// chartRoom is how wide a ring may be drawn.
//
// The floor rather than the window in hand, and deliberately: a picture is
// remembered by its shape, and a diagram that is one row on a wide terminal and
// two rows on a narrow one is two different pictures of one chart. Every reader
// gets the same drawing.
func chartRoom() int {
	const indent = 4
	return minWidth - 1 - indent
}

// ringLines draws one ring as a closed loop.
//
// A loop rather than a chain, because a chain is the one thing a ring is not: it
// reads as an order of precedence, and the reader has to be told separately that
// the last member beats the first. Here the line goes back.
//
//	,--> water --> fire --> grass --> ground --,
//	`------------------------------------------'
//
// A ring too wide for one row turns back on itself instead, so the return leg
// carries the rest of the members rather than being drawn empty:
//
//	,--> water --> metal --> grass --> wind --> fire --,
//	|                                                  |
//	'--- electric <-- ground <-- ice <-----------------'
//
// Read that second one clockwise: along the top to fire, down the right edge,
// then **right to left** along the bottom — ice, ground, electric — up the left
// edge, and back into water. The arrows on the bottom row point the way it is
// read, which is why they are reversed rather than the names being.
//
// Wider than two rows is refused rather than drawn. A third leg would have to
// come back left-to-right again and the picture stops being a loop a reader can
// follow; the fallback is the plain chain, which is honest about being a list.
// Nothing in the shipped chart reaches it — the eight-member ring fits two rows
// at the floor with room over — and a ring that did would be a ring of about
// sixteen, which is a chart with a bigger problem than its drawing.
func ringLines(names []string, width int) []string {
	if len(names) == 0 {
		return nil
	}
	if one, ok := ringOneRow(names, width); ok {
		return one
	}
	// The narrowest split rather than the first that fits, and that is a
	// legibility decision rather than a tidiness one: filling the top row leaves
	// one member alone on the return leg beside a rule half a screen long, which
	// draws as a chain with a stray box round it. Balancing the two legs draws as
	// a loop. The widest leg decides the picture, so the narrowest picture is the
	// most balanced one, and it is the same search either way.
	var best []string
	for head := 1; head < len(names); head++ {
		two, ok := ringTwoRows(names[:head], names[head:], width)
		if !ok {
			continue
		}
		if best == nil || len(two[0]) < len(best[0]) {
			best = two
		}
	}
	if best != nil {
		return best
	}
	return []string{strings.Join(names, chainMark) + chainMark + "(" + names[0] + ")"}
}

func ringOneRow(names []string, width int) ([]string, bool) {
	// " --," is the closing corner: a rule, then the corner the return line
	// hangs from.
	body := ringOpen + strings.Join(names, beatsMark)
	full := len(body) + len(" --,")
	if full > width {
		return nil, false
	}
	return []string{
		body + " " + strings.Repeat("-", full-len(body)-2) + ",",
		"`" + strings.Repeat("-", full-2) + "'",
	}, true
}

func ringTwoRows(head, tail []string, width int) ([]string, bool) {
	top := ringOpen + strings.Join(head, beatsMark)
	// The return leg is printed in reverse so that its arrows point the way the
	// line is read, right to left.
	bottom := ringBack + strings.Join(reversed(tail), backMark)
	// Both legs are ruled out to the same corner column, or the two ends of the
	// loop would not meet.
	full := max(len(top)+len(" --,"), len(bottom)+len(" <-'"))
	if full > width {
		return nil, false
	}
	return []string{
		top + " " + strings.Repeat("-", full-len(top)-2) + ",",
		"|" + strings.Repeat(" ", full-2) + "|",
		bottom + " <" + strings.Repeat("-", full-len(bottom)-3) + "'",
	}, true
}

func reversed(names []string) []string {
	out := make([]string, 0, len(names))
	for index := len(names) - 1; index >= 0; index-- {
		out = append(out, names[index])
	}
	return out
}
