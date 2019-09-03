package juroku

import (
	"bytes"
	"errors"
	"image"
	"image/color"
)

var colorAlphabet = "0123456789abcdef"

// GenerateFrameChunk generates a frame chunk for the given image.
func GenerateFrameChunk(img image.Image) (*FrameChunk, error) {
	palette := GetPalette(img)
	if len(palette) > 16 {
		return nil, errors.New("juroku: GenerateFrameChunk: palette must have <= 16 colors")
	}

	frame := &FrameChunk{
		Width:  img.Bounds().Dx() / 2,
		Height: img.Bounds().Dy() / 3,
	}

	paletteToColor := make(map[color.RGBA]byte)
	for i, col := range palette {
		paletteToColor[col.(color.RGBA)] = byte(i)
	}

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 3 {
		row := FrameRow{
			TextColor:       new(bytes.Buffer),
			BackgroundColor: new(bytes.Buffer),
			Text:            new(bytes.Buffer),
		}

		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 2 {
			chunk := make([]byte, 0, 6)
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 2; dx++ {
					chunk = append(chunk,
						paletteToColor[img.At(x+dx, y+dy).(color.RGBA)])
				}
			}

			text, textColor, bgColor := chunkToBlit(chunk)
			row.TextColor.WriteByte(colorAlphabet[textColor])
			row.BackgroundColor.WriteByte(colorAlphabet[bgColor])
			row.Text.WriteByte(text)
		}

		frame.Rows = append(frame.Rows, row)
	}

	for i := range frame.Palette {
		if len(palette) <= i {
			// Unused color
			frame.Palette[i] = color.RGBA{}
		} else {
			frame.Palette[i] = palette[i].(color.RGBA)
		}
	}

	return frame, nil
}

func chunkToBlit(chunk []byte) (char byte, textColor byte, bgColor byte) {
	bgColor = chunk[5]

	var b byte
	var i byte
	for i = 0; i < 6; i++ {
		if chunk[i] != bgColor {
			textColor = chunk[i]
			b |= 1 << i
		} else {
			b |= 0 << i
		}
	}

	if textColor == 0 {
		textColor = '0'
	}

	char = b + 128
	return
}
