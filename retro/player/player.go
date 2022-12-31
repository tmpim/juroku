package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/1lann/dissonance/audio"
	"github.com/1lann/dissonance/filters/samplerate"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tmpim/juroku"
	"github.com/tmpim/juroku/retro"
	"golang.org/x/sync/errgroup"

	_ "image/png"
)

const (
	framerate = 20
)

type connWithMutex struct {
	*websocket.Conn
	mu sync.Mutex
}

var (
	upgrader = websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
	}

	videoConnMutex = new(sync.Mutex)
	videoConns     []*connWithMutex

	audioConnMutex = new(sync.Mutex)
	audioConns     []*websocket.Conn
)

func main() {
	log.Println("starting juroku retro")

	var libretroPath string

	err := filepath.WalkDir("/usr/lib", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if libretroPath != "" {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Base(path) == "mgba_libretro.so" {
			libretroPath = path
			return filepath.SkipDir
		}

		return nil
	})

	if libretroPath == "" {
		if err != nil {
			panic(fmt.Sprintf("finding mgba_libretro.so: %+v\n", err))
		}
		panic("could not find mgba_libretro.so")
	}

	core, err := retro.NewCore(libretroPath, &retro.Options{
		Username:  os.Getenv("JUROKU_USERNAME"),
		SystemDir: os.Getenv("JUROKU_SYSTEM_DIR"),
		SaveDir:   os.Getenv("JUROKU_SAVE_DIR"),
	})
	if err != nil {
		panic(err)
	}

	e := echo.New()

	e.Use(middleware.Logger())
	pprof.Register(e)

	e.GET("/audio", func(c echo.Context) error {
		return c.HTML(http.StatusOK, audioHTML)
	})

	api := e.Group("/api")

	api.GET("/ws/audio", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		sampleRate := core.AVInfo().Timing.SampleRate

		ws.WriteMessage(websocket.TextMessage,
			[]byte(strconv.FormatFloat(sampleRate, 'f', -1, 64)))

		audioConnMutex.Lock()
		audioConns = append(audioConns, ws)

		defer func() {
			audioConnMutex.Lock()
			for i, conn := range audioConns {
				if conn == ws {
					audioConns[i] = audioConns[len(audioConns)-1]
					audioConns = audioConns[:len(audioConns)-1]
					break
				}
			}
			audioConnMutex.Unlock()
		}()

		audioConnMutex.Unlock()

		for err == nil {
			_, _, err = ws.ReadMessage()
		}

		return nil
	})

	api.GET("/ws/video", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		videoConnMutex.Lock()
		videoConns = append(videoConns, &connWithMutex{
			Conn: ws,
		})

		defer func() {
			videoConnMutex.Lock()
			for i, conn := range videoConns {
				if conn.Conn == ws {
					videoConns[i] = videoConns[len(videoConns)-1]
					videoConns = videoConns[:len(videoConns)-1]
					break
				}
			}
			videoConnMutex.Unlock()
		}()

		videoConnMutex.Unlock()

		for err == nil {
			_, _, err = ws.ReadMessage()
		}

		return nil
	})

	api.GET("/ws/control/:id", func(c echo.Context) error {
		deviceID, err := strconv.Atoi(c.Param("id"))
		if err != nil || deviceID < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "device ID must be a positive integer")
		}

		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		device := core.ConnectDevice(deviceID)

		for {
			typ, data, err := ws.ReadMessage()
			if err != nil {
				break
			}

			if typ != websocket.TextMessage {
				continue
			}

			var keyID, keyState int
			n, err := fmt.Sscanf(string(data), "%d %d", &keyID, &keyState)
			if err != nil || n != 2 {
				log.Println("juroku retro: /ws/control: malformed message:", string(data))
				continue
			}

			device.Set(keyID, int16(keyState))
		}

		return nil
	})

	api.GET("/state/save/:name", func(c echo.Context) error {
		if err := core.Save(c.Param("name")); err != nil {
			return err
		}

		return c.String(http.StatusOK, "state saved")
	})

	api.GET("/state/load/:name", func(c echo.Context) error {
		if err := core.Load(c.Param("name")); err != nil {
			return err
		}

		return c.String(http.StatusOK, "state loaded")
	})

	fastChunker := &juroku.FastChunker{}
	frameSeqer := make(chan struct{}, 1)
	frameSeqer <- struct{}{}
	audioSeqer := make(chan struct{}, 1)
	audioSeqer <- struct{}{}
	ccAudioSeqer := make(chan struct{}, 1)
	ccAudioSeqer <- struct{}{}
	lastFrame := time.Now()
	minDelay := (time.Second / time.Duration(framerate)) - (5 * time.Millisecond)
	// useDebug := os.Getenv("JUROKU_DEBUG") != "" && os.Getenv("JUROKU_DEBUG") != "0"
	audioGroupFrames, err := strconv.Atoi(os.Getenv("JUROKU_AUDIO_FRAMES"))
	if err != nil {
		panic("JUROKU_AUDIO_FRAMES must be an integer")
	}

	core.OnFrameDraw(func(img image.Image) {
		if time.Since(lastFrame) < minDelay {
			return
		}

		lastFrame = time.Now()

		go func() {
			out, palette, err := juroku.Quantize(img, img, 10, 0.3)
			if err != nil {
				panic(err)
			}

			frame, err := fastChunker.ChunkImage(out, palette)
			if err != nil {
				panic(err)
			}

			buf := new(bytes.Buffer)
			frame.WriteTo(buf)

			videoConnMutex.Lock()
			connState := make([]*connWithMutex, len(videoConns))
			copy(connState, videoConns)
			videoConnMutex.Unlock()

			<-frameSeqer
			defer func() {
				frameSeqer <- struct{}{}
			}()
			for _, conn := range connState {
				conn.mu.Lock()
				err := conn.WriteMessage(websocket.BinaryMessage, append([]byte{1}, buf.Bytes()...))
				conn.mu.Unlock()
				if err != nil {
					conn.Close()
				}
			}
		}()
	})

	var inputStream *audio.OfflineStream
	var bootComplete atomic.Bool

	core.OnAudioBuffer(1024, func(data []int16) {
		if !bootComplete.Load() {
			return
		}
		go func() {
			byteData := make([]byte, len(data)*2)
			for i, d := range data {
				byteData[i*2] = byte(d & 0xff)
				byteData[i*2+1] = byte(d >> 8)
			}

			monoAverage := make([]int16, len(data)/2)
			for i := 0; i < len(data); i += 2 {
				monoAverage[i/2] = (data[i] + data[i+1]) / 2
			}

			audioConnMutex.Lock()
			connState := make([]*websocket.Conn, len(audioConns))
			copy(connState, audioConns)
			audioConnMutex.Unlock()

			<-audioSeqer
			defer func() {
				audioSeqer <- struct{}{}
			}()

			for _, conn := range connState {
				err := conn.WriteMessage(websocket.BinaryMessage, byteData)
				if err != nil {
					conn.Close()
				}
			}

			if err := inputStream.WriteValues(monoAverage); err != nil {
				log.Printf("failed to write to input audio stream: %v", err)
			}
		}()
	})

	core.LoadGame(os.Args[1])

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eg, ctx := errgroup.WithContext(rootCtx)

	eg.Go(func() error {
		core.Run()
		return errors.New("core exited")
	})

	sysSampleRate := core.AVInfo().Timing.SampleRate
	inputStream = audio.NewOfflineStream(int(sysSampleRate), 512)

	encoder := new(juroku.PCMEncoder)
	audioEncRd, audioEncWr := io.Pipe()

	eg.Go(func() error {
		sampleRateFilter := samplerate.NewFilter(encoder.SampleRate()).Filter(inputStream)
		log.Println("output sample rate:", sampleRateFilter.SampleRate())
		return encoder.Encode(sampleRateFilter, audioEncWr, juroku.EncoderOptions{})
	})

	eg.Go(func() error {
		frameAudio := make([]byte, (encoder.SampleRateBytes()/framerate)*audioGroupFrames)
		for {
			_, err := io.ReadFull(audioEncRd, frameAudio)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("juroku: EncodeVideo: audio encode error: %v", err)
			}

			videoConnMutex.Lock()
			connState := make([]*connWithMutex, len(videoConns))
			copy(connState, videoConns)
			videoConnMutex.Unlock()

			go func() {
				<-ccAudioSeqer
				defer func() {
					ccAudioSeqer <- struct{}{}
				}()

				for _, conn := range connState {
					conn.mu.Lock()
					err := conn.WriteMessage(websocket.BinaryMessage, append([]byte{2}, frameAudio...))
					conn.mu.Unlock()
					if err != nil {
						conn.Close()
					}
				}
			}()
		}
	})

	e.GET("/healthz", func(c echo.Context) error {
		if ctx.Err() != nil {
			log.Println("healthz: context error (shutting donw?):", ctx.Err())
			return c.NoContent(http.StatusServiceUnavailable)
		}
		if !bootComplete.Load() {
			log.Println("healthz: boot not complete")
			return c.NoContent(http.StatusServiceUnavailable)
		}

		return c.NoContent(http.StatusOK)
	})

	eg.Go(func() error {
		return e.Start(":4600")
	})

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		e.Shutdown(ctx)
	}()

	log.Println("sample rate:", core.AVInfo().Timing.SampleRate)
	bootComplete.Store(true)

	log.Println(eg.Wait())
}
