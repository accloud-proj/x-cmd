package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/accloud-proj/x-cmd/internal/githuburl"
)

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
			t.Fatalf("unexpected request path: %s", request.URL.Path)
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
