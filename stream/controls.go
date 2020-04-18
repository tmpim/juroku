package stream

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/tmpim/juroku"
)

const (
	stateTimeout = 3 * time.Second
)

type dummyReadCloser struct{}

func (d dummyReadCloser) Close() error {
	return nil
}

func (d dummyReadCloser) Read(data []byte) (int, error) {
	return 0, io.EOF
}

func (s *StreamManager) PlaySource(meta *Metadata, rawInput interface{},
	ctx context.Context, cancel func()) error {
	var input io.ReadCloser
	switch v := rawInput.(type) {
	case io.ReadCloser:
		input = v
	case string:
		input = dummyReadCloser{}
	default:
		return errors.New("juroku stream: bad input type")
	}

	if !s.UpdateState(State{
		Title:         meta.Title,
		State:         StateTransitioning,
		Timestamp:     0,
		RelativeStart: time.Now(),
		Duration:      meta.Duration,
		Context:       ctx,
		Cancel:        cancel,
	}, []int{StateStopped}) {
		cancel()
		input.Close()
		return errors.New("juroku stream: play: player must be stopped to play")
	}

	opts := s.baseOptions
	opts.Context = ctx
	opts.Realtime = true

	output := make(chan juroku.VideoChunk, juroku.Framerate)
	go func() {
		defer input.Close()
		log.Println("juroku stream: play: encode video started")
		err := juroku.EncodeVideo(rawInput, output, opts)
		if err != nil {
			log.Println("juroku stream: play: encode ended:", err)
		}
	}()

	syncedOutput := make(chan juroku.VideoChunk, juroku.Framerate)

	go func() {
		defer log.Println("juroku stream: play: output syncer quitting")
		defer close(syncedOutput)

		t := time.NewTicker(time.Second / juroku.Framerate)
		defer t.Stop()

		for range t.C {
			if len(output) >= minBufferSize {
				break
			}
		}

		t.Stop()

		var videoQueue [][]*juroku.FrameChunk

		for frame := range output {
			if len(videoQueue) < 60 {
				videoQueue = append(videoQueue, frame.Frames)
				syncedOutput <- juroku.VideoChunk{
					Audio: frame.Audio,
				}

				continue
			}

			syncedOutput <- juroku.VideoChunk{
				Frames: videoQueue[0],
				Audio:  frame.Audio,
			}

			videoQueue = append(videoQueue[1:], frame.Frames)
		}

		for _, videoFrame := range videoQueue {
			syncedOutput <- juroku.VideoChunk{
				Frames: videoFrame,
			}
		}
	}()

	atomic.StoreUint32(&s.targetState, StatePlaying)
	statePlaying := make(chan error, 2)

	go func() {
		defer log.Println("juroku stream: play: video outputter quitting")
		defer cancel()
		defer close(statePlaying)
		defer func() {
			log.Println("juroku stream: play: playback has stopped")
			if !s.UpdateState(State{
				Title:         "",
				State:         StateStopped,
				Timestamp:     0,
				RelativeStart: time.Now(),
				Duration:      0,
			}, []int{StateTransitioning, StatePlaying, StatePaused}) {
				log.Println("juroku stream: already stopped or something")
			}
		}()

		t := time.NewTicker(time.Second / juroku.Framerate)
		defer t.Stop()
		for range t.C {
			if len(syncedOutput) >= minBufferSize {
				break
			}
		}

		buf := new(bytes.Buffer)

		frameCount := 0
		hasPaused := false

		// debugRd, debugWr := io.Pipe()

		// dec := dfpwm.NewDecoder(debugRd, 48000)
		// debugSink := ffplay.NewFFPlaySink()
		// go debugSink.PlayStream(dec)

		for range t.C {
			targetState := atomic.LoadUint32(&s.targetState)
			if !hasPaused && targetState == StatePaused {
				state := NewEmptyState()
				state.Timestamp = time.Duration(frameCount) * (time.Second / juroku.Framerate)
				state.State = StatePaused
				state.RelativeStart = time.Now()
				if !s.UpdateState(state, []int{StatePlaying, StatePaused}) {
					log.Println("juroku stream: play: state inconsistency????")
				}
				hasPaused = true
				continue
			} else if hasPaused && targetState == StatePlaying && frameCount > 0 {
				state := NewEmptyState()
				duration := time.Duration(frameCount) * (time.Second / juroku.Framerate)
				state.State = StatePlaying
				state.RelativeStart = time.Now().Add(-duration)
				if !s.UpdateState(state, []int{StatePlaying, StatePaused}) {
					log.Println("juroku stream: play: state inconsistency????")
				}
				hasPaused = false
			} else if targetState == StateStopped {
				go func() {
					// because we're in an early abort, we need to drain the output channel
					for range syncedOutput {
					}
				}()
				break
			} else if hasPaused {
				continue
			}

			frame, more := <-syncedOutput
			if !more {
				break
			}

			if frameCount == 0 {
				state := NewEmptyState()
				state.RelativeStart = time.Now()
				state.State = StatePlaying
				if !s.UpdateState(state, []int{StateTransitioning}) {
					log.Println("juroku stream: play: state inconsistency, expected transitioning")
					statePlaying <- errors.New("juroku stream: play: state inconsistency, expected transitioning")
					return
				}

				statePlaying <- nil
			} else {
				currentState := s.State()
				expectedElapsed := time.Duration(frameCount) * (time.Second / juroku.Framerate)
				diffElapsed := math.Abs(time.Since(currentState.RelativeStart).Seconds() - expectedElapsed.Seconds())
				if diffElapsed > 1 {
					state := NewEmptyState()
					state.RelativeStart = time.Now().Add(-expectedElapsed)
					log.Println("juroku stream: play: drift > 1 second detected")
					s.UpdateState(state, []int{StatePlaying})
				}
			}

			if len(frame.Audio) > 0 {
				// go debugWr.Write([]byte(frame.Audio))
				log.Println("sending audio:", len(frame.Audio))
				buf.Reset()
				buf.WriteByte(PacketAudio)
				s.Broadcast(SubscriptionAudio, append([]byte{PacketAudio}, []byte(frame.Audio)...))
			}

			for i, subFrame := range frame.Frames {
				buf.Reset()
				buf.WriteByte(PacketVideo)
				buf.WriteByte(byte(i))
				subFrame.WriteTo(buf)
				s.Broadcast(SubscriptionVideo, buf.Bytes())
			}

			frameCount++
		}
	}()

	err, ok := <-statePlaying
	if !ok {
		return errors.New("juroku stream: play: failed to load first frame")
	}

	return err
}

func (s *StreamManager) PlayFile(path string) error {
	// meta, err := FileSource(path)
	// if err != nil {
	// 	return err
	// }

	// ctx, cancel := context.WithCancel(context.Background())

	// return s.PlaySource(meta, path, ctx, cancel)

	return s.PlaySource(&Metadata{}, path, context.Background(), func() {})
}

func (s *StreamManager) PlayURL(videoURL string) error {
	ctx, cancel := context.WithCancel(context.Background())
	meta, rd, err := YoutubeDLSource(videoURL, ctx)
	if err != nil {
		cancel()
		return err
	}

	return s.PlaySource(meta, rd, ctx, cancel)
}

func (s *StreamManager) PlayStream(streamURL string) error {
	ctx, cancel := context.WithCancel(context.Background())
	meta, rd, err := StreamlinkSource(streamURL, ctx)
	if err != nil {
		cancel()
		return err
	}

	return s.PlaySource(meta, rd, ctx, cancel)
}

func (s *StreamManager) waitForStateDefaultTimeout(state int) (State, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(stateTimeout))
	defer cancel()

	finalState, ok := s.WaitForState(state, ctx)
	if !ok {
		return State{}, errors.New("juroku video: timeout waiting for desired state")
	}

	return finalState, nil
}

// Resume casually attempts to resume playback of paused media.
func (s *StreamManager) Resume() (State, error) {
	state := s.State()
	if state.State != StatePaused {
		return State{}, errors.New("juroku stream: resume: current state must be paused to resume")
	}

	atomic.StoreUint32(&s.targetState, StatePlaying)

	return s.waitForStateDefaultTimeout(StatePlaying)
}

// Pause casually attempts to pause playback of playing media.
func (s *StreamManager) Pause() (State, error) {
	state := s.State()
	if state.State != StatePlaying {
		return State{}, errors.New("juroku stream: pause: current state must be playing to pause")
	}

	atomic.StoreUint32(&s.targetState, StatePaused)

	return s.waitForStateDefaultTimeout(StatePaused)
}

// Stop stops the playing/paused media
func (s *StreamManager) Stop() (State, error) {
	state := s.State()

	validState := state.State != StateStopped
	validCtx := state.Context != nil && state.Context.Err() == nil && state.Cancel != nil

	if !validState || !validCtx {
		return State{}, errors.New("juroku stream: not in a valid state to stop")
	}

	state.Cancel()
	atomic.StoreUint32(&s.targetState, StateStopped)

	return s.waitForStateDefaultTimeout(StateStopped)
}

func (s *StreamManager) State() State {
	s.stateCond.L.Lock()
	defer s.stateCond.L.Unlock()

	return s.state
}
