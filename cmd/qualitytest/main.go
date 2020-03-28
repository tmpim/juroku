package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"

	"github.com/disintegration/gift"
	"github.com/tmpim/juroku"
)

func main() {
	f, err := os.Create("./cpuprof.out")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer f.Close() // error handling omitted for example
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()
	defer fmt.Println("goodbye")

	files, err := ioutil.ReadDir("./input_test")
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		convert(filepath.Base(f.Name()))
	}
}

func convert(name string) {
	start := time.Now()

	img := image.NewRGBA(image.Rect(0, 0, 650, 366))

	func() {
		var orig image.Image
		input, err := os.Open("./input_test/" + name)
		if err != nil {
			log.Println("Failed to open image:", err)
			os.Exit(1)
		}
		defer input.Close()

		orig, _, err = image.Decode(input)
		if err != nil {
			log.Println("Failed to decode image:", name, err)
			os.Exit(1)
		}

		log.Println("read+decode:", time.Since(start))

		gift.Resize(650, 366, gift.LanczosResampling).Draw(img, orig, &gift.Options{
			Parallelization: false,
		})
	}()

	log.Println("resize:", time.Since(start))

	quant, palette, err := juroku.Quantize(img, img, 5, 0.2)
	if err != nil {
		log.Println("Failed to quantize image:", err)
		os.Exit(1)
	}

	log.Println("quant:", time.Since(start))

	chunked, err := juroku.ChunkImage(quant)
	if err != nil {
		log.Println("Failed to chunk image:", err)
		os.Exit(1)
	}

	log.Println("chunk:", time.Since(start))

	_, err = juroku.GenerateFrameChunk(chunked, palette)
	if err != nil {
		log.Println("Failed to generate code:", err)
		os.Exit(1)
	}

	log.Println("[complete] generate:", time.Since(start))

	basename := strings.TrimSuffix(name, filepath.Ext(name))
	preview, err := os.Create("./output_test_2/" + basename + ".png")
	if err != nil {
		log.Println("Warning: Failed to create preview image:", err)
		return
	}

	defer preview.Close()

	err = png.Encode(preview, chunked)
	if err != nil {
		log.Println("Warning: Failed to encode preview image:", err)
	}
}
