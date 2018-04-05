# Juroku File Format (JUF) v1
Juroku makes use of a custom streamable multimedia file format for pictures,
videos (with and without audio), and just audio.

## Format
```
+-----------------+------------------+---------------+
| Magic (3 bytes) | Version (1 byte) | Type (1 byte) |
+-----------------+------------------+---------------+
+---------+---------+-----+ - - - - - - - - - - - -+
| Chunk 0 | Chunk 1 | ... | EOF (Optional 8 bytes) |
+---------+---------+-----+ - - - - - - - - - - - -+
```
The format consists of a global header, followed by chunks, somewhat similar
to many other formats such as PNG.

The global header MUST consist of the magic number: 0x4a 0x55 0x46, which is "JUF"
encoded as ASCII, followed by the version (specification) number of the JUF file
and the data type of the JUF file. The version and type numbers are encoded
as single byte binary integers.

JUF files MAY NOT have backwards compatability,
and could become incompatible between versions at any time.

Chunks in JUF do not have a standardised header, they are dependent
on the context of the chunk, the version of JUF, and the type of the JUF file.
This is explained below.

All integers in JUF MUST be encoded in unsigned big-endian binary form.

### EOF
It is RECOMMENDED to use EOF for indication of the end of a stream, or a
simulated EOF. If this is
not possible, 8 zero bytes (`0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`)
MAY be used instead. All chunk decoders SHOULD detect this and terminate if an
EOF is present, although there are some exceptions in the rule detailed below.

## Version
This specification details version 1 of the JUF format, thus in order to fufil
this specification, the first byte of a JUF file SHALL be 1.

## Types
The following types are supported, and the type number SHALL be one of these
values:
- `1`: static image
- `2`: video with audio
- `3`: video without audio
- `4`: only audio

## Frame Chunk
```
Overall structure of a frame chunk:
+------------------+-----------------------------------+--------------------+
| Header (4 bytes) | Pixels (width * height * 2 bytes) | Palette (48 bytes) |
+------------------+-----------------------------------+--------------------+
```
A frame chunk is the what the video data will primarily consist of, and is also
what is used to represent a single static image. This
contains the height and width of the frame,
character, background color, text color, and palette that is used for a
single frame.

For JUF types `2` and `4`, a single frame chunk MUST be present after an
audio chunk with the exception of the final audio chunk.

### Header
```
+-------------- 4 bytes -------------+
+-----------------+------------------+
| Width (2 bytes) | Height (2 bytes) |
+-----------------+------------------+`
```
A frame chunk's header consists of the width, followed by the width of the
frame, each encoded as an integer.

In order to support the fallback approach for EOF detailed above, if the width
and height are equal to 0, decoding MUST terminate as the stream has ended.

### Pixel
```
+------------------------------ 2 bytes -------------------------------+------------- ... ---------------+
+---------------------+---------------------------+--------------------+
| Text color (4 bits) | Background color (4 bits) | Character (1 byte) | Repeats width * height times...
+---------------------+---------------------------+--------------------+
```
Then follows `width * height` pixel parts, representing the character
that should be drawn in order from the top left of the frame to the bottom
right of the frame. That is, the first `width` pixel parts represent the first
row of characters that should be drawn, and is repeated `height` times.
Each pixel part contains the text color and background color encoded in 4 bits,
in a single byte, followed by the raw ASCII character that should be drawn at
that location.

### Palette
```
+---------------------------- 3 bytes ----------------------------------+--------- ...--------+
+----------------------+------------------------+-----------------------+
| Red channel (1 byte) | Green channel (1 byte) | Blue channel (1 byte) | Repeats 16 times...
+----------------------+------------------------+-----------------------+
```
After the pixel parts come the color palette, which MUST consist of 16, 24-bit
RGB colors. The color palette is in order of the colors specified in the frame.
That is, the first RGB color in the palette is color `0` used in the pixel part,
the second is `1`, etc..

## Audio Chunk
```
+----------------+-------------------------+
| Size (4 bytes) | DFPWM Data (size bytes) |
+----------------+-------------------------+
```
An audio chunk represents a portion of audio data, and occurs in a number of
scenarios, namely ONLY if the JUF type is `2` or `4`, all other types
MUST not have any audio chunks.

If the JUF type is `2` or `4`, there MUST be an initial audio chunk
that occurs after the global header. It is RECOMMENDED that the initial
length is 1 minute worth of samples (2,880,000 samples or 360 kB).
This is because Computronics tapes are played in a streaming manner by
swapping between tapes and require a short 0.2s audio glitch when switching,
and 1 minute is an ideal length to swap between tapes. Increasing this value
will decrease the frequency of glitches but increases the required streaming
buffer.

To clarify, that means the audio chunks after frames are ahead by the initial
buffer's worth of samples compared to the frame chunks they are adjacent to.
This is intentional to allow for the audio data to buffer into a tape.

Also when the JUF type is `2` or `4`, there MUST be an audio chunk at the
end of every frame chunk, with the exception of the initial audio chunk,
representing the audio for that frame (0.2 seconds) and SHOULD be 300 bytes,
with the exception of the last buffer's worth of audio which SHOULD be 0 bytes
in size.

Note that the size for an audio chunk is in bytes and not in samples, the
total number of samples can thus be calculated by multiplying the number
of bytes by 8.

For JUF type `4`, in order to support the fallback approach for EOF detailed
above, if the size is equal to 0, decoding MUST terminate as the stream has
ended. JUF type `2` is exempt from this because 0 sized audio chunks
are necessary for the final buffer's worth of audio, as the audio stream
is always ahead of the frame. Instead, the 4 byte EOF MUST be detected in
the frame chunks.

### Type `4` audio only files
Audio only files MUST only be composed of audio chunks, no frame chunks
may be present. Although unnecessary, it is permitted to split the audio
into multiple chunks in order to fit the chunks appropriately in
packets such as for WebSocket or HTTP streaming in ComputerCraft.
