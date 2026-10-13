package xray

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLatestReleaseUsesFinalRedirectTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/latest" {
			http.Redirect(writer, request, "/releases/tag/v26.10.10-beta.1", http.StatusFound)
			return
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	release, err := LatestRelease(context.Background(), server.URL+"/latest")
	if err != nil {
		t.Fatal(err)
	}
	if release.TagName != "v26.10.10-beta.1" {
		t.Fatalf("tag = %q, want v26.10.10-beta.1", release.TagName)
	}
}

func TestReleasesIncludesPrereleasesAndLimitsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if accept := request.Header.Get("Accept"); accept != "application/vnd.github+json" {
			t.Errorf("Accept = %q", accept)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{"tag_name":"v26.10.0","draft":true},
			{"tag_name":""},
			{"tag_name":"v26.9.1","prerelease":true},
			{"tag_name":"v26.9.0-beta.1","prerelease":true},
			{"tag_name":"v26.8.1","prerelease":false}
		]`))
	}))
	defer server.Close()

	releases, err := Releases(context.Background(), server.URL, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 || releases[0].TagName != "v26.9.1" || releases[1].TagName != "v26.9.0-beta.1" {
		t.Fatalf("unexpected releases: %#v", releases)
	}
}

func TestCopyWithProgress(t *testing.T) {
	content := bytes.Repeat([]byte("x"), 128<<10)
	var destination bytes.Buffer
	var progress bytes.Buffer
	if err := copyWithProgress(&destination, bytes.NewReader(content), int64(len(content)), &progress); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(destination.Bytes(), content) {
		t.Fatal("downloaded content differs")
	}
	if !strings.Contains(progress.String(), "100%") {
		t.Fatalf("progress = %q", progress.String())
	}
}

func TestPlatformAssetForAdditionalPlatforms(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
	}{
		{"openbsd", "amd64", "Xray-openbsd-64.zip"},
		{"openbsd", "arm", "Xray-openbsd-arm32-v7a.zip"},
		{"openbsd", "arm64", "Xray-openbsd-arm64-v8a.zip"},
		{"freebsd", "amd64", "Xray-freebsd-64.zip"},
		{"freebsd", "arm", "Xray-freebsd-arm32-v7a.zip"},
		{"freebsd", "arm64", "Xray-freebsd-arm64-v8a.zip"},
		{"linux", "riscv64", "Xray-linux-riscv64.zip"},
		{"linux", "loong64", "Xray-linux-loong64.zip"},
	}
	for _, test := range tests {
		got, err := platformAssetFor(test.goos, test.goarch)
		if err != nil {
			t.Fatalf("platformAssetFor(%q, %q): %v", test.goos, test.goarch, err)
		}
		if got != test.want {
			t.Fatalf("platformAssetFor(%q, %q) = %q, want %q", test.goos, test.goarch, got, test.want)
		}
	}
}
