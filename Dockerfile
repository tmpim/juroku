# syntax=docker/dockerfile:1
FROM golang:1.19-buster AS build

RUN apt-get update && apt-get install -y  \
	ca-certificates \
	unzip \
	xz-utils \
	git \
	curl \
	make \
	cmake \
	gcc-8 \
	g++ \
	pkg-config \
	libgcc-8-dev \
	&& apt-get clean

ENV GOPATH /go
ENV GOPROXY=https://proxy.golang.org,direct
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
ENV CGO_ENABLED=1

WORKDIR /workdir

COPY go.mod /workdir/go.mod
COPY go.sum /workdir/go.sum

RUN go mod download

COPY . /workdir

RUN git config --global --add safe.directory /workdir
RUN CGO_CFLAGS_ALLOW=".*" CGO_LDFLAGS_ALLOW=".*" go build -ldflags '-extldflags "-fno-PIC -static -lm"' -buildmode pie -tags 'osusergo netgo static_build' -o juroku ./stream/server

FROM alpine:3

RUN apk add ca-certificates ffmpeg python3 curl
RUN curl --show-error --fail -L https://github.com/yt-dlp/yt-dlp/releases/download/2022.11.11/yt-dlp -o /usr/local/bin/yt-dlp && chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /

COPY --from=build /workdir/juroku /juroku

EXPOSE 4600
ENTRYPOINT ["/juroku"]
