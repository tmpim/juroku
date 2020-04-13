package juroku

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
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
	Context             context.Context
	Width               int
	Height              int
	Workers             int
	Speed               int
	Dither              float64
	Debug               bool
	Realtime            bool
	GroupAudioNumFrames int
	Splitter            FrameSplitter
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
func EncodeVideo(input interface{}, output chan<- VideoChunk,
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

	chanBufSize := opts.Workers * 2
	if opts.GroupAudioNumFrames > opts.Workers {
		chanBufSize = opts.GroupAudioNumFrames * 2
	}

	filename := "-"
	if parsedFilename, ok := input.(string); ok {
		filename = parsedFilename
	}

	var args []string
	if opts.Realtime {
		args = append(args, "-re")
	}

	args = append(args, "-f", "lavfi", "-i", "anullsrc", "-probesize", "32", "-analyzeduration", "0",
		"-i", filename, "-acodec", "pcm_s8",
		"-f", "s8", "-ac", "1", "-ar", strconv.Itoa(dfpwm.SampleRate), "-af", "lowpass=f=10000",
		"pipe:3", "-f", "image2pipe", "-vcodec", "bmp",
		"-r", strconv.Itoa(Framerate), "-vf",
		"scale="+strconv.Itoa(opts.Width)+":"+strconv.Itoa(opts.Height),
		"-fflags", "nobuffer", "-flags", "low_delay",
		"-strict", "experimental",
		"pipe:4")

	eg, egCtx := errgroup.WithContext(opts.Context)

	cmd := exec.CommandContext(egCtx, "ffmpeg", args...)
	if filename == "-" {
		cmd.Stdin = input.(io.Reader)
	}
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

	eg.Go(func() error {
		defer log.Println("cmd.Wait is quitting")
		defer stop()
		return cmd.Wait()
	})

	inbox := make(chan frameJob, chanBufSize)
	for i := 0; i < opts.Workers; i++ {
		eg.Go(func() error {
			return jurokuWorker(inbox, opts.Splitter, opts.Speed, opts.Dither)
		})
	}

	outputChan := make(chan chan []*FrameChunk, chanBufSize)

	eg.Go(func() error {
		defer log.Println("decodeToWorkerPump is quitting")
		return decodeToWorkerPump(frameRd, inbox, outputChan)
	})

	// Prepare audio buffer.
	stream := audio.NewOfflineStream(dfpwm.SampleRate, dfpwm.SampleRate/4)

	eg.Go(func() error {
		defer log.Println("audio stream is quitting")
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
		defer log.Println("EncodeDFPWM is quitting")

		if opts.GroupAudioNumFrames == 0 {
			log.Println("standard encode")
			err := dfpwm.EncodeDFPWM(dfpwmWr, stream)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				dfpwmWr.Close()
				return nil
			}

			dfpwmWr.CloseWithError(err)
			return err
		}

		input := make([]int8, (dfpwm.SampleRate/Framerate)*opts.GroupAudioNumFrames)

		for {
			count := 0
			for count < len(input) {
				n, err := stream.Read(input[count:])
				if err == io.EOF {
					dfpwmWr.Close()
					return nil
				} else if err != nil {
					dfpwmWr.CloseWithError(err)
					return err
				}
				count += n
			}

			paddedInput := append(input, make([]int8, 48000*3)...)
			dat := dfpwm.OneOffEncodeDFPWM(paddedInput)
			log.Println("dfpwm wrote:", len(dat))
			dfpwmWr.Write(dat)
		}
	})

	eg.Go(func() error {
		defer log.Println("output pump is quitting")
		defer close(output)

		var frameAudio []byte
		if opts.GroupAudioNumFrames == 0 {
			frameAudio = make([]byte, dfpwm.SampleRate/(Framerate*8))
		} else {
			frameAudio = make([]byte, (dfpwm.SampleRate/(Framerate*8))*opts.GroupAudioNumFrames+((48000*3)/8))
		}

		count := 0

		for frameOutput := range outputChan {
			frames := <-frameOutput

			count++

			if opts.GroupAudioNumFrames == 0 || (count%opts.GroupAudioNumFrames) == 0 {
				_, err := io.ReadFull(dfpwmRd, frameAudio)
				if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
					return fmt.Errorf("juroku: EncodeVideo: audio encode error: %v", err)
				}

				log.Println("read:", len(frameAudio))

				output <- VideoChunk{
					Frames: frames,
					Audio:  frameAudio,
				}
			} else {
				output <- VideoChunk{
					Frames: frames,
				}
			}
		}

		return nil
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
