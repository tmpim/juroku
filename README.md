# Juroku

Juroku converts an image (PNG or JPG) into a Lua script that can be loaded as a
ComputerCraft API to be used to draw on terminals and monitors.

## Building and Running
1. A cgo security exception is required to allow the libimagequant to build:
```
export CGO_CFLAGS_ALLOW=.*
export CGO_LDFLAGS_ALLOW=.*
```
2. `go get -u github.com/1lann/juroku/cmd/juroku`
3. Find `juroku` in your `$GOPATH/bin` and run it.

TODO: Release pre-built binaries.

## Usage:
```
Usage: juroku [options] input_image

Juroku converts an image (PNG or JPG) into a Lua script that can be
loaded as a ComputerCraft API to be used to draw on terminals and monitors.
Images are not automatically downscaled or cropped.

input_image must have a height that is a multiple of 3 in pixels,
and a width that is a multiple of 2 in pixels.

Options:
  -d float
    	set the amount of allowed dithering (0 = none, 1 = most) (default 0.2)
  -license
    	show GPLv3 licensing disclaimer
  -o string
    	set location of output script (default "image.lua")
  -p string
    	set location of output preview (will be PNG) (default "preview.png")
  -q int
    	set the processing speed/quality (1 = slowest, 10 = fastest) (default 1)
```

## Disclaimer

Juroku when built contains libimagequant which is licensed under GPLv3. For more
information and the source code of the relevant portions, see: https://github.com/1lann/imagequant.
