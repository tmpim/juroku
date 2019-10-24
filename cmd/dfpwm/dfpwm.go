package main

import (
	"fmt"
	"io"
	"os"

	"github.com/1lann/dissonance/drivers/paudio"
	"github.com/1lann/dissonance/ffmpeg"
	"github.com/tmpim/juroku/dfpwm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("You must specify a file to read from")
		os.Exit(1)
	}

	stream, err := ffmpeg.NewFFMPEGStreamFromFile(os.Args[1], false)
	if err != nil {
		panic(err)
	}

	p, _ := paudio.NewPlaybackDevice()
	// p.PlayStream(result)

	rd, wr := io.Pipe()

	go func() {
		dfpwm.EncodeDFPWM(wr, stream)
		wr.Close()
	}()
	dec := dfpwm.NewDecoder(rd, 48000)

	// file, err := os.Create("./outfile")
	// if err != nil {
	// 	panic(err)
	// }
	// defer file.Close()

	// log.Println(dfpwm.EncodeDFPWM(file, stream))

	// file, err := os.Open("outfile")
	// if err != nil {
	// 	panic(err)
	// }

	// defer file.Close()

	// dec := dfpwm.NewDecoder(file, 48000)
	p.PlayStream(dec)
}
