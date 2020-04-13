package stream

import (
	"context"
	"io"
	"os/exec"
	"time"
)

func StreamlinkSource(videoURL string, ctx context.Context) (*Metadata, io.ReadCloser, error) {
	args := []string{videoURL, "best", "-O"}

	// wrappedCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(ctx, "streamlink", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	return nil, nil, err
	// }

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	// stderrBuf := new(bytes.Buffer)
	// rd := io.TeeReader(stderr, stderrBuf)

	var metadata YouTubeDLMetadata

	// jsonErr = json.NewDecoder(rd).Decode(&metadata)
	// go io.Copy(ioutil.Discard, rd)
	// if err != nil {
	// 	origErr := err
	// 	// assume video failed
	// 	go io.Copy(ioutil.Discard, stdout)
	// 	cancel()
	// 	err = cmd.Wait()
	// 	if err == nil {
	// 		return nil, nil, errors.New("youtube-dl: exit code discrepancy???")
	// 	}

	// 	if stderrBuf.Len() == 0 {
	// 		if ctx.Err() != nil {
	// 			return nil, nil, ctx.Err()
	// 		}
	// 		return nil, nil, err
	// 	}

	// 	return nil, nil, fmt.Errorf("youtube-dl: failed to parse json: %w; output: %s", origErr, stderrBuf.String())
	// }

	meta := Metadata{
		Title:    metadata.Title,
		Duration: time.Duration(metadata.Duration) * time.Second,
	}

	return &meta, stdout, nil
}
