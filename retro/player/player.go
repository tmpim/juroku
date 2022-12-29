package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tmpim/juroku"
	"github.com/tmpim/juroku/retro"

	_ "image/png"
)

const (
	framerate = 10
)

var (
	upgrader = websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
	}

	videoConnMutex = new(sync.Mutex)
	videoConns     []*websocket.Conn

	audioConnMutex = new(sync.Mutex)
	audioConns     []*websocket.Conn
)

func main() {
	log.Println("starting juroku retro")
	core, err := retro.NewCore("./mgba_libretro.so", &retro.Options{
		Username:  "1lann",
		SystemDir: "./system",
		SaveDir:   "./saves",
	})
	if err != nil {
		panic(err)
	}

	e := echo.New()

	e.Use(middleware.Logger())

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
		videoConns = append(videoConns, ws)

		defer func() {
			videoConnMutex.Lock()
			for i, conn := range videoConns {
				if conn == ws {
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
	lastFrame := time.Now()
	minDelay := (time.Second / time.Duration(framerate)) - (5 * time.Millisecond)

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
			connState := make([]*websocket.Conn, len(videoConns))
			copy(connState, videoConns)
			videoConnMutex.Unlock()

			<-frameSeqer
			defer func() {
				frameSeqer <- struct{}{}
			}()
			for _, conn := range connState {
				err := conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
				if err != nil {
					conn.Close()
				}
			}
		}()
	})

	core.OnAudioBuffer(1024, func(data []int16) {
		go func() {
			byteData := make([]byte, len(data)*2)
			for i, d := range data {
				byteData[i*2] = byte(d & 0xff)
				byteData[i*2+1] = byte(d >> 8)
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
		}()
	})

	core.LoadGame(os.Args[1])

	go core.Run()

	log.Println("sample rate:", core.AVInfo().Timing.SampleRate)

	log.Fatal(e.Start(":9999"))
}
