# Juroku

Juroku converts an image (PNG or JPG) into a Lua script that can be loaded as a
ComputerCraft API to be used to draw on terminals and monitors.

## Building and Running
1. You have to be using gcc 8 or 7 (even older versions may work as well, just untested). Juroku will not build on gcc 9+. If you see errors like
```
# github.com/1lann/imagequant
kmeans.c: In function ‘kmeans_do_iteration’:
kmeans.c:90:13: error: ‘hist_size’ not specified in enclosing ‘parallel’
   90 |     #pragma omp parallel for if (hist_size > 3000) \
      |             ^~~
kmeans.c:90:13: error: enclosing ‘parallel’
kmeans.c:94:69: error: ‘achv’ not specified in enclosing ‘parallel’
   94 |         unsigned int match = nearest_search(n, &achv[j].acolor, achv[j].tmp.likely_colormap_index, &diff);
      |                                                                     ^
kmeans.c:90:13: error: enclosing ‘parallel’
   90 |     #pragma omp parallel for if (hist_size > 3000) \
      |             ^~~
kmeans.c:94:30: error: ‘n’ not specified in enclosing ‘parallel’
   94 |         unsigned int match = nearest_search(n, &achv[j].acolor, achv[j].tmp.likely_colormap_index, &diff);
      |                              ^~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
kmeans.c:90:13: error: enclosing ‘parallel’
   90 |     #pragma omp parallel for if (hist_size > 3000) \
      |             ^~~
kmeans.c:98:9: error: ‘map’ not specified in enclosing ‘parallel’
   98 |         kmeans_update_color(achv[j].acolor, achv[j].perceptual_weight, map, match, omp_get_thread_num(), average_color);
      |         ^~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
kmeans.c:90:13: error: enclosing ‘parallel’
   90 |     #pragma omp parallel for if (hist_size > 3000) \
      |             ^~~
```
You are using the wrong version of gcc.

2. A cgo security exception is required to allow the libimagequant to build:
```
export CGO_CFLAGS_ALLOW=.*
export CGO_LDFLAGS_ALLOW=.*
```
3. `go get -u github.com/tmpim/juroku/cmd/juroku` (or if you want to specify a specific C compiler, you can do `CC="gcc-8" go get ...`)
4. Find `juroku` in your `$GOPATH/bin` and run it.

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
