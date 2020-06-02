package main

import (
	"bytes"
	"fmt"
	_ "image/jpeg"
	"image/png"
	"os"
	"sync"
	"time"

	"github.com/tmpim/juroku"
	_ "golang.org/x/image/bmp"
)

func main() {
	if len(os.Args) != 2 {
		panic("must have path to image")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		panic(err)
	}

	wg := new(sync.WaitGroup)

	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				chunker := &juroku.FastChunker{}

				out, pal, err := juroku.Quantize(img, img, 10, 0.4)
				if err != nil {
					panic(err)
				}

				data, err := chunker.ChunkImage(out, pal)
				if err != nil {
					panic(err)
				}

				buf := new(bytes.Buffer)
				data.WriteTo(buf)
			}
		}()
	}

	start := time.Now()
	wg.Wait()
	fmt.Println("took:", time.Since(start))
}
