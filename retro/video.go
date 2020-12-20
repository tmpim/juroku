package retro

import (
	"image"
	"image/color"
	"unsafe"

	"github.com/libretro/ludo/libretro"
)

func (c *RetroCore) refresh(data unsafe.Pointer, width, height, pitch int32) {
	c.drawCallback(NewVideoFrame(c.env.pixelFormat, int(width), int(height),
		int(pitch), data))
}

type RetroVideoFrame struct {
	pixelFormat uint32
	width       int
	height      int
	pitch       int
	data        []byte
}

func NewVideoFrame(pixelFormat uint32, width, height, pitch int, data unsafe.Pointer) *RetroVideoFrame {
	frameData := make([]byte, height*pitch)
	copy(frameData, (*[1 << 24]byte)(data)[:height*pitch])

	return &RetroVideoFrame{
		pixelFormat: pixelFormat,
		width:       width,
		height:      height,
		pitch:       pitch,
		data:        frameData,
	}
}

func (f *RetroVideoFrame) ColorModel() color.Model {
	return color.RGBAModel
}

func (f *RetroVideoFrame) Bounds() image.Rectangle {
	return image.Rect(0, 0, f.width, f.height)
}

func (f *RetroVideoFrame) At(x, y int) color.Color {
	switch f.pixelFormat {
	case libretro.PixelFormatRGB565:
		offset := y*f.pitch + x*2
		pixel := int(f.data[offset+1])<<8 | int(f.data[offset])
		return color.RGBA{
			R: uint8(float64((pixel>>11)&0b11111) * (255.0 / 0b11111)),
			G: uint8(float64((pixel>>5)&0b111111) * (255.0 / 0b111111)),
			B: uint8(float64(pixel&0b11111) * (255.0 / 0b11111)),
			A: 255,
		}
	case libretro.PixelFormatXRGB8888:
		offset := y*f.pitch + x*4
		return color.RGBA{
			R: f.data[offset+2],
			G: f.data[offset+1],
			B: f.data[offset],
			A: f.data[offset+3],
		}
	default:
		panic("juroku retro: RetroVideoFrame: unsupported pixel format")
	}
}
