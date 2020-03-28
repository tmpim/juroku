package juroku

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/1lann/dissonance/audio"
	"github.com/tmpim/juroku/dfpwm"
	"golang.org/x/image/bmp"
	"golang.org/x/sync/errgroup"

	_ "image/png"
)

const Framerate = 10

// VideoChunk is composed of a Frame and Audio chunk.
type VideoChunk struct {
	Frames []*FrameChunk
	Audio  AudioChunk
}

// FrameSplitter splits an image into multiple frames for multiple monitors.
type FrameSplitter func(img image.Image) []image.Image

// AudioChunk represents a chunk of audio data.
type AudioChunk []byte

func (a AudioChunk) WriteTo(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, uint32(len(a)))
	if err != nil {
		return err
	}

	_, err = w.Write(a)
	return err
}

type EncoderOptions struct {
	Context  context.Context
	Width    int
	Height   int
	Workers  int
	Speed    int
	Dither   float64
	Debug    bool
	Splitter FrameSplitter
}

func (e *EncoderOptions) validate() error {
	if e.Context == nil {
		return errors.New("juroku: EncodeVideo: context must be specified")
	}
	if e.Width == 0 {
		return errors.New("juroku: EncodeVideo: width must be specified")
	}
	if e.Height == 0 {
		return errors.New("juroku: EncodeVideo: height must be specified")
	}

	return nil
}

// EncodeVideo encodes the video from the given reader which can be of any
// format that FFMPEG supports, into the output.
func EncodeVideo(rd io.Reader, output chan<- VideoChunk,
	opts EncoderOptions) error {
	if err := opts.validate(); err != nil {
		return err
	}

	frameRd, frameWr, err := os.Pipe()
	if err != nil {
		return err
	}

	audioRd, audioWr, err := os.Pipe()
	if err != nil {
		return err
	}

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			frameWr.Close()
			audioWr.Close()
		})
	}

	cmd := exec.CommandContext(opts.Context,
		"ffmpeg", "-i", "-", "-acodec", "pcm_s8",
		"-f", "s8", "-ac", "1", "-ar", strconv.Itoa(dfpwm.SampleRate),
		"pipe:3", "-f", "image2pipe", "-vcodec", "bmp",
		"-r", strconv.Itoa(Framerate), "-vf",
		"scale="+strconv.Itoa(opts.Width)+":"+strconv.Itoa(opts.Height),
		"pipe:4")
	cmd.Stdin = rd
	cmd.ExtraFiles = []*os.File{audioWr, frameWr}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	defer cmd.Process.Kill()

	if opts.Debug {
		go io.Copy(os.Stderr, stdErr)
	} else {
		go io.Copy(ioutil.Discard, stdErr)
	}

	eg, egCtx := errgroup.WithContext(opts.Context)

	eg.Go(func() error {
		return cmd.Wait()
	})

	inbox := make(chan frameJob, opts.Workers*3)
	for i := 0; i < opts.Workers; i++ {
		eg.Go(func() error {
			return jurokuWorker(inbox, opts.Splitter, opts.Speed, opts.Dither)
		})
	}

	outputChan := make(chan chan []*FrameChunk, opts.Workers*3)

	eg.Go(func() error {
		return decodeToWorkerPump(frameRd, inbox, outputChan)
	})

	// Prepare audio buffer.
	stream := audio.NewOfflineStream(dfpwm.SampleRate, dfpwm.SampleRate/4)

	eg.Go(func() error {
		defer stream.Close()
		err := stream.ReadBytes(audioRd, binary.BigEndian, audio.Int8)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		} else if err != nil {
			return err
		}

		return nil
	})

	dfpwmRd, dfpwmWr := io.Pipe()
	eg.Go(func() error {
		err := dfpwm.EncodeDFPWM(dfpwmWr, stream)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			dfpwmWr.Close()
			return nil
		}

		dfpwmWr.CloseWithError(err)
		return err
	})

	eg.Go(func() error {
		defer stop()

		for {
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			case frameOutput, more := <-outputChan:
				if !more {
					return nil
				}

				frames := <-frameOutput

				frameAudio := make([]byte, dfpwm.SampleRate/(Framerate*8))
				n, err := io.ReadFull(dfpwmRd, frameAudio)
				if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
					return fmt.Errorf("juroku: EncodeVideo: audio encode error: %v", err)
				}

				output <- VideoChunk{
					Frames: frames,
					Audio:  frameAudio[:n],
				}
			}
		}
	})

	return eg.Wait()
}

func decodeToWorkerPump(frameRd io.Reader, inbox chan frameJob,
	outputChan chan chan []*FrameChunk) error {
	defer close(inbox)
	defer close(outputChan)

	for {
		img, err := bmp.Decode(frameRd)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		} else if err != nil {
			return err
		}

		frameOutput := make(chan []*FrameChunk, 1)
		outputChan <- frameOutput

		inbox <- frameJob{
			img:    img,
			output: frameOutput,
		}
	}
}

type frameJob struct {
	img    image.Image
	output chan<- []*FrameChunk
}

func jurokuWorker(inbox <-chan frameJob, splitter FrameSplitter,
	speed int, dither float64) error {
	chunker := &FastChunker{}

	for job := range inbox {
		imgs := splitter(job.img)

		var frames []*FrameChunk
		for _, img := range imgs {
			quant, palette, err := Quantize(img, img, speed, dither)
			if err != nil {
				close(job.output)
				return err
			}

			frame, err := chunker.ChunkImage(quant, palette)
			if err != nil {
				close(job.output)
				return err
			}

			frames = append(frames, frame)
		}

		job.output <- frames
	}

	return nil
}
