// Package forge, art: turning the picture a character names into pixels.
//
// This is the only place in the repository that rasterises anything, and it is
// here for the same reason ImageExists is: internal/core may not read the
// filesystem, and a picture has to be read before it can be drawn. What comes
// back is an image and nothing more — how it reaches a screen is the front
// end's business, and a terminal and a graphical client answer that very
// differently.
package forge

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // registered so a .jpg mistakenly named .png still decodes
	_ "image/png"
	"os"
	"path"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
)

// MaxArtPixels bounds a single side of a rasterised picture.
//
// A terminal can be resized to something enormous, and the cost of rasterising
// an SVG grows with the area rather than the side: the shipped art takes tens of
// milliseconds at a hundred pixels a side and a full second is not far away. A
// preview is redrawn on a keystroke, so the bound is what keeps it a preview.
const MaxArtPixels = 240

// ArtStamp identifies the bytes behind an authored path without reading them:
// its size and when it was last written, or false when there is nothing there.
//
// It exists so that a front end caching a drawing can key the cache on the file
// rather than on the path. A cache keyed on the path alone goes stale the moment
// an author redraws the art, and a tool whose whole job is telling somebody the
// truth about a data directory must not be the last thing to notice it changed.
// One stat is microseconds against the tens of milliseconds a raster costs.
func (l *Library) ArtStamp(art string) (string, bool) {
	info, err := os.Stat(l.ImagePath(art))
	if err != nil || info.IsDir() {
		return "", false
	}
	return fmt.Sprintf("%d@%d", info.Size(), info.ModTime().UnixNano()), true
}

// ArtImage rasterises the picture at an authored path, fitted inside a box of
// the given size in pixels and centred in it.
//
// The picture's own proportions are kept, so a box wider than the art leaves
// transparent margins rather than stretching it. Transparency survives: what a
// caller does with it — a terminal leaving those cells blank, a graphical client
// compositing them — is the caller's decision, and flattening it here would take
// that decision away.
func (l *Library) ArtImage(art string, boxWidth, boxHeight int) (*image.RGBA, error) {
	if boxWidth < 1 || boxHeight < 1 {
		return nil, fmt.Errorf("a %dx%d box has nothing to draw in", boxWidth, boxHeight)
	}
	if boxWidth > MaxArtPixels {
		boxWidth = MaxArtPixels
	}
	if boxHeight > MaxArtPixels {
		boxHeight = MaxArtPixels
	}
	full := l.ImagePath(art)
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("read the art %s: %w", art, err)
	}
	if strings.EqualFold(path.Ext(art), ".svg") {
		return rasteriseSVG(raw, art, boxWidth, boxHeight)
	}
	return scaleRaster(raw, art, boxWidth, boxHeight)
}

// rasteriseSVG draws a vector at the size it is going to be looked at, which is
// the whole reason the art is vector: there is no resampling step and no blur.
func rasteriseSVG(raw []byte, art string, boxWidth, boxHeight int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(raw), oksvg.WarnErrorMode)
	if err != nil {
		return nil, fmt.Errorf("read the vector art %s: %w", art, err)
	}
	// The viewBox is the picture's own proportions. SetTarget scales to whatever
	// box it is handed, so fitting is done here rather than left to it: handing
	// it the whole box would stretch a tall picture into a wide one.
	sourceWidth, sourceHeight := icon.ViewBox.W, icon.ViewBox.H
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return nil, fmt.Errorf("the art %s declares no size to draw at", art)
	}
	width, height := fit(sourceWidth, sourceHeight, boxWidth, boxHeight)
	icon.SetTarget(0, 0, float64(width), float64(height))
	drawn := image.NewRGBA(image.Rect(0, 0, width, height))
	icon.Draw(rasterx.NewDasher(width, height,
		rasterx.NewScannerGV(width, height, drawn, drawn.Bounds())), 1)
	return centre(drawn, boxWidth, boxHeight), nil
}

// scaleRaster resamples a bitmap, which is the case the authored path allows
// even though nothing ships as one.
func scaleRaster(raw []byte, art string, boxWidth, boxHeight int) (*image.RGBA, error) {
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode the art %s: %w", art, err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 {
		return nil, fmt.Errorf("the art %s has no pixels", art)
	}
	width, height := fit(float64(bounds.Dx()), float64(bounds.Dy()), boxWidth, boxHeight)
	drawn := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(drawn, drawn.Bounds(), decoded, bounds, xdraw.Over, nil)
	return centre(drawn, boxWidth, boxHeight), nil
}

// fit is the largest size with the source's proportions that fits the box, and
// never smaller than one pixel either way.
func fit(sourceWidth, sourceHeight float64, boxWidth, boxHeight int) (width, height int) {
	scale := float64(boxWidth) / sourceWidth
	if other := float64(boxHeight) / sourceHeight; other < scale {
		scale = other
	}
	width = int(sourceWidth * scale)
	height = int(sourceHeight * scale)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return width, height
}

// centre puts a drawn picture in the middle of the box that was asked for, so a
// caller laying out cells can rely on the size it named.
func centre(drawn *image.RGBA, boxWidth, boxHeight int) *image.RGBA {
	if drawn.Bounds().Dx() == boxWidth && drawn.Bounds().Dy() == boxHeight {
		return drawn
	}
	out := image.NewRGBA(image.Rect(0, 0, boxWidth, boxHeight))
	at := image.Pt((boxWidth-drawn.Bounds().Dx())/2, (boxHeight-drawn.Bounds().Dy())/2)
	draw.Draw(out, drawn.Bounds().Add(at), drawn, drawn.Bounds().Min, draw.Src)
	return out
}
