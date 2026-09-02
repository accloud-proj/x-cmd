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
)

type Release struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
}

func RecentReleases(ctx context.Context, endpoint string, limit int) ([]Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "x-cmd")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("获取 Xray Release 失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 Xray Release 失败: HTTP %s", response.Status)
	}
	var releases []Release
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("解析 Xray Release 失败: %w", err)
	}
	result := make([]Release, 0, limit)
	for _, release := range releases {
		if release.Draft || release.TagName == "" {
			continue
		}
		result = append(result, release)
		if len(result) == limit {
			break
		}
	}
	return result, nil
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

func Install(ctx context.Context, version, baseURL, destination string) (string, error) {
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
	if _, err := io.Copy(temporary, response.Body); err != nil {
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

func defaultInstallDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "xray")
	}
	return filepath.Join(dir, "x-cmd", "xray")
}

func platformAsset() (string, error) {
	arch := map[string]string{"amd64": "64", "386": "32", "arm64": "arm64-v8a", "arm": "arm32-v7a"}[runtime.GOARCH]
	if arch == "" {
		return "", fmt.Errorf("不支持的 CPU 架构: %s", runtime.GOARCH)
	}
	osName := map[string]string{"windows": "windows", "linux": "linux", "darwin": "macos"}[runtime.GOOS]
	if osName == "" {
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
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
