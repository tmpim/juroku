package juroku

import (
	"image"
	"image/color"
)

// FastChunker represents an image chunker that focuses on quality
type FastChunker struct{}

var colorAlphabet = []byte("0123456789abcdef")

// ChunkImage chunks an image, but super quickly!
func (c *FastChunker) ChunkImage(img image.Image, palette color.Palette) (*FrameChunk, error) {
	// [textColor][bgColor][colorQuery] -> 1 (text color) or 0 (bg color)
	colorToCode := make([]byte, 33554431)

	var result FrameChunk

	for i, col := range palette {
		rgb := col.(color.RGBA)
		result.Palette[i] = rgb
		colorToCode[(int(rgb.R)<<16)|(int(rgb.G)<<8)|int(rgb.B)] = colorAlphabet[i]
	}

	startY := img.Bounds().Min.Y
	endY := img.Bounds().Max.Y
	startX := img.Bounds().Min.X
	endX := img.Bounds().Max.X

	result.Rows = make([]FrameRow, (endY-startY)/3)

	for chunkY := startY; chunkY < endY; chunkY += 3 {
		row := chunkY / 3
		result.Rows[row].BgColor = make([]byte, (endX-startX)/2)
		result.Rows[row].TextColor = make([]byte, (endX-startX)/2)
		result.Rows[row].Text = make([]byte, (endX-startX)/2)

		for chunkX := startX; chunkX < endX; chunkX += 2 {
			col := chunkX / 2
			// auto-accept bg color for performance
			bgRGB := img.At(chunkX+1, chunkY+2).(color.RGBA)
			result.Rows[row].BgColor[col] = colorToCode[(int(bgRGB.R)<<16)|(int(bgRGB.G)<<8)|int(bgRGB.B)]
			result.Rows[row].TextColor[col] = '0'
			result.Rows[row].Text[col] = 128
			hasTextColor := false

			subpixel := 0
			for y := chunkY; y < chunkY+3; y++ {
				for x := chunkX; x < chunkX+2; x++ {
					var textRGB color.RGBA
					pixelRGB := img.At(x, y).(color.RGBA)
					bgDelta := deltaPixel(bgRGB, pixelRGB)

					if hasTextColor {
						textDelta := deltaPixel(textRGB, pixelRGB)
						if textDelta < bgDelta {
							result.Rows[row].Text[col] |= (1 << subpixel)
						}
					} else {
						if bgDelta >= 15 {
							hasTextColor = true
							textRGB = pixelRGB
							result.Rows[row].TextColor[col] = colorToCode[(int(textRGB.R)<<16)|(int(textRGB.G)<<8)|int(textRGB.B)]
							result.Rows[row].Text[col] |= (1 << subpixel)
						}
					}

					subpixel++
				}
			}
		}
	}

	return &result, nil
}

func deltaPixel(a color.RGBA, b color.RGBA) int {
	return abs(int(a.R-b.R)) + abs(int(a.G-b.G)) + abs(int(a.B-b.B))
}

func abs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}
