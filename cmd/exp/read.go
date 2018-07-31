package main

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
)

func main() {
	out, err := os.Create("./out.aud")
	if err != nil {
		panic(err)
	}

	file, err := os.Open("./shelter.juf")
	if err != nil {
		panic(err)
	}

	io.CopyN(ioutil.Discard, file, 5)
	readAudio(out, file)

	for readFrame(file) {
		readAudio(out, file)
	}

	out.Close()
}

func readFrame(in io.Reader) bool {
	var width uint16
	var height uint16
	err := binary.Read(in, binary.BigEndian, &width)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return false
	} else if err != nil {
		panic(err)
	}
	err = binary.Read(in, binary.BigEndian, &height)
	if err != nil {
		panic(err)
	}

	io.CopyN(ioutil.Discard, in, int64(width)*int64(height)*2+16*3)

	return true
}

func readAudio(out io.Writer, in io.Reader) {
	var size uint32
	err := binary.Read(in, binary.BigEndian, &size)
	if err != nil {
		panic(err)
	}

	_, err = io.CopyN(out, in, int64(size))
	if err != nil {
		panic(err)
	}
}
