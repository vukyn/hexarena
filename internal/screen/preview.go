package screen

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/i18n"
)

// PreviewScreen draws the picture a character shows at the level it is being
// read at.
//
// It keeps no cursor and no level of its own: the browser that raises it hands
// both over as a Subject and re-pushes on every key that moves either, so the two
// cannot disagree about which character is in front and walking the level here
// walks it there. ⚠️ It used to read the authoring tool's browser — its level and
// its cursor — directly, which is a screen that could not live here and could not
// be drawn by a client whose browser is a different one.
//
// What it does own is the cache, because rasterising the shipped art takes tens
// of milliseconds and a full-screen client redraws on every keystroke — without
// one, holding the arrow key down would spend a second a second.
//
// ⚠️ **It is the one screen in this package that draws something other than
// text**, which is why it asks the palette whether colour is available rather
// than only rendering through it: the monochrome path is a *different drawing*,
// a ramp of weights keeping the shading, not the coloured one with the escape
// codes stripped out.
type PreviewScreen struct {
	// Subject is the character and the level being looked at, pushed in by the
	// browser. The zero value draws the same "nothing here" line an empty
	// listing does.
	Subject Subject
	// rendered is keyed by the picture and the size it was drawn at, so a
	// resized window redraws and an unchanged one does not. Written and read by
	// key only; nothing ranges over it.
	//
	// A map rather than a field because a client's model is passed by value:
	// every method here has a value receiver, so a plain field written in View
	// would be thrown away with the copy. A map header copies to the same map,
	// which is what lets a draw be remembered at all.
	rendered map[string]string
}

// NewPreviewScreen is a preview with an empty cache, which is the only state it
// has that a zero value would get wrong.
func NewPreviewScreen() PreviewScreen {
	return PreviewScreen{rendered: map[string]string{}}
}

func (p PreviewScreen) View(c Context) (string, string) {
	footer := c.Text(i18n.PreviewFooter)
	if p.Subject.Of == 0 {
		return "  " + c.Text(i18n.BrowseNothingHere) + "\n", footer
	}
	character, known := c.Lib.Characters().Get(p.Subject.ID)
	if !known {
		return "  " + c.Text(i18n.BrowseNothingHere) + "\n", footer
	}
	_, stage, err := character.Resolve(p.Subject.Level, progression.Furthest)
	if err != nil {
		return "  " + c.Style.Bad.Render(c.Lang.Error(err)) + "\n", footer
	}
	art := character.StageArt(stage)

	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(character.ID+" — "+character.Name) + "\n")
	out.WriteString("  " + c.Style.Label.Render(c.Text(i18n.PreviewTitle,
		art, p.Subject.Level, stage.Name)) + "\n\n")
	// A file that is simply not there is said the way the browser and the check
	// screen say it, rather than as a decode error carrying an absolute path: a
	// missing picture is the ordinary case while art is still being drawn, and
	// the raw error belongs to the case that is actually strange.
	stamp, present := c.Lib.ArtStamp(art)
	if !present {
		out.WriteString("  " + c.Style.Bad.Render(c.Text(i18n.ArtMissing)) + "\n")
		return out.String(), footer
	}
	picture, err := p.picture(c, art, stamp)
	if err != nil {
		out.WriteString("  " + c.Style.Bad.Render(
			c.Text(i18n.PreviewArtUnreadable, c.Lang.Error(err))) + "\n")
		return out.String(), footer
	}
	out.WriteString(picture)
	return out.String(), footer
}

// The rows the preview cannot draw a picture in, counted rather than guessed.
//
// It was guessed once, at five, and the picture came out three rows too tall at
// every window size — so the frame replaced the bottom row of the drawing with
// its "there was more than this" notice, on a screen with visible space above
// and below the sprite. That is the same arithmetic cmd/hexforge-tui's skill
// form records having to learn twice, and it is worth naming every row here:
//
//	2  the frame's own header and the blank under it
//	1  the character's id and name
//	1  the line naming the art, the level and the stage
//	1  the blank under that
//	1  the empty string strings.Split leaves after the picture's last newline
//	2  the blank the frame keeps above the footer, and the footer
//
// The last three are the ones a guess misses, because none of them is a row this
// file writes on purpose.
//
// ⚠️ Five of those eight are a **mirror** of a client's frame rather than its
// declaration — a screen package cannot see what wraps it, exactly as the
// `- 4` every Room helper here spends cannot. What holds the real frame is
// cmd/hexforge-tui's TestThePreviewFitsTheWindowItWasGiven, which walks seven
// window heights and refuses the notice at any of them.
const previewChrome = 8

// picture is the cached drawing, rasterised on the first look at this size.
//
// The stamp is what the file was when the caller looked, so redrawing the art
// outside the program invalidates the drawing rather than being ignored until a
// restart.
func (p PreviewScreen) picture(c Context, art, stamp string) (string, error) {
	cells := c.UsableWidth() - 4
	rows := c.Height - previewChrome
	if cells < 8 || rows < 4 {
		return "", fmt.Errorf("%dx%d", cells, rows)
	}
	// Two pixels to a cell vertically, one horizontally: a terminal cell is
	// about twice as tall as it is wide, so a half block is very nearly square
	// and the picture keeps its proportions without the caller correcting for
	// the font.
	key := fmt.Sprintf("%s|%s|%dx%d|%t", art, stamp, cells, rows, c.Style.Plain)
	if held, known := p.rendered[key]; known {
		return held, nil
	}
	drawn, err := c.Lib.ArtImage(art, cells, rows*2)
	if err != nil {
		return "", err
	}
	rendered := cellRows(drawn, c.Style)
	p.rendered[key] = rendered
	return rendered, nil
}

// cellRows turns pixels into terminal rows, two pixel rows to a line.
//
// This is the one place in these clients where colour carries the information
// rather than decorating it, which is why the monochrome path is a different
// drawing rather than the same one with the colour taken out: a silhouette in
// one character says the shape and nothing else, while a ramp of weights keeps
// the shading that tells a leaf from a shadow. The palette's rule still holds —
// everything a reader has to *decide* from is words elsewhere on the screen.
func cellRows(drawn *image.RGBA, style Palette) string {
	bounds := drawn.Bounds()
	plain := style.Plain
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
		r:       unpremultiply(pixel.R, pixel.A),
		g:       unpremultiply(pixel.G, pixel.A),
		b:       unpremultiply(pixel.B, pixel.A),
		painted: true,
	}
}

// unpremultiply recovers one channel of a colour from its stored value.
//
// The result saturates instead of wrapping. A premultiplied channel never
// exceeds its own alpha, so the division stays inside a byte for every pixel a
// rasteriser produces; the ceiling is there so a pixel built by hand cannot
// turn a bright colour dark by overflowing on the way back.
func unpremultiply(channel, alpha uint8) uint8 {
	if alpha == 0 {
		return 0
	}
	recovered := int(channel) * 255 / int(alpha)
	if recovered > 255 {
		return 255
	}
	// #nosec G115 -- the ceiling above is the bound, and both operands of the
	// division are bytes, so the value is between nought and 255 here.
	return uint8(recovered)
}

// hex is the pixel as a colour a style can take.
//
// lipgloss v2 returns the standard library's color.Color from Color rather than
// a named string type of its own, so this hands back that interface: the value
// is the same, and it is now the one every other colour in Go already is.
func (i inkColour) hex() color.Color {
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
func blockCell(top, bottom inkColour, style Palette) string {
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

// Ramp is the monochrome drawing's alphabet, lightest to heaviest. Inverted
// against luminance, so dark ink reads as a heavy character on a light terminal
// and a pale one on a dark terminal — the shape is the same either way.
//
// Exported because a client's own test tells a row of drawn art from a row of
// wording by its alphabet, and an alphabet written down twice is two alphabets:
// a character added here and not there turns a picture row into a wording row
// and quietly stops that test measuring anything.
const Ramp = " .:-=+*#%@"

func rampCell(top, bottom inkColour) string {
	switch {
	case !top.painted && !bottom.painted:
		return " "
	case !top.painted:
		return string(Ramp[step(bottom.luminance())])
	case !bottom.painted:
		return string(Ramp[step(top.luminance())])
	default:
		return string(Ramp[step((top.luminance()+bottom.luminance())/2)])
	}
}

func step(luminance int) int {
	// At least one step in, so a painted pixel is never drawn as a blank: a
	// white pixel that reads as nothing turns a filled shape into a hole.
	position := (255-luminance)*(len(Ramp)-1)/255 + 1
	if position > len(Ramp)-1 {
		return len(Ramp) - 1
	}
	return position
}
