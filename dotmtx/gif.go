package dotmtx

import (
	"bytes"
	_ "embed"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"petbots.fbbdev.it/dotmtxbot/log"
)

//go:embed toobig.gif
var tooBigGifData []byte
var tooBigGif *gif.GIF

func init() {
	img, err := gif.DecodeAll(bytes.NewReader(tooBigGifData))
	if err != nil {
		log.ErrorLogger.Print("gif: ", err)
		log.FatalLogger.Fatal("could not load error message gif")
	}

	tooBigGif = img
}

var errWidthOverflow = errors.New("maximum width exceeded")
var errUnexpectedSubImageFormat = errors.New("unexpected sub-image format")

var palette = [...]color.Color{
	color.Black,
	color.Gray{50},
	color.RGBA{255, 170, 0, 255},
}

func max(x, y int) int {
	if x >= y {
		return x
	} else {
		return y
	}
}

func drawDotMatrix(text string) (img *image.Paletted, err error) {
	advance := font.MeasureString(Font, text)
	if advance.Ceil() > MaxWidth/DotSize {
		return nil, errWidthOverflow
	}

	img = image.NewPaletted(
		image.Rect(0, 0, advance.Ceil(), Font.Metrics().Height.Ceil()),
		palette[:],
	)

	draw.Draw(img, img.Bounds(), image.NewUniform(palette[1]), image.Point{}, draw.Src)

	drawer := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(palette[2]),
		Face: Font,
		Dot:  fixed.Point26_6{X: fixed.I(0), Y: Font.Metrics().Ascent},
	}

	drawer.DrawString(text)

	return
}

func MakeGif(speed float64, width float64, blank float64, text string) (*gif.GIF, error) {
	width = math.Min(width, 1+blank)
	if speed == 0 {
		blank = math.Max(0, width-1)
	}

	dotMatrix, err := drawDotMatrix(text)
	if err != nil {
		if err == errWidthOverflow {
			return tooBigGif, nil
		}

		return nil, err
	}

	// log.InfoLogger.Print("text rendering done")

	dotMatrixWidth := dotMatrix.Rect.Dx()
	dotMatrixHeight := dotMatrix.Rect.Dy()

	windowColumns := math.Ceil(width * float64(dotMatrixWidth))
	windowWidth := windowColumns*DotSize + 2*DotPadding

	backingImageColumns := math.Ceil((1 + blank) * float64(dotMatrixWidth))
	backingImageWidth := (backingImageColumns+windowColumns)*DotSize + 2*DotPadding
	backingImageHeight := dotMatrixHeight*DotSize + 2*DotPadding

	// handle overflows; no need to check windowWidth as width <= 1+blank
	if math.IsNaN(backingImageWidth) || math.IsInf(backingImageWidth, 0) || backingImageWidth < 0 || backingImageWidth > MaxWidth {
		return tooBigGif, nil
	}

	// log.InfoLogger.Print("size is valid", dotMatrixWidth, dotMatrixHeight, windowColumns, windowWidth, backingImageColumns, backingImageWidth)

	// compute frame count and delay
	frameCount := int(backingImageColumns)

	reverse := speed < 0
	if reverse {
		speed = -speed
	}

	delay := 100 / (speed * float64(CharWidthInDots))
	if speed == 0 {
		delay = 0
	}

	// log.InfoLogger.Print("timing:", frameCount, reverse, delay)

	// handle overflows
	if math.IsNaN(delay) || math.IsInf(delay, 0) || delay < 0 || delay > math.MaxUint16 {
		delay = 0
	}

	// log.InfoLogger.Print("timing is valid")

	if delay > 0 {
		// delay must be at least 2 otherwise some players won't work
		delay = math.Max(2, math.Round(delay))
	}

	backingImage := image.NewPaletted(
		image.Rect(0, 0, int(backingImageWidth), backingImageHeight),
		palette[:],
	)

	dmtxStride := dotMatrix.Stride
	bimgStride := backingImage.Stride

	// draw dots
	for y := 0; y < dotMatrixHeight; y++ {
		for x := 0; x < dotMatrixWidth; x++ {
			dotState := dotMatrix.Pix[y*dmtxStride+x]

			for dy := 0; dy < DotInnerSize; dy++ {
				for dx := 0; dx < DotInnerSize; dx++ {
					backingImage.Pix[(2*DotPadding+y*DotSize+dy)*bimgStride+(2*DotPadding+x*DotSize+dx)] = dotState
					if x < int(windowColumns) {
						backingImage.Pix[(2*DotPadding+y*DotSize+dy)*bimgStride+(2*DotPadding+(x+int(backingImageColumns))*DotSize+dx)] = dotState
					}
				}
			}
		}
	}

	// log.InfoLogger.Print("text written to backing image")

	dotMatrix = nil

	// draw blank dots
	blankWidth := math.Floor(blank * float64(dotMatrixWidth))
	for y := 0; y < dotMatrixHeight; y++ {
		for x := dotMatrixWidth; x < dotMatrixWidth+int(blankWidth); x++ {
			for dy := 0; dy < DotInnerSize; dy++ {
				for dx := 0; dx < DotInnerSize; dx++ {
					backingImage.Pix[(2*DotPadding+y*DotSize+dy)*bimgStride+(2*DotPadding+x*DotSize+dx)] = 1
					if x < int(windowColumns) {
						backingImage.Pix[(2*DotPadding+y*DotSize+dy)*bimgStride+(2*DotPadding+(x+int(backingImageColumns))*DotSize+dx)] = 1
					}
				}
			}
		}
	}

	// log.InfoLogger.Print("blank written to backing image")

	if delay == 0 {
		subImage := backingImage.SubImage(image.Rect(0, 0, int(windowWidth), backingImageHeight))
		palettedSubImage, ok := subImage.(*image.Paletted)
		if !ok {
			return nil, errUnexpectedSubImageFormat
		}
		return &gif.GIF{
			Image:     []*image.Paletted{palettedSubImage},
			Delay:     []int{0},
			LoopCount: 0,
			Disposal:  []byte{0},
			Config: image.Config{
				ColorModel: color.Palette(palette[:]),
				Width:      palettedSubImage.Rect.Dx(),
				Height:     palettedSubImage.Rect.Dy(),
			},
			BackgroundIndex: 0,
		}, nil
	}

	// determine starting point of animation and blank range
	var column, step, blankRangeStart, blankRangeEnd int

	if reverse {
		column = int(math.Min(backingImageColumns, float64(dotMatrixWidth)+windowColumns) - windowColumns)
		step = -1
		blankRangeStart = column + 1
		blankRangeEnd = int(backingImageColumns-windowColumns) + 1
	} else {
		column = max(dotMatrixWidth, int(backingImageColumns-windowColumns))
		step = 1
		blankRangeStart = dotMatrixWidth
		blankRangeEnd = column
	}

	blankCount := blankRangeEnd - blankRangeStart
	if blankCount < 0 {
		blankCount = 0
	}

	// compress blank frames into one
	if blankCount > 0 {
		frameCount -= (blankCount - 1)
	}

	// compute delay of last (possibly blank) frame
	lastDelay := int(delay)
	if blankCount > 0 {
		lastDelay *= blankCount
	}

	// log.InfoLogger.Print(
	// 	"column:", column,
	// 	"step:", step,
	// 	"blankRangeStart:", blankRangeStart,
	// 	"blankRangeEnd:", blankRangeEnd,
	// 	"blankCount:", blankCount,
	// 	"frameCount:", frameCount,
	//  "delay:", delay,
	// 	"lastDelay:", lastDelay,
	// )

	anim := gif.GIF{
		Image:     make([]*image.Paletted, frameCount),
		Delay:     make([]int, frameCount),
		LoopCount: 0,
		Disposal:  make([]byte, frameCount),
		Config: image.Config{
			ColorModel: color.Palette(palette[:]),
			Width:      int(windowWidth),
			Height:     backingImageHeight,
		},
		BackgroundIndex: 0,
	}

	for i := range anim.Image {
		x := column * DotSize

		subImage, ok := backingImage.SubImage(image.Rect(x, 0, x+int(windowWidth), backingImageHeight)).(*image.Paletted)
		if !ok {
			return nil, errUnexpectedSubImageFormat
		}

		subImage.Rect = image.Rect(0, 0, int(windowWidth), backingImageHeight)

		anim.Image[i] = subImage
		anim.Delay[i] = int(delay)
		anim.Disposal[i] = 0

		column += step
		if column >= int(backingImageColumns) {
			column -= int(backingImageColumns)
		} else if column < 0 {
			column += int(backingImageColumns)
		}
	}

	anim.Delay[frameCount-1] = lastDelay

	// log.InfoLogger.Print("gif generation complete")

	return &anim, nil
}
