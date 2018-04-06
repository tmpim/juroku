package juroku

import (
	"fmt"
	"image"

	"github.com/1lann/imagequant"
)

// Quantize quantizes an image into a maximum of 16 colors with the given
// parameters.
func Quantize(ref, img image.Image, speed int, dither float64) (image.Image, error) {
	attr, err := getAttributes(speed)
	if err != nil {
		return nil, fmt.Errorf("Attribues: %s", err.Error())
	}
	defer attr.Release()

	quant, err := imagequant.NewImage(attr, imagequant.GoImageToRgba32(ref),
		ref.Bounds().Dx(), ref.Bounds().Dy(), 0)
	if err != nil {
		return nil, fmt.Errorf("ref NewImage: %s", err.Error())
	}
	defer quant.Release()

	res, err := quant.Quantize(attr)
	if err != nil {
		return nil, fmt.Errorf("Quantize: %s", err.Error())
	}

	if ref != img {
		outputImg, err := imagequant.NewImage(attr, imagequant.GoImageToRgba32(img),
			img.Bounds().Dx(), img.Bounds().Dy(), 0)
		if err != nil {
			return nil, fmt.Errorf("img NewImage: %s", err.Error())
		}
		defer outputImg.Release()

		res.SetOutputImage(outputImg)
	}

	err = res.SetDitheringLevel(float32(dither))
	if err != nil {
		return nil, fmt.Errorf("SetDitheringLevel: %s", err.Error())
	}

	rgb8data, err := res.WriteRemappedImage()
	if err != nil {
		return nil, fmt.Errorf("WriteRemappedImage: %s", err.Error())
	}

	result := imagequant.Rgb8PaletteToGoImage(res.GetImageWidth(),
		res.GetImageHeight(), rgb8data, res.GetPalette())
	return result, nil
}

func getAttributes(speed int) (*imagequant.Attributes, error) {
	attr, err := imagequant.NewAttributes()
	if err != nil {
		return nil, fmt.Errorf("NewAttributes: %s", err.Error())
	}

	err = attr.SetSpeed(speed)
	if err != nil {
		return nil, fmt.Errorf("SetSpeed: %s", err.Error())
	}

	err = attr.SetMaxColors(16)
	if err != nil {
		return nil, fmt.Errorf("SetMaxColors: %s", err.Error())
	}

	return attr, nil
}
