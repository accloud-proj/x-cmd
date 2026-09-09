package githuburl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const probeBytes = 256 << 10

func DownloadFastEnough(ctx context.Context, target string, minimumBytesPerSecond int64) (bool, float64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, 0, err
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=0-%d", probeBytes-1))
	request.Header.Set("User-Agent", "x-cmd-speed-test")
	started := time.Now()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		return false, 0, fmt.Errorf("GitHub 测速失败: HTTP %s", response.Status)
	}
	read, err := io.Copy(io.Discard, io.LimitReader(response.Body, probeBytes))
	if err != nil {
		return false, 0, err
	}
	elapsed := time.Since(started).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	bytesPerSecond := float64(read) / elapsed
	return read > 0 && bytesPerSecond >= float64(minimumBytesPerSecond), bytesPerSecond, nil
}
