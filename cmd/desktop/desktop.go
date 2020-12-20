package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("specify a bash script to run")
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9999"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// cmd := exec.Command("ffmpeg.exe", "-f", "gdigrab", "-framerate", "10", "-i", "desktop", "-filter:v",
		// 	"crop=2560:1440:4178:390,scale=-1:720", "-preset", "ultrafast", "-vcodec", "libx264", "-tune", "zerolatency", "-b:v", "3000k", "-f", "mpegts", "-")
		cmd := exec.Command("bash", os.Args[1])

		rd, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			panic(err)
		}

		err = cmd.Start()
		if err != nil {
			panic(err)
		}

		go io.Copy(os.Stderr, stderr)

		defer cmd.Process.Kill()

		w.Header().Set("Content-Type", "video/MP2T")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, rd)
	})

	log.Println("will be listening on port:", port)
	http.ListenAndServe(port, nil)
}
