package juroku

import (
	"errors"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/disintegration/gift"
)

// GetPalette returns the palette of the image.
func GetPalette(img image.Image) color.Palette {
	colors := make(map[color.RGBA]bool)

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			colors[color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}] = true
		}
	}

	var palette color.Palette
	for col := range colors {
		palette = append(palette, col)
	}

	return palette
}

func getScore(edges image.Image, x, y int) float64 {
	r, _, _, _ := edges.At(x, y).RGBA()
	return math.Log(float64(r)+7.0)*0.65 + 0.45
}

// ChunkImage chunks an image following the ComputerCraft requirements of
// maximum of 2 colors per 2x3 chunk of pixels and returns it. It is assumed
// that the palette has already been reduced to 16 colors.
func ChunkImage(img image.Image) (image.Image, error) {
	if img.Bounds().Dx()%2 != 0 {
		return nil, errors.New("juroku: image width must be a multiple of 2")
	}

	if img.Bounds().Dy()%3 != 0 {
		return nil, errors.New("juroku: image height must be a multiple of 3")
	}

	edges := image.NewRGBA(img.Bounds())
	g := gift.New(gift.Sobel(), gift.Grayscale())
	g.Draw(edges, img)

	output := image.NewRGBA(img.Bounds())

	type pixel struct {
		color color.RGBA
		image image.Image
		x     int
		y     int
	}

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 3 {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 2 {
			var pixels []pixel
			pixelScore := make(map[color.RGBA]float64)

			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 2; dx++ {
					r, g, b, a := img.At(x+dx, y+dy).RGBA()
					col := color.RGBA{
						R: uint8(r >> 8),
						G: uint8(g >> 8),
						B: uint8(b >> 8),
						A: uint8(a >> 8),
					}
					pixels = append(pixels, pixel{
						color: col,
						image: img,
						x:     x + dx,
						y:     y + dy,
					})
					pixelScore[col] += getScore(edges, x+dx, y+dy)
				}
			}

			type colorCount struct {
				color  color.RGBA
				weight float64
			}

			var aggrPixels []colorCount
			for k, v := range pixelScore {
				aggrPixels = append(aggrPixels, colorCount{
					color:  k,
					weight: v,
				})
			}

			if len(aggrPixels) < 3 {
				// we're gucci
				for _, pix := range pixels {
					output.Set(pix.x, pix.y, pix.color)
				}
				continue
			}

			sort.Slice(aggrPixels, func(i int, j int) bool {
				return aggrPixels[i].weight > aggrPixels[j].weight
			})

			for _, pix := range pixels {
				if pix.color != aggrPixels[0].color &&
					pix.color != aggrPixels[1].color {
					output.Set(pix.x, pix.y,
						color.Palette{
							aggrPixels[0].color,
							aggrPixels[1].color,
						}.Convert(pix.color))
				} else {
					output.Set(pix.x, pix.y, pix.color)
				}
			}
		}
	}

	return output, nil
}
