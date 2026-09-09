package xray

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/accloud-proj/x-cmd/internal/appdir"
)

type Release struct {
	TagName string
}

func StableReleases(ctx context.Context, endpoint string, limit int) ([]Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "x-cmd")
	response, err := (&http.Client{Timeout: 8 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("获取 Xray 稳定版本失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 Xray 稳定版本失败: HTTP %s", response.Status)
	}
	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 256<<10)).Decode(&tags); err != nil {
		return nil, fmt.Errorf("解析 Xray 稳定版本失败: %w", err)
	}
	releases := make([]Release, 0, limit)
	for _, tag := range tags {
		if !stableVersionTag(tag.Name) {
			continue
		}
		releases = append(releases, Release{TagName: strings.TrimSpace(tag.Name)})
		if len(releases) == limit {
			break
		}
	}
	return releases, nil
}

func stableVersionTag(tag string) bool {
	version := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	return version != "" && !strings.Contains(version, "-")
}

func Version(ctx context.Context, binary string) (string, error) {
	if binary == "" {
		binary = "xray"
	}
	output, err := exec.CommandContext(ctx, binary, "version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("执行 %s 失败: %w", binary, err)
	}
	line := strings.SplitN(strings.TrimSpace(string(output)), "\n", 2)[0]
	return line, nil
}

func Install(ctx context.Context, version, baseURL, destination string, progress io.Writer) (string, error) {
	if version == "" {
		return "", fmt.Errorf("必须指定版本，例如 v26.3.27")
	}
	if destination == "" {
		destination = defaultInstallDir()
	}
	asset, err := platformAsset()
	if err != nil {
		return "", err
	}
	downloadURL := strings.TrimRight(baseURL, "/") + "/" + version + "/" + asset
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("下载内核失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载内核失败: HTTP %s (%s)", response.Status, downloadURL)
	}
	temporary, err := os.CreateTemp("", "x-cmd-xray-*.zip")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := copyWithProgress(temporary, response.Body, response.ContentLength, progress); err != nil {
		temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := extractZIP(temporaryPath, destination); err != nil {
		return "", err
	}
	binary := filepath.Join(destination, executableName())
	if err := os.Chmod(binary, 0o755); err != nil {
		return "", err
	}
	return binary, nil
}

func copyWithProgress(destination io.Writer, source io.Reader, total int64, output io.Writer) error {
	var downloaded int64
	buffer := make([]byte, 64<<10)
	lastPercent := -1
	for {
		count, readErr := source.Read(buffer)
		if count > 0 {
			if _, err := destination.Write(buffer[:count]); err != nil {
				return err
			}
			downloaded += int64(count)
			if output != nil {
				if total > 0 {
					percent := int(downloaded * 100 / total)
					if percent != lastPercent {
						fmt.Fprintf(output, "\r[下载] %3d%%  %s / %s", percent, formatBytes(downloaded), formatBytes(total))
						lastPercent = percent
					}
				} else {
					fmt.Fprintf(output, "\r[下载] %s", formatBytes(downloaded))
				}
			}
		}
		if readErr == io.EOF {
			if output != nil {
				fmt.Fprintln(output)
			}
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	value := float64(size)
	units := []string{"KB", "MB", "GB"}
	for _, label := range units {
		value /= unit
		if value < unit || label == "GB" {
			return fmt.Sprintf("%.1f %s", value, label)
		}
	}
	return ""
}

func defaultInstallDir() string {
	dir, err := appdir.Default()
	if err != nil {
		return filepath.Join(".", "xray")
	}
	return dir
}

func platformAsset() (string, error) {
	return platformAssetFor(runtime.GOOS, runtime.GOARCH)
}

func platformAssetFor(goos, goarch string) (string, error) {
	arch := map[string]string{
		"amd64": "64", "386": "32", "arm64": "arm64-v8a", "arm": "arm32-v7a",
		"riscv64": "riscv64", "loong64": "loong64",
	}[goarch]
	if arch == "" {
		return "", fmt.Errorf("不支持的 CPU 架构: %s", goarch)
	}
	osName := map[string]string{
		"windows": "windows", "linux": "linux", "darwin": "macos",
		"freebsd": "freebsd", "openbsd": "openbsd",
	}[goos]
	if osName == "" {
		return "", fmt.Errorf("不支持的操作系统: %s", goos)
	}
	if (goarch == "riscv64" || goarch == "loong64") && goos != "linux" {
		return "", fmt.Errorf("不支持的平台组合: %s/%s", goos, goarch)
	}
	return "Xray-" + osName + "-" + arch + ".zip", nil
}

func extractZIP(source, destination string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return fmt.Errorf("打开内核压缩包失败: %w", err)
	}
	defer reader.Close()
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return err
	}
	root := filepath.Clean(destination) + string(os.PathSeparator)
	for _, file := range reader.File {
		target := filepath.Join(destination, file.Name)
		if !strings.HasPrefix(filepath.Clean(target), root) {
			return fmt.Errorf("压缩包包含不安全路径: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		input.Close()
		output.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return "xray.exe"
	}
	return "xray"
}
