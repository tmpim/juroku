package juroku

import (
	"io"
	"log"

	"github.com/1lann/dissonance/audio"
	"github.com/tmpim/juroku/dfpwm"
)

type DFPWMEncoder struct {
}

func (e *DFPWMEncoder) SampleRateBytes() int {
	return dfpwm.SampleRate / 8
}

func (e *DFPWMEncoder) Encode(stream audio.Stream, wr io.WriteCloser, opts EncoderOptions) error {
	defer log.Println("EncodeDFPWM is quitting")

	if opts.GroupAudioNumFrames == 0 {
		log.Println("standard encode")
		err := dfpwm.EncodeDFPWM(wr, stream)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			wr.Close()
			return nil
		}

		wr.Close()
		return err
	}

	dataLength := (dfpwm.SampleRate / opts.Framerate) * opts.GroupAudioNumFrames
	bufferLength := dfpwm.SampleRate * 3
	totalLength := dataLength + bufferLength
	zeros := make([]int8, totalLength)
	input := make([]int8, totalLength)

	for {
		count := 0
		for count < dataLength {
			n, err := stream.Read(input[count:dataLength])
			count += n
			if err == io.EOF {
				copy(input[count:], zeros)
				dat := dfpwm.OneOffEncodeDFPWM(input)
				wr.Write(dat)
				log.Println("dfpwm final write:", len(dat))
				wr.Close()
				return nil
			} else if err != nil {
				wr.Close()
				return err
			}
		}

		dat := dfpwm.OneOffEncodeDFPWM(input)
		log.Println("dfpwm wrote:", len(dat))
		wr.Write(dat)
	}

	return nil
}
