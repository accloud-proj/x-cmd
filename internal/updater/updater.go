package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/accloud-proj/x-cmd/internal/githuburl"
)

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

type releaseFeed struct {
	Entries []struct {
		Title string `xml:"title"`
	} `xml:"entry"`
}

func Latest(ctx context.Context, repository string, rewriter githuburl.Rewriter) (Release, error) {
	requestURL, err := rewriter.Rewrite("https://api.github.com/repos/" + repository + "/releases/latest")
	if err != nil {
		return Release{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "x-cmd-updater")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	if err != nil {
		return latestFromAtom(ctx, repository, rewriter)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return latestFromAtom(ctx, repository, rewriter)
	}
	defer response.Body.Close()
	var release Release
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&release); err != nil {
		return latestFromAtom(ctx, repository, rewriter)
	}
	if release.TagName == "" {
		return latestFromAtom(ctx, repository, rewriter)
	}
	return release, nil
}

func latestFromAtom(ctx context.Context, repository string, rewriter githuburl.Rewriter) (Release, error) {
	requestURL, err := rewriter.Rewrite("https://github.com/" + repository + "/releases.atom")
	if err != nil {
		return Release{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/atom+xml, application/xml;q=0.9, */*;q=0.8")
	request.Header.Set("User-Agent", "x-cmd-updater")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	if err != nil {
		return Release{}, fmt.Errorf("检查更新失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("检查更新失败: GitHub Release Feed HTTP %s", response.Status)
	}
	var feed releaseFeed
	if err := xml.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&feed); err != nil {
		return Release{}, fmt.Errorf("解析 GitHub Release Feed 失败: %w", err)
	}
	if len(feed.Entries) == 0 || strings.TrimSpace(feed.Entries[0].Title) == "" {
		return Release{}, fmt.Errorf("GitHub Release Feed 没有版本标签")
	}
	tag := strings.TrimSpace(feed.Entries[0].Title)
	assetName := platformAssetName()
	return Release{
		TagName: tag,
		HTMLURL: "https://github.com/" + repository + "/releases/tag/" + tag,
		Assets: []Asset{{
			Name: assetName,
			URL:  "https://github.com/" + repository + "/releases/download/" + tag + "/" + assetName,
		}},
	}, nil
}

func IsNewer(latest, current string) bool {
	left, leftOK := versionParts(latest)
	right, rightOK := versionParts(current)
	if !leftOK || !rightOK {
		return strings.TrimPrefix(latest, "v") != strings.TrimPrefix(current, "v")
	}
	for index := 0; index < 3; index++ {
		if left[index] != right[index] {
			return left[index] > right[index]
		}
	}
	return false
}

func Install(ctx context.Context, release Release, rewriter githuburl.Rewriter) error {
	assetName := platformAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.URL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("Release %s 缺少当前平台资产 %s", release.TagName, assetName)
	}
	downloadURL, err := rewriter.Rewrite(downloadURL)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "x-cmd-updater")
	response, err := (&http.Client{Timeout: 5 * time.Minute}).Do(request)
	if err != nil {
		return fmt.Errorf("下载更新失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("下载更新失败: HTTP %s", response.Status)
	}
	archive, err := os.CreateTemp("", "x-cmd-update-*")
	if err != nil {
		return err
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	if _, err := io.Copy(archive, io.LimitReader(response.Body, 200<<20)); err != nil {
		archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return err
	}
	newPath := executable + ".new"
	_ = os.Remove(newPath)
	if err := extractExecutable(archivePath, assetName, newPath); err != nil {
		return err
	}
	if err := os.Chmod(newPath, 0o755); err != nil {
		os.Remove(newPath)
		return err
	}
	backup := executable + ".old"
	_ = os.Remove(backup)
	if runtime.GOOS != "windows" {
		if err := os.Rename(newPath, executable); err != nil {
			os.Remove(newPath)
			return fmt.Errorf("替换程序失败: %w", err)
		}
		return nil
	}
	if err := os.Rename(executable, backup); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("备份当前程序失败: %w", err)
	}
	if err := os.Rename(newPath, executable); err != nil {
		_ = os.Rename(backup, executable)
		return fmt.Errorf("替换程序失败，已回滚: %w", err)
	}
	return nil
}

func platformAssetName() string {
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	return fmt.Sprintf("x-cmd_%s_%s%s", runtime.GOOS, runtime.GOARCH, extension)
}

func extractExecutable(archivePath, assetName, destination string) error {
	if strings.HasSuffix(assetName, ".zip") {
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return err
		}
		defer reader.Close()
		for _, file := range reader.File {
			if filepath.Base(file.Name) != "x-cmd.exe" {
				continue
			}
			input, err := file.Open()
			if err != nil {
				return err
			}
			defer input.Close()
			return writeExecutable(destination, input)
		}
	} else {
		file, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer file.Close()
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader := tar.NewReader(gzipReader)
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if filepath.Base(header.Name) == "x-cmd" {
				return writeExecutable(destination, reader)
			}
		}
	}
	return fmt.Errorf("更新压缩包中没有 x-cmd 可执行文件")
}

func writeExecutable(destination string, source io.Reader) error {
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, source)
	closeErr := output.Close()
	if copyErr != nil {
		os.Remove(destination)
		return copyErr
	}
	return closeErr
}

func versionParts(value string) ([3]int, bool) {
	var result [3]int
	value = strings.TrimPrefix(value, "v")
	value = strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, false
	}
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return result, false
		}
		result[index] = number
	}
	return result, true
}
