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
	"time"

	"github.com/1lann/dissonance/audio"
	"github.com/tmpim/juroku/dfpwm"
	"golang.org/x/image/bmp"
)

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
	Context     context.Context
	Width       int
	Height      int
	Workers     int
	Speed       int
	Dither      float64
	AudioBuffer time.Duration
	Debug       bool
	Splitter    FrameSplitter
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
	if e.AudioBuffer == 0 {
		return errors.New("juroku: EncodeVideo: audio buffer must be specified")
	}
	if e.AudioBuffer < 5*time.Second {
		return errors.New("juroku: EncodeVideo: audio buffer must be no smaller than 5 seconds")
	}
	if e.AudioBuffer > 5*time.Minute {
		return errors.New("juroku: EncodeVideo: audio buffer must be no greater than 5 minutes")
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

	cmd := exec.CommandContext(opts.Context,
		"ffmpeg", "-i", "-", "-acodec", "pcm_s8",
		"-f", "s8", "-ac", "1", "-ar", strconv.Itoa(dfpwm.SampleRate),
		"pipe:3", "-f", "image2pipe", "-vcodec", "bmp", "-r", "5", "-vf",
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

	errChan := make(chan error, 10)
	go func() {
		err := cmd.Wait()
		frameWr.Close()
		audioWr.Close()
		if err != nil {
			errChan <- err
		}
	}()

	// frameBuffer := make(chan image.)

	inbox := make(chan frameJob, opts.Workers*2)
	for i := 0; i < opts.Workers; i++ {
		go jurokuWorker(inbox, opts.Splitter, opts.Speed, opts.Dither)
	}

	outputChan := make(chan chan framesOrError,
		int((opts.AudioBuffer.Seconds()+1.0)*100.0))

	go decodeToWorkerPump(frameRd, inbox, outputChan, errChan)

	// Prepare audio buffer.
	stream := audio.NewOfflineStream(dfpwm.SampleRate, dfpwm.SampleRate/4)
	go func() {
		defer stream.Close()

		err := stream.ReadBytes(audioRd, binary.BigEndian, audio.Int8)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		} else if err != nil {
			errChan <- err
		}
	}()

	dfpwmRd, dfpwmWr := io.Pipe()
	go func() {
		err := dfpwm.EncodeDFPWM(dfpwmWr, stream)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			dfpwmWr.Close()
			return
		} else {
			dfpwmWr.CloseWithError(err)
			return
		}
	}()

	// Buffer the initial audio.
	bufferBytes := int(float64(dfpwm.SampleRate/8) * opts.AudioBuffer.Seconds())
	initialBuffer := make([]byte, bufferBytes)
	n, err := io.ReadFull(dfpwmRd, initialBuffer)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		select {
		case err := <-errChan:
			return err
		default:
		}
	} else if err != nil {
		return fmt.Errorf("juroku: EncodeVideo: failed to read initial audio buffer: %v", err)
	}

	output <- VideoChunk{
		Frames: nil,
		Audio:  initialBuffer[:n],
	}

	// Drain the outputs.
	defer func() {
		go io.Copy(ioutil.Discard, dfpwmRd)

		go func() {
			for range outputChan {
			}
		}()
	}()

	for {
		select {
		case <-opts.Context.Done():
			return opts.Context.Err()
		case frameOutput, more := <-outputChan:
			if !more {
				return nil
			}

			frame := <-frameOutput
			if frame.err != nil {
				return fmt.Errorf("juroku: EncodeVideo: frame encode error: %v", err)
			}

			frameAudio := make([]byte, dfpwm.SampleRate/160)
			n, err := io.ReadFull(dfpwmRd, frameAudio)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("juroku: EncodeVideo: audio encode error: %v", err)
			}

			output <- VideoChunk{
				Frames: frame.frames,
				Audio:  frameAudio[:n],
			}
		case err := <-errChan:
			return fmt.Errorf("juroku: EncodeVideo: pump error: %v", err)
		}
	}
}

func decodeToWorkerPump(frameRd io.Reader, inbox chan frameJob,
	outputChan chan chan framesOrError, errChan chan error) {
	defer close(inbox)
	defer close(outputChan)

	for {
		img, err := bmp.Decode(frameRd)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		} else if err != nil {
			errChan <- err
			return
		}

		frameOutput := make(chan framesOrError, 1)
		outputChan <- frameOutput

		inbox <- frameJob{
			img:    img,
			output: frameOutput,
		}
	}
}

type framesOrError struct {
	frames []*FrameChunk
	err    error
}

type frameJob struct {
	img    image.Image
	output chan<- framesOrError
}

func jurokuWorker(inbox <-chan frameJob, splitter FrameSplitter,
	speed int, dither float64) {
	for job := range inbox {
		imgs := splitter(job.img)

		var frames []*FrameChunk
		for _, img := range imgs {
			quant, err := Quantize(img, img, speed, dither)
			if err != nil {
				job.output <- framesOrError{err: err}
				return
			}

			chunked, err := ChunkImage(quant)
			if err != nil {
				job.output <- framesOrError{err: err}
				return
			}

			frame, err := GenerateFrameChunk(chunked)
			if err != nil {
				job.output <- framesOrError{err: err}
				return
			}

			frames = append(frames, frame)
		}

		job.output <- framesOrError{frames: frames}
	}
}
