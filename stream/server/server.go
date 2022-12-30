package main

import (
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tmpim/juroku"
	"github.com/tmpim/juroku/stream"

	_ "image/png"
)

var (
	upgrader = websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
	}
)

var encoderOpts = juroku.EncoderOptions{
	Width:  0,
	Height: 0,
	//Width: 286,
	//Height: 156,
	Speed:               10,
	Dither:              0.3,
	Debug:               false,
	Framerate:           20,
	GroupAudioNumFrames: 5,
	Splitter: func(img image.Image) []image.Image {
		return []image.Image{img}
		// sub := img.(interface {
		// 	SubImage(r image.Rectangle) image.Image
		// })
		// return []image.Image{
		// 	sub.SubImage(image.Rect(0, 0, 328, 201)),
		// 	sub.SubImage(image.Rect(336, 0, 664, 201)),
		// 	sub.SubImage(image.Rect(0, 210, 328, 366)),
		// 	sub.SubImage(image.Rect(336, 210, 664, 366)),
		// }
	},
}

func main() {
	width, err := strconv.Atoi(os.Getenv("JUROKU_WIDTH"))
	if err != nil {
		panic("JUROKU_WIDTH must be an integer")
	}

	height, err := strconv.Atoi(os.Getenv("JUROKU_HEIGHT"))
	if err != nil {
		panic("JUROKU_HEIGHT must be an integer")
	}

	workerCount, err := strconv.Atoi(os.Getenv("JUROKU_WORKER_COUNT"))
	if err != nil {
		panic("JUROKU_WORKER_COUNT must be an integer")
	}

	encoderOpts.Width = width
	encoderOpts.Height = height
	encoderOpts.Workers = workerCount

	encoderOpts.Debug = os.Getenv("JUROKU_DEBUG") != "" && os.Getenv("JUROKU_DEBUG") != "0"
	encoderOpts.Verbose = os.Getenv("JUROKU_VERBOSE") != "" && os.Getenv("JUROKU_VERBOSE") != "0"
	encoderOpts.Realtime = os.Getenv("JUROKU_REALTIME") != "" && os.Getenv("JUROKU_REALTIME") != "0"
	encoderOpts.AudioEncoder = &juroku.PCMEncoder{Verbose: encoderOpts.Verbose}

	log.Printf("encoder options: %+v", encoderOpts)

	mgr := stream.NewStreamManager(encoderOpts)

	e := echo.New()

	e.Use(middleware.Logger())

	pprof.Register(e)

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	api := e.Group("/api")

	api.GET("/client", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		mgr.HandleConn(ws)

		return nil
	})

	api.POST("/pause", func(c echo.Context) error {
		state, err := mgr.Pause()
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, &state)
	})

	api.POST("/resume", func(c echo.Context) error {
		state, err := mgr.Resume()
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, &state)
	})

	api.POST("/stop", func(c echo.Context) error {
		state, err := mgr.Stop()
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, &state)
	})

	api.POST("/play/file", func(c echo.Context) error {
		data, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}

		path := strings.TrimSpace(string(data))

		log.Println("juroku stream: playing file:", path)

		err = mgr.PlayFile(path)
		if err != nil {
			return err
		}

		state := mgr.State()
		return c.JSON(http.StatusOK, &state)
	})

	api.POST("/play/url", func(c echo.Context) error {
		data, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}

		playURL := strings.TrimSpace(string(data))

		log.Println("juroku stream: playing URL:", playURL)

		err = mgr.PlayURL(playURL)
		if err != nil {
			return err
		}

		state := mgr.State()
		return c.JSON(http.StatusOK, &state)
	})

	api.POST("/play/stream", func(c echo.Context) error {
		data, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}

		playURL := strings.TrimSpace(string(data))

		log.Println("juroku stream: playing stream URL:", playURL)

		err = mgr.PlayStream(playURL)
		if err != nil {
			return err
		}

		state := mgr.State()
		return c.JSON(http.StatusOK, &state)
	})

	log.Fatal(e.Start(":4600"))
}
