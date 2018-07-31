package main

import (
	"context"
	"image"
	"log"
	"os"
	"time"

	"github.com/tmpim/juroku"
)

type subImagable interface {
	SubImage(r image.Rectangle) image.Image
}

func main() {
	file, err := os.Open("./loli_dance.webm")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	of, err := os.Create("./dance.juf")
	if err != nil {
		panic(err)
	}
	of.Write([]byte("JUF"))
	of.Write([]byte{0x01, 0x02, 0x04})

	output := make(chan juroku.VideoChunk, 10)

	go func() {
		defer of.Close()

		for out := range output {
			for _, frame := range out.Frames {
				err := frame.WriteTo(of)
				if err != nil {
					log.Println("error writing:", err)
					return
				}
			}

			out.Audio.WriteTo(of)
		}
	}()

	log.Println("result:", juroku.EncodeVideo(file, output, juroku.EncoderOptions{
		Context:     context.Background(),
		Width:       665,
		Height:      366,
		Workers:     8,
		Speed:       10,
		Dither:      0.2,
		AudioBuffer: time.Second * 120,
		Debug:       true,
		Splitter: func(img image.Image) []image.Image {
			sub := img.(subImagable)
			return []image.Image{
				sub.SubImage(image.Rectangle{
					Min: image.Point{
						X: 0,
						Y: 0,
					},
					Max: image.Point{
						X: 328,
						Y: 201,
					},
				}),
				sub.SubImage(image.Rectangle{
					Min: image.Point{
						X: 336,
						Y: 0,
					},
					Max: image.Point{
						X: 664,
						Y: 201,
					},
				}),
				sub.SubImage(image.Rectangle{
					Min: image.Point{
						X: 0,
						Y: 210,
					},
					Max: image.Point{
						X: 328,
						Y: 366,
					},
				}),
				sub.SubImage(image.Rectangle{
					Min: image.Point{
						X: 336,
						Y: 210,
					},
					Max: image.Point{
						X: 664,
						Y: 366,
					},
				}),
			}
		},
	}))
	close(output)

	time.Sleep(10 * time.Second)
}
