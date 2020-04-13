package juroku

import (
	"image"
	"image/color"
	"math"

	"github.com/disintegration/gift"
	"github.com/lucasb-eyer/go-colorful"
)

// FastChunker represents an image chunker that focuses on quality
type FastChunker struct{}

var colorAlphabet = []byte("0123456789abcdef")

func getScore(edges image.Image, x, y int) float64 {
	r, g, b, _ := edges.At(x, y).RGBA()
	return math.Log((float64(r)+float64(g)+float64(b))/3.0+7.0)*0.65 + 0.45
}

// ChunkImage chunks an image, but super quickly!
func (c *FastChunker) ChunkImage(img image.Image, palette color.Palette) (*FrameChunk, error) {
	// [textColor][bgColor][colorQuery] -> 1 (text color) or 0 (bg color)
	colorToPos := make(map[struct{ R, G, B byte }]int)

	var distanceTable [16][16]float64

	var result FrameChunk

	edges := image.NewRGBA(img.Bounds())
	gift.Sobel().Draw(edges, img, &gift.Options{
		Parallelization: false,
	})

	for i, col := range palette {
		rgb := col.(color.RGBA)
		result.Palette[i] = rgb
		colorToPos[struct{ R, G, B byte }{rgb.R, rgb.G, rgb.B}] = i

		if i < len(palette)-1 {
			for j := i + 1; j < len(palette); j++ {
				secondRGB := palette[j].(color.RGBA)
				distance := colorful.Color{
					R: float64(rgb.R)/255.0 + 0.5,
					G: float64(rgb.G)/255.0 + 0.5,
					B: float64(rgb.B)/255.0 + 0.5,
				}.DistanceLab(colorful.Color{
					R: float64(secondRGB.R)/255.0 + 0.5,
					G: float64(secondRGB.G)/255.0 + 0.5,
					B: float64(secondRGB.B)/255.0 + 0.5,
				})
				distanceTable[i][j] = distance
				distanceTable[j][i] = distance
			}
		}
	}

	startY := img.Bounds().Min.Y
	endY := img.Bounds().Max.Y
	startX := img.Bounds().Min.X
	endX := img.Bounds().Max.X

	result.Width = (endX - startX) / 2
	result.Height = (endY - startY) / 3

	result.Rows = make([]FrameRow, (endY-startY)/3)
	var pixelChunk [6]int

	for chunkY := startY; chunkY < endY; chunkY += 3 {
		row := chunkY / 3
		result.Rows[row].BgColor = make([]byte, result.Width)
		result.Rows[row].TextColor = make([]byte, result.Width)
		result.Rows[row].Text = make([]byte, result.Width)

		for chunkX := startX; chunkX < endX; chunkX += 2 {
			col := chunkX / 2
			// auto-accept bg color for performance
			// bgRGB := img.At(chunkX+1, chunkY+2).(color.RGBA)
			// result.Rows[row].BgColor[col] = colorToPos[(int(bgRGB.R)<<16)|(int(bgRGB.G)<<8)|int(bgRGB.B)]
			// result.Rows[row].TextColor[col] = '0'
			// result.Rows[row].Text[col] = 128

			var chunk byte = 128
			colorScore := make([]float64, 6)

			subpixel := 0
			for y := chunkY; y < chunkY+3; y++ {
				for x := chunkX; x < chunkX+2; x++ {
					// var textRGB color.RGBA
					pixelRGB := img.At(x, y).(color.RGBA)
					pixelChunk[subpixel] = colorToPos[struct{ R, G, B byte }{pixelRGB.R, pixelRGB.G, pixelRGB.B}]

					for i := 0; i < subpixel+1; i++ {
						if pixelChunk[i] == pixelChunk[subpixel] {
							colorScore[i] += getScore(edges, x, y)
							break
						}
					}

					subpixel++

					// bgDelta := deltaPixel(bgRGB, pixelRGB)

					// if hasTextColor {
					// 	textDelta := deltaPixel(textRGB, pixelRGB)
					// 	if textDelta < bgDelta {
					// 		result.Rows[row].Text[col] |= (1 << subpixel)
					// 	}
					// } else {
					// 	if bgDelta >= 15 {
					// 		hasTextColor = true
					// 		textRGB = pixelRGB
					// 		result.Rows[row].TextColor[col] = colorToPos[(int(textRGB.R)<<16)|(int(textRGB.G)<<8)|int(textRGB.B)]
					// 		result.Rows[row].Text[col] |= (1 << subpixel)
					// 	}
					// }

					// subpixel++
				}
			}

			var top1 float64
			var top1Color int
			var top2 float64
			var top2Color int

			for col, score := range colorScore {
				if score > top1 {
					top2 = top1
					top2Color = top1Color
					top1Color = pixelChunk[col]
					top1 = score
				} else if score > top2 {
					top2Color = pixelChunk[col]
					top2 = score
				}
			}

			// var top1Color int
			// var top2Color int
			// var bestDistance float64

			// for firstColor := 0; firstColor < 5; firstColor++ {
			// 	for secondColor := firstColor + 1; secondColor < 6; secondColor++ {
			// 		dist := distanceTable[pixelChunk[firstColor]][pixelChunk[secondColor]]
			// 		if dist >= bestDistance {
			// 			bestDistance = dist
			// 			top1Color = pixelChunk[firstColor]
			// 			top2Color = pixelChunk[secondColor]
			// 		}
			// 	}
			// }

			// log.Println("best colors:", top1Color, top2Color, colorScore)

			var bgColor, textColor int

			if distanceTable[pixelChunk[5]][top1Color] < distanceTable[pixelChunk[5]][top2Color] {
				bgColor = top1Color
				textColor = top2Color
			} else {
				bgColor = top2Color
				textColor = top1Color
			}

			for i, p := range pixelChunk {
				if distanceTable[p][textColor] < distanceTable[p][bgColor] {
					chunk |= (1 << i)
				}
			}

			result.Rows[row].Text[col] = chunk
			result.Rows[row].BgColor[col] = colorAlphabet[bgColor]
			result.Rows[row].TextColor[col] = colorAlphabet[textColor]
		}
	}

	// d, err := json.Marshal(result)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println("size:", len(d))

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
