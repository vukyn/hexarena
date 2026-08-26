package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// previewScreen draws the picture a character shows at the level the browser is
// sitting on.
//
// It keeps no cursor and no level of its own: both are read off the browser it
// was raised from, so the two cannot disagree about which character is in front
// and walking the level here walks it there. What it does own is the cache,
// because rasterising the shipped art takes tens of milliseconds and bubbletea
// redraws on every keystroke — without one, holding the arrow key down would
// spend a second a second.
type previewScreen struct {
	// rendered is keyed by the picture and the size it was drawn at, so a
	// resized window redraws and an unchanged one does not. Written and read by
	// key only; nothing ranges over it.
	//
	// A map rather than a field because the model is passed by value: every
	// method here has a value receiver, so a plain field written in View would
	// be thrown away with the copy. A map header copies to the same map, which
	// is what lets a draw be remembered at all.
	rendered map[string]string
}

func newPreviewScreen() previewScreen {
	return previewScreen{rendered: map[string]string{}}
}

func (p previewScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "p":
		m.screen = screenBrowse
		return m, nil
	case "left", "h":
		m.browse.level = clamp(m.browse.level-1, 1, progression.LevelCap)
	case "right", "l":
		m.browse.level = clamp(m.browse.level+1, 1, progression.LevelCap)
	case "home":
		m.browse.level = 1
	case "end":
		m.browse.level = progression.LevelCap
	}
	return m, nil
}

func (p previewScreen) view(m model) (string, string) {
	footer := m.text(i18n.PreviewFooter)
	rows := m.browse.rows()
	if len(rows) == 0 {
		return "  " + m.text(i18n.BrowseNothingHere) + "\n", footer
	}
	character := rows[clamp(m.browse.cursor, 0, len(rows)-1)]
	_, stage, err := character.Resolve(m.browse.level)
	if err != nil {
		return "  " + m.style.bad.Render(m.lang.Error(err)) + "\n", footer
	}
	art := character.StageArt(stage)

	var out strings.Builder
	out.WriteString(m.style.heading.Render(character.ID+" — "+character.Name) + "\n")
	out.WriteString("  " + m.style.label.Render(m.text(i18n.PreviewTitle,
		art, m.browse.level, stage.Name)) + "\n\n")
	// A file that is simply not there is said the way the browser and the check
	// screen say it, rather than as a decode error carrying an absolute path: a
	// missing picture is the ordinary case while art is still being drawn, and
	// the raw error belongs to the case that is actually strange.
	stamp, present := m.lib.ArtStamp(art)
	if !present {
		out.WriteString("  " + m.style.bad.Render(m.text(i18n.ArtMissing)) + "\n")
		return out.String(), footer
	}
	picture, err := p.picture(m, art, stamp)
	if err != nil {
		out.WriteString("  " + m.style.bad.Render(
			m.text(i18n.PreviewArtUnreadable, m.lang.Error(err))) + "\n")
		return out.String(), footer
	}
	out.WriteString(picture)
	return out.String(), footer
}

// The rows the preview spends on its own heading, the blank under it and the
// footer, counted here rather than guessed for the reason the skill form's
// formRoom records.
const previewChrome = 5

// picture is the cached drawing, rasterised on the first look at this size.
//
// The stamp is what the file was when the caller looked, so redrawing the art
// outside the program invalidates the drawing rather than being ignored until a
// restart.
func (p previewScreen) picture(m model, art, stamp string) (string, error) {
	cells := m.usableWidth() - 4
	rows := m.height - previewChrome
	if cells < 8 || rows < 4 {
		return "", fmt.Errorf("%dx%d", cells, rows)
	}
	// Two pixels to a cell vertically, one horizontally: a terminal cell is
	// about twice as tall as it is wide, so a half block is very nearly square
	// and the picture keeps its proportions without the caller correcting for
	// the font.
	key := fmt.Sprintf("%s|%s|%dx%d|%t", art, stamp, cells, rows, plainTerminal())
	if held, known := p.rendered[key]; known {
		return held, nil
	}
	drawn, err := m.lib.ArtImage(art, cells, rows*2)
	if err != nil {
		return "", err
	}
	rendered := cellRows(drawn, m.style)
	p.rendered[key] = rendered
	return rendered, nil
}

// cellRows turns pixels into terminal rows, two pixel rows to a line.
//
// This is the one place in the program where colour carries the information
// rather than decorating it, which is why the monochrome path is a different
// drawing rather than the same one with the colour taken out: a silhouette in
// one character says the shape and nothing else, while a ramp of weights keeps
// the shading that tells a leaf from a shadow. The palette's rule still holds —
// everything a reader has to *decide* from is words elsewhere on the screen.
func cellRows(drawn *image.RGBA, style palette) string {
	bounds := drawn.Bounds()
	plain := plainTerminal()
	var out strings.Builder
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		out.WriteString("  ")
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			top := ink(drawn.RGBAAt(x, y))
			bottom := ink(drawn.RGBAAt(x, y+1))
			if plain {
				out.WriteString(rampCell(top, bottom))
				continue
			}
			out.WriteString(blockCell(top, bottom, style))
		}
		out.WriteString("\n")
	}
	return out.String()
}

// inkColour is one pixel decided: its colour, and whether there is any.
type inkColour struct {
	r, g, b uint8
	painted bool
}

// alphaFloor is how much coverage a pixel needs before it is drawn at all.
//
// Low on purpose. The art is traced vector with hard edges, and the pixels that
// land between two colours are the outlines — dropping them at a half-coverage
// threshold thins every line in the picture, which at this size is most of what
// there is to see.
const alphaFloor = 40

// ink un-premultiplies a pixel and says whether it is worth drawing.
//
// Go's RGBA is alpha-premultiplied, so a half-transparent red is stored dark.
// Drawing it as stored would composite correctly over black and be wrong over
// every other terminal background, so the colour is recovered and the coverage
// becomes a yes or no instead.
func ink(pixel color.RGBA) inkColour {
	if pixel.A < alphaFloor {
		return inkColour{}
	}
	return inkColour{
		r:       uint8(int(pixel.R) * 255 / int(pixel.A)),
		g:       uint8(int(pixel.G) * 255 / int(pixel.A)),
		b:       uint8(int(pixel.B) * 255 / int(pixel.A)),
		painted: true,
	}
}

func (i inkColour) hex() lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", i.r, i.g, i.b))
}

// luminance is the perceptual weight of a pixel, in integer thousandths, which
// is the same shape every other ratio in this repository is written in.
func (i inkColour) luminance() int {
	return (299*int(i.r) + 587*int(i.g) + 114*int(i.b)) / 1000
}

// blockCell is two pixels in one cell: the upper half block painted in the top
// pixel's colour over the bottom pixel's.
//
// A cell with nothing in either half is a plain space rather than a styled one,
// so a transparent margin shows the terminal's own background instead of a
// rectangle of whatever this program guessed it to be.
func blockCell(top, bottom inkColour, style palette) string {
	switch {
	case !top.painted && !bottom.painted:
		return " "
	case top.painted && !bottom.painted:
		return lipgloss.NewStyle().Foreground(top.hex()).Render("▀")
	case !top.painted && bottom.painted:
		return lipgloss.NewStyle().Foreground(bottom.hex()).Render("▄")
	default:
		return lipgloss.NewStyle().
			Foreground(top.hex()).Background(bottom.hex()).Render("▀")
	}
}

// ramp is the monochrome drawing, lightest to heaviest. Inverted against
// luminance, so dark ink reads as a heavy character on a light terminal and a
// pale one on a dark terminal — the shape is the same either way.
const ramp = " .:-=+*#%@"

func rampCell(top, bottom inkColour) string {
	switch {
	case !top.painted && !bottom.painted:
		return " "
	case !top.painted:
		return string(ramp[step(bottom.luminance())])
	case !bottom.painted:
		return string(ramp[step(top.luminance())])
	default:
		return string(ramp[step((top.luminance()+bottom.luminance())/2)])
	}
}

func step(luminance int) int {
	// At least one step in, so a painted pixel is never drawn as a blank: a
	// white pixel that reads as nothing turns a filled shape into a hole.
	position := (255-luminance)*(len(ramp)-1)/255 + 1
	if position > len(ramp)-1 {
		return len(ramp) - 1
	}
	return position
}
