package juroku

import (
	"image"
	"image/color"
)

// // PixelChunk represents a Juroku processed chunk of pixels.
// // structure is as follows from LSB to MSB:
// // 6 bits: blit order foreground/background color for a single chunk
// // foreground = 1, background = 0
// // 4 bits: foreground color palette location
// // 4 bits: background color palette location
// type PixelChunk uint16

// const (
// 	backgroundColorMask = 0b11110000000000
// 	foregroundColorMask = 0b00001111000000
// 	chunkMask           = 0b00000000111111

// 	backgroundColorShift = 10
// 	foregroundColorShift = 6
// )

// func (p PixelChunk) Blit() (char byte, textColor byte, bgColor byte) {
// 	bgColor = (p & backgroundColorMask) >> backgroundColorShift
// 	textColor = (p & foregroundColorMask) >> foregroundColorShift
// 	char = (p & chunkMask) + 128
// 	return

// }

// ImageChunker represents an image chunker.
type ImageChunker interface {
	ChunkImage(img image.Image, palette color.Palette) ([]byte, error)
}
