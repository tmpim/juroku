package juroku

import (
	"errors"
	"io"
	"log"

	"github.com/1lann/dissonance/audio"
)

type PCMEncoder struct {
}

func (e *PCMEncoder) SampleRateBytes() int {
	return 48000
}

func (e *PCMEncoder) SampleRate() int {
	return 48000
}

func (e *PCMEncoder) Encode(stream audio.Stream, wr io.WriteCloser, opts EncoderOptions) error {
	defer log.Println("EncodePCM is quitting")
	log.Println("PCM encoder is starting")

	byteBuf := make([]byte, 1024)
	buf := make([]int8, 1024)

	for {
		offset := 0

		for offset < 1024 {
			n, err := stream.Read(buf[offset:])
			if errors.Is(err, io.EOF) {
				wr.Close()
				return nil
			} else if err != nil {
				wr.Close()
				return err
			}

			offset += n
		}

		for i := range buf {
			byteBuf[i] = byte(buf[i])
		}

		wr.Write(byteBuf)
	}
}
