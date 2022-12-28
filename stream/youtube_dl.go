package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"time"
)

type YouTubeDLMetadata struct {
	ID          string        `json:"id"`
	Uploader    string        `json:"uploader"`
	UploaderID  string        `json:"uploader_id"`
	UploaderURL string        `json:"uploader_url"`
	ChannelID   string        `json:"channel_id"`
	ChannelURL  string        `json:"channel_url"`
	UploadDate  string        `json:"upload_date"`
	License     interface{}   `json:"license"`
	Creator     string        `json:"creator"`
	Title       string        `json:"title"`
	AltTitle    string        `json:"alt_title"`
	Thumbnail   string        `json:"thumbnail"`
	Description string        `json:"description"`
	Categories  []string      `json:"categories"`
	Tags        []interface{} `json:"tags"`
	Subtitles   struct {
	} `json:"subtitles"`
	AutomaticCaptions struct {
	} `json:"automatic_captions"`
	Duration      float64     `json:"duration"`
	AgeLimit      int         `json:"age_limit"`
	Annotations   interface{} `json:"annotations"`
	Chapters      interface{} `json:"chapters"`
	WebpageURL    string      `json:"webpage_url"`
	ViewCount     int         `json:"view_count"`
	LikeCount     int         `json:"like_count"`
	DislikeCount  int         `json:"dislike_count"`
	AverageRating float64     `json:"average_rating"`
	Formats       []struct {
		FormatID          string      `json:"format_id"`
		URL               string      `json:"url"`
		PlayerURL         string      `json:"player_url"`
		Ext               string      `json:"ext"`
		FormatNote        string      `json:"format_note"`
		Acodec            string      `json:"acodec"`
		Abr               int         `json:"abr,omitempty"`
		Asr               int         `json:"asr"`
		Filesize          int         `json:"filesize"`
		Fps               interface{} `json:"fps"`
		Height            interface{} `json:"height"`
		Tbr               float64     `json:"tbr"`
		Width             interface{} `json:"width"`
		Vcodec            string      `json:"vcodec"`
		DownloaderOptions struct {
			HTTPChunkSize int `json:"http_chunk_size"`
		} `json:"downloader_options,omitempty"`
		Format      string `json:"format"`
		Protocol    string `json:"protocol"`
		HTTPHeaders struct {
			UserAgent      string `json:"User-Agent"`
			AcceptCharset  string `json:"Accept-Charset"`
			Accept         string `json:"Accept"`
			AcceptEncoding string `json:"Accept-Encoding"`
			AcceptLanguage string `json:"Accept-Language"`
		} `json:"http_headers"`
		Container string `json:"container,omitempty"`
	} `json:"formats"`
	IsLive             interface{} `json:"is_live"`
	StartTime          interface{} `json:"start_time"`
	EndTime            interface{} `json:"end_time"`
	Series             interface{} `json:"series"`
	SeasonNumber       interface{} `json:"season_number"`
	EpisodeNumber      interface{} `json:"episode_number"`
	Track              string      `json:"track"`
	Artist             string      `json:"artist"`
	Album              string      `json:"album"`
	ReleaseDate        interface{} `json:"release_date"`
	ReleaseYear        interface{} `json:"release_year"`
	Extractor          string      `json:"extractor"`
	WebpageURLBasename string      `json:"webpage_url_basename"`
	ExtractorKey       string      `json:"extractor_key"`
	Playlist           interface{} `json:"playlist"`
	PlaylistIndex      interface{} `json:"playlist_index"`
	Thumbnails         []struct {
		URL string `json:"url"`
		ID  string `json:"id"`
	} `json:"thumbnails"`
	DisplayID          string      `json:"display_id"`
	RequestedSubtitles interface{} `json:"requested_subtitles"`
	FormatID           string      `json:"format_id"`
	URL                string      `json:"url"`
	PlayerURL          string      `json:"player_url"`
	Ext                string      `json:"ext"`
	Width              int         `json:"width"`
	Height             int         `json:"height"`
	Acodec             string      `json:"acodec"`
	Abr                int         `json:"abr"`
	Vcodec             string      `json:"vcodec"`
	Asr                int         `json:"asr"`
	Filesize           interface{} `json:"filesize"`
	FormatNote         string      `json:"format_note"`
	Fps                interface{} `json:"fps"`
	Tbr                float64     `json:"tbr"`
	Format             string      `json:"format"`
	Protocol           string      `json:"protocol"`
	HTTPHeaders        struct {
		UserAgent      string `json:"User-Agent"`
		AcceptCharset  string `json:"Accept-Charset"`
		Accept         string `json:"Accept"`
		AcceptEncoding string `json:"Accept-Encoding"`
		AcceptLanguage string `json:"Accept-Language"`
	} `json:"http_headers"`
	Fulltitle string `json:"fulltitle"`
	Filename  string `json:"_filename"`
}

type CloseWrapper struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c CloseWrapper) Close() error {
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}

	go io.Copy(ioutil.Discard, c.ReadCloser)

	c.cmd.Wait()

	return nil
}

func YoutubeDLTitle(videoURL string) (string, error) {
	// TODO: implement by copy-pasting YoutubeDLSource with --dump-json
	return "", errors.New("youtube-dl: not implemented")
}

func YoutubeDLSource(videoURL string, ctx context.Context) (*Metadata, io.ReadCloser, error) {
	args := []string{"-f", "[height<=720]", "-o", "-", "--print-json", videoURL}

	wrappedCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(wrappedCtx, "yt-dlp", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	stderrBuf := new(bytes.Buffer)
	rd := io.TeeReader(stderr, stderrBuf)

	var metadata YouTubeDLMetadata

	jsonErr := json.NewDecoder(rd).Decode(&metadata)
	go io.Copy(ioutil.Discard, rd)
	if err != nil {
		// assume video failed
		go io.Copy(ioutil.Discard, stdout)
		cancel()
		err = cmd.Wait()
		if err == nil {
			return nil, nil, errors.New("youtube-dl: exit code discrepancy???")
		}

		if stderrBuf.Len() == 0 {
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			return nil, nil, err
		}

		return nil, nil, fmt.Errorf("youtube-dl: failed to parse json: %w; output: %s", jsonErr, stderrBuf.String())
	}

	meta := Metadata{
		Title:    metadata.Title,
		Duration: time.Duration(metadata.Duration) * time.Second,
	}

	return &meta, stdout, nil
}
