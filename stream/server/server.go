package main

import (
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
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
	Width:  328,
	Height: 201,
	//Width: 286,
	//Height: 156,
	Realtime:            true,
	Workers:             6,
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
	AudioEncoder: new(juroku.PCMEncoder),
}

func main() {
	mgr := stream.NewStreamManager(encoderOpts)

	e := echo.New()

	e.Use(middleware.Logger())

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

	log.Fatal(e.Start(":9999"))
}
