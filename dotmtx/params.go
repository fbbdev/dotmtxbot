package dotmtx

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/zachomedia/go-bdf"

	"petbots.fbbdev.it/dotmtxbot/log"
)

const (
	DotInnerSize = 6
	DotPadding   = 1
	DotSize      = DotInnerSize + 2*DotPadding
)

// Resource limits
const (
	MaxChars = 100
	MaxWidth = 10922
)

var Font font.Face
var CharWidthInDots int
var CharWidthInPixels int

//go:embed cherry-11-r.bdf
var fontData []byte

func init() {
	bdfFont, err := bdf.Parse(fontData)
	if err != nil {
		log.ErrorLogger.Print("bdf:", err)
		log.FatalLogger.Fatal("could not load dot matrix font")
	}

	Font = bdfFont.NewFace()
	advance, ok := Font.GlyphAdvance(bdfFont.DefaultChar)
	if !ok {
		advance = fixed.I(6)
	}

	CharWidthInDots = advance.Ceil()
	CharWidthInPixels = CharWidthInDots * DotSize
}
