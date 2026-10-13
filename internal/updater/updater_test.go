package updater

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/accloud-proj/x-cmd/internal/githuburl"
)

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

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.1", "0.2.0", true},
		{"v0.3.0", "0.2.9", true},
		{"v1.0.0", "0.99.99", true},
		{"v0.2.0", "0.2.0", false},
		{"v0.1.9", "0.2.0", false},
	}
	for _, test := range tests {
		if got := IsNewer(test.latest, test.current); got != test.want {
			t.Errorf("IsNewer(%q, %q) = %t", test.latest, test.current, got)
		}
	}
}

func TestLatestFallsBackToAtomWhenAPIIsForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/releases/latest") {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		if !strings.HasSuffix(request.URL.Path, "/releases.atom") {
			t.Errorf("unexpected request path: %s", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/atom+xml")
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><entry><title>v1.2.3</title></entry></feed>`))
	}))
	defer server.Close()
	release, err := Latest(context.Background(), "owner/repository", githuburl.Rewriter{Mirror: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if release.TagName != "v1.2.3" || len(release.Assets) != 1 || release.Assets[0].Name != platformAssetName() {
		t.Fatalf("unexpected release: %#v", release)
	}
	wantURL := "https://github.com/owner/repository/releases/download/v1.2.3/" + platformAssetName()
	if release.Assets[0].URL != wantURL {
		t.Fatalf("asset URL = %q, want %q", release.Assets[0].URL, wantURL)
	}
}
func TestLatestFallsBackToReleasePage(t *testing.T) {
	for _, test := range []struct {
		name, finalPath string
		pageStatus      int
		wantError       string
	}{
		{"tag redirect", "/https://github.com/owner/repository/releases/tag/v1.2.3", http.StatusOK, ""},
		{"no redirect", "", http.StatusOK, "无法从 GitHub Release 页面解析版本标签"},
		{"wrong repository", "/https://github.com/other/repository/releases/tag/v1.2.3", http.StatusOK, "无法从 GitHub Release 页面解析版本标签"},
		{"empty tag", "/https://github.com/owner/repository/releases/tag/", http.StatusOK, "无效版本标签"},
		{"all forbidden", "", http.StatusForbidden, "GitHub Release 页面 HTTP 403"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				paths = append(paths, request.URL.Path)
				switch request.URL.Path {
				case "/https://api.github.com/repos/owner/repository/releases/latest",
					"/https://github.com/owner/repository/releases.atom":
					writer.WriteHeader(http.StatusForbidden)
				case "/https://github.com/owner/repository/releases/latest":
					if test.finalPath != "" {
						writer.Header().Set("Location", test.finalPath)
						writer.WriteHeader(http.StatusFound)
						return
					}
					writer.WriteHeader(test.pageStatus)
				default:
					if request.URL.Path != test.finalPath {
						t.Errorf("unexpected request: %s", request.URL.Path)
						writer.WriteHeader(http.StatusNotFound)
						return
					}
					writer.WriteHeader(test.pageStatus)
				}
			}))
			defer server.Close()

			rewriter := githuburl.Rewriter{Mirror: server.URL}
			release, err := Latest(context.Background(), "owner/repository", rewriter)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) ||
					!strings.Contains(err.Error(), "GitHub Release API HTTP 403") ||
					!strings.Contains(err.Error(), "GitHub Release Feed HTTP 403") {
					t.Fatalf("Latest() error = %v, want all request failures including %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(paths) != 4 {
				t.Fatalf("request paths = %#v, want API, feed, page and redirect", paths)
			}
			if release.TagName != "v1.2.3" || release.HTMLURL != "https://github.com/owner/repository/releases/tag/v1.2.3" ||
				len(release.Assets) != 1 || release.Assets[0].Name != platformAssetName() {
				t.Fatalf("unexpected release: %#v", release)
			}
			wantURL := "https://github.com/owner/repository/releases/download/v1.2.3/" + platformAssetName()
			if release.Assets[0].URL != wantURL {
				t.Fatalf("asset URL = %q, want %q", release.Assets[0].URL, wantURL)
			}
			downloadURL, err := rewriter.Rewrite(release.Assets[0].URL)
			if err != nil || downloadURL != server.URL+"/"+wantURL {
				t.Fatalf("mirrored asset URL = %q, error = %v", downloadURL, err)
			}
		})
	}
}

func TestLatestUsesAPIAssetsWithoutFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/https://api.github.com/repos/owner/repository/releases/latest" {
			t.Errorf("unexpected fallback request: %s", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = writer.Write([]byte(`{"tag_name":"v1.2.3","assets":[{"name":"custom.zip","browser_download_url":"https://github.com/owner/repository/releases/download/v1.2.3/custom.zip"}]}`))
	}))
	defer server.Close()
	release, err := Latest(context.Background(), "owner/repository", githuburl.Rewriter{Mirror: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if release.TagName != "v1.2.3" || len(release.Assets) != 1 || release.Assets[0].Name != "custom.zip" {
		t.Fatalf("API assets were not preserved: %#v", release)
	}
}
