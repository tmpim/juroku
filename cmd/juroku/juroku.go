package main

import (
	"flag"
	"image"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/1lann/imagequant"
	"github.com/1lann/juroku"
)

var (
	outputPath  = flag.String("o", "image.lua", "set location of output script")
	previewPath = flag.String("p", "preview.png", "set location of output preview (will be PNG)")
	speed       = flag.Int("q", 1, "set the processing speed/quality (1 = slowest, 10 = fastest)")
	dither      = flag.Float64("d", 0.2, "set the amount of allowed dithering (0 = none, 1 = most)")
	license     = flag.Bool("license", false, "show licensing disclaimers and exit")
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	if *license {
		log.Println("Juroku itself is licensed under the MIT license, which can be found here:")
		log.Println("https://github.com/1lann/juroku/blob/master/LICENSE")
		log.Println("However other portions of Juroku are under different licenses,")
		log.Println("the information of which can be found below.")
		log.Println(imagequant.License())
		os.Exit(0)
	}

	if *speed < 1 {
		log.Println("Speed cannot be less than 1.")
		os.Exit(1)
	}

	if *speed > 10 {
		log.Println("Speed cannot be greater than 10.")
		os.Exit(1)
	}

	if *dither < 0.0 {
		log.Println("Dither cannot be less than 0.")
		os.Exit(1)
	}

	if *dither > 1.0 {
		log.Println("Dither cannot be greater than 1.")
		os.Exit(1)
	}

	if flag.Arg(0) == "" {
		log.Println("Usage: juroku [options] input_image")
		log.Println("")
		log.Println("Juroku converts an image (PNG or JPG) into a Lua script that can be")
		log.Println("loaded as a ComputerCraft API to be used to draw on terminals and monitors.")
		log.Println("Images are not automatically downscaled or cropped.")
		log.Println("")
		log.Println("input_image must have a height that is a multiple of 3 in pixels,")
		log.Println("and a width that is a multiple of 2 in pixels.")
		log.Println("")
		log.Println("Options:")
		flag.PrintDefaults()
		log.Println("")
		log.Println("Disclaimer:")
		log.Println("  Juroku contains code licensed under GPLv3 which is subject to certain restrictions.")
		log.Println("  For full details and to view the full license, run `juroku -license`.")
		os.Exit(1)
	}

	start := time.Now()

	var img image.Image

	func() {
		input, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Println("Failed to open image:", err)
			os.Exit(1)
		}
		defer input.Close()

		img, _, err = image.Decode(input)
		if err != nil {
			log.Println("Failed to decode image:", err)
			os.Exit(1)
		}

		if img.Bounds().Dy()%3 != 0 {
			log.Println("Image height must be a multiple of 3.")
			os.Exit(1)
		}

		if img.Bounds().Dx()%2 != 0 {
			log.Println("Image width must be a multiple of 2.")
			os.Exit(1)
		}
	}()

	log.Println("Image loaded, quantizing...")

	quant, err := juroku.Quantize(img, *speed, *dither)
	if err != nil {
		log.Println("Failed to quantize image:", err)
		os.Exit(1)
	}

	log.Println("Image quantized, chunking and generating code...")

	chunked, err := juroku.ChunkImage(quant)
	if err != nil {
		log.Println("Failed to chunk image:", err)
		os.Exit(1)
	}

	code, err := juroku.GenerateCode(chunked)
	if err != nil {
		log.Println("Failed to generate code:", err)
		os.Exit(1)
	}

	func() {
		preview, err := os.Create(*previewPath)
		if err != nil {
			log.Println("Warning: Failed to create preview image:", err)
		}

		defer preview.Close()

		err = png.Encode(preview, chunked)
		if err != nil {
			log.Println("Warning: Failed to encode preview image:", err)
		}
	}()

	err = ioutil.WriteFile(*outputPath, code, 0644)
	if err != nil {
		log.Println("Failed to write to output file:", err)
		os.Exit(1)
	}

	log.Println("\nDone! That took " + time.Since(start).String() + ".")
	log.Printf("Code outputted to \"%s\", preview outputted to \"%s\".\n",
		*outputPath, *previewPath)
}
