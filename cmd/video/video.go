package main

import (
	"bytes"
	"context"
	"image"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/tmpim/juroku"

	_ "image/png"
)

const (
	minBufferSize = 2
)

var (
	upgrader = websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
	}
)

func main() {
	e := echo.New()
	// e.Logger.SetOutput(ioutil.Discard)
	e.GET("/ws", onWS)
	log.Fatal(e.Start(":9999"))
}

func onWS(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	file, err := os.Open("./input.mkv")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	output := make(chan juroku.VideoChunk, 10)

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))

	go func() {
		err := juroku.EncodeVideo(file, output, juroku.EncoderOptions{
			Context: ctx,
			Width:   665,
			Height:  366,
			// Width:       328,
			// Height:      177,
			Realtime: true,
			Workers:  6,
			Speed:    10,
			Dither:   0.2,
			Debug:    true,
			Splitter: func(img image.Image) []image.Image {
				sub := img.(interface {
					SubImage(r image.Rectangle) image.Image
				})
				return []image.Image{
					sub.SubImage(image.Rectangle{
						Min: image.Point{
							X: 0,
							Y: 0,
						},
						Max: image.Point{
							X: 328,
							Y: 201,
						},
					}),
					sub.SubImage(image.Rectangle{
						Min: image.Point{
							X: 336,
							Y: 0,
						},
						Max: image.Point{
							X: 664,
							Y: 201,
						},
					}),
					sub.SubImage(image.Rectangle{
						Min: image.Point{
							X: 0,
							Y: 210,
						},
						Max: image.Point{
							X: 328,
							Y: 366,
						},
					}),
					sub.SubImage(image.Rectangle{
						Min: image.Point{
							X: 336,
							Y: 210,
						},
						Max: image.Point{
							X: 664,
							Y: 366,
						},
					}),
				}
			},
		})

		if err != nil {
			panic(err)
		}
	}()

	t := time.NewTicker(time.Second / 10)
	defer t.Stop()
	for range t.C {
		if len(output) >= minBufferSize {
			break
		}
	}

	buf := new(bytes.Buffer)

	log.Println("rendering!")
	// lastTime := time.Now()
	for range t.C {
		frame, more := <-output
		if !more {
			break
		}

		// log.Println(time.Since(lastTime))
		// lastTime = time.Now()

		for _, subFrame := range frame.Frames {
			buf.Reset()
			subFrame.WriteTo(buf)
			ws.WriteMessage(websocket.BinaryMessage, buf.Bytes())
		}
	}

	log.Println("render complete!")

	return nil
}
