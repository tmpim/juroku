package juroku

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/disintegration/gift"
)

func getScore(edges image.Image, x, y int) float64 {
	r, g, b, _ := edges.At(x, y).RGBA()
	return math.Log((float64(r)+float64(g)+float64(b))/3.0+7.0)*0.65 + 0.45
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
	gift.Sobel().Draw(edges, img, &gift.Options{
		Parallelization: true,
	})

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
					col := img.At(x+dx, y+dy).(color.RGBA)
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

			var max colorCount
			var secondMax colorCount

			for k, v := range pixelScore {
				if v > max.weight {
					secondMax = max
					max.weight = v
					max.color = k
				} else if v > secondMax.weight {
					secondMax.weight = v
					secondMax.color = k
				}
			}

			if len(pixelScore) <= 2 {
				// we're gucci
				for _, pix := range pixels {
					output.Set(pix.x, pix.y, pix.color)
				}
				continue
			}

			palette := color.Palette{
				max.color,
				secondMax.color,
			}

			for _, pix := range pixels {
				if pix.color != max.color &&
					pix.color != secondMax.color {
					output.Set(pix.x, pix.y, palette.Convert(pix.color))
				} else {
					output.Set(pix.x, pix.y, pix.color)
				}
			}
		}
	}

	return output, nil
}
