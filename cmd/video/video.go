package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"sync"

	"github.com/tmpim/juroku"

	_ "image/png"
)

func main() {
	output := make([][]byte, 1800)
	wg := new(sync.WaitGroup)

	wg.Add(1800)

	for i := 1; i <= 1800; i++ {
		go addFrame(output, i, wg)
	}

	wg.Wait()

	fmt.Println("Done! Compiling file...")

	file, err := os.Create("./video.juf")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for _, out := range output {
		_, err := file.Write(out)
		if err != nil {
			panic(err)
		}
	}
}

func addFrame(output [][]byte, i int, wg *sync.WaitGroup) {
	defer wg.Done()

	f, err := os.Open("ffmpeg_" + strconv.Itoa(i) + ".png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	quant, err := juroku.Quantize(img, img, 5, 0.1)
	if err != nil {
		panic(err)
	}

	chunked, err := juroku.ChunkImage(quant)
	if err != nil {
		panic(err)
	}

	code, err := juroku.GenerateCode(chunked)
	if err != nil {
		panic(err)
	}

	output[i-1] = code
}
