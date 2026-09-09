package xray

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStableReleasesFiltersPrereleasesAndLimitsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if accept := request.Header.Get("Accept"); accept != "application/vnd.github+json" {
			t.Errorf("Accept = %q", accept)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{"name":"v26.9.1"},
			{"name":"v26.9.0-beta.1"},
			{"name":"v26.8.1"},
			{"name":"v26.8.0-preview"},
			{"name":"v26.7.1"}
		]`))
	}))
	defer server.Close()

	releases, err := StableReleases(context.Background(), server.URL, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 || releases[0].TagName != "v26.9.1" || releases[1].TagName != "v26.8.1" {
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
