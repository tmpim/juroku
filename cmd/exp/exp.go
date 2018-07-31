package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/tmpim/juroku"
)

func main() {
	file, err := os.Open("./again.mkv")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	output := make(chan juroku.VideoChunk, 10)

	go func() {
		for out := range output {
			// if out.Frame != nil {
			// 	log.Println("visual:", len(out.Frame.Pixels))
			// }
			log.Println("audio:", len(out.Audio))
		}
	}()

	log.Println("result:", juroku.EncodeVideo(file, output, juroku.EncoderOptions{
		Context:     context.Background(),
		Width:       328,
		Height:      189,
		Workers:     10,
		Speed:       10,
		Dither:      0.2,
		AudioBuffer: time.Second * 30,
		Debug:       true,
	}))
}
