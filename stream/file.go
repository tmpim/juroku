package stream

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func FileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

var durationPattern = regexp.MustCompile(`^\s+Duration: (\d*):(\d*):(\d*)\.(\d*),`)

func FileSource(path string) (*Metadata, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	out, err := exec.Command("ffprobe", path).CombinedOutput()
	if err != nil {
		return nil, err
	}

	meta := Metadata{
		Title: FileTitle(path),
	}

	matches := durationPattern.FindStringSubmatch(string(out))
	if len(matches) != 0 {
		var h, m, s, ms int
		if len(matches[4]) < 3 {
			matches[4] = matches[4] + strings.Repeat("0", 3-len(matches[4]))
		}
		_, err := fmt.Sscanf(matches[1]+" "+matches[2]+" "+matches[3]+" "+
			matches[4][:3], "%d %d %d %d", &h, &m, &s, &ms)
		if err == nil {
			meta.Duration = time.Duration(h)*time.Hour +
				time.Duration(m)*time.Minute +
				time.Duration(s)*time.Second +
				time.Duration(ms)*time.Millisecond
		}
	}

	return &meta, err
}
