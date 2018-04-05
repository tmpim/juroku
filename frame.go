package juroku

import (
	"bufio"
	"encoding/binary"
	"image/color"
	"io"
)

// FrameChunk represents a frame chunk.
type FrameChunk struct {
	Width  int
	Height int

	Pixels [][2]byte

	Palette [16]color.RGBA
}

// WriteTo writes the frame chunk to a writer.
func (f *FrameChunk) WriteTo(w io.Writer) error {
	wr := bufio.NewWriter(w)

	binary.Write(wr, binary.BigEndian, uint16(f.Width))
	binary.Write(wr, binary.BigEndian, uint16(f.Height))

	for _, pixel := range f.Pixels {
		wr.Write(pixel[:])
	}

	for _, color := range f.Palette {
		wr.Write([]byte{color.R, color.G, color.B})
	}

	return wr.Flush()
}
