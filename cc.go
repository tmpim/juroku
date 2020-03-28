package juroku

// // GenerateFrameChunk generates a frame chunk for the given image.
// func GenerateFrameChunk(img image.Image, palette color.Palette) (*FrameChunk, error) {
// 	if len(palette) > 16 {
// 		return nil, errors.New("juroku: GenerateFrameChunk: palette must have <= 16 colors")
// 	}

// 	frame := &FrameChunk{
// 		Width:  img.Bounds().Dx() / 2,
// 		Height: img.Bounds().Dy() / 3,
// 	}

// 	paletteToColor := make(map[color.RGBA]byte)
// 	for i, col := range palette {
// 		paletteToColor[col.(color.RGBA)] = byte(i)
// 	}

// 	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 3 {
// 		row := FrameRow{}

// 		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 2 {
// 			chunk := make([]byte, 0, 6)
// 			for dy := 0; dy < 3; dy++ {
// 				for dx := 0; dx < 2; dx++ {
// 					chunk = append(chunk,
// 						paletteToColor[img.At(x+dx, y+dy).(color.RGBA)])
// 				}
// 			}

// 			text, textColor, bgColor := chunkToBlit(chunk)

// 			row.Color = append(row.Color, colorAlphabet[textColor])
// 			row.BackgroundColor = append(row.BackgroundColor, colorAlphabet[bgColor])
// 			row.Text = append(row.Text, text)
// 		}

// 		frame.Rows = append(frame.Rows, &row)
// 	}

// 	for i := range frame.Palette {
// 		if len(palette) <= i {
// 			// Unused color
// 			frame.Palette[i] = color.RGBA{}
// 		} else {
// 			frame.Palette[i] = palette[i].(color.RGBA)
// 		}
// 	}

// 	return frame, nil
// }

// func chunkToBlit(chunk []byte) (char byte, textColor byte, bgColor byte) {
// 	bgColor = chunk[5]

// 	var b byte
// 	var i byte
// 	for i = 0; i < 6; i++ {
// 		if chunk[i] != bgColor {
// 			textColor = chunk[i]
// 			b |= 1 << i
// 		} else {
// 			b |= 0 << i
// 		}
// 	}

// 	char = b + 128
// 	return
// }
