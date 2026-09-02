package xray

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecentReleasesFiltersDraftsAndLimitsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
            {"tag_name":"v26.8.1","published_at":"2026-08-01T00:00:00Z","draft":false},
            {"tag_name":"v26.8.0-draft","published_at":"2026-07-31T00:00:00Z","draft":true},
            {"tag_name":"v26.7.1","published_at":"2026-07-01T00:00:00Z","draft":false}
        ]`))
	}))
	defer server.Close()

	releases, err := RecentReleases(context.Background(), server.URL, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 || releases[0].TagName != "v26.8.1" || releases[1].TagName != "v26.7.1" {
		t.Fatalf("unexpected releases: %#v", releases)
	}
}

func TestRecentReleasesParsesAtomFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if accept := request.Header.Get("Accept"); accept != "application/atom+xml, application/xml;q=0.9, */*;q=0.8" {
			t.Errorf("unexpected Accept header: %q", accept)
		}
		writer.Header().Set("Content-Type", "application/atom+xml")
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom">
<entry><title>v26.9.1</title><updated>2026-09-01T00:00:00Z</updated></entry>
<entry><title>v26.8.1</title><updated>2026-08-01T00:00:00Z</updated></entry>
</feed>`))
	}))
	defer server.Close()
	releases, err := RecentReleases(context.Background(), server.URL+"/releases.atom", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].TagName != "v26.9.1" {
		t.Fatalf("unexpected releases: %#v", releases)
	}
}
