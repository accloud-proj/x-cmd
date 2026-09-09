package githuburl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDownloadFastEnough(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
		want  bool
	}{
		{name: "fast", want: true},
		{name: "slow", delay: 250 * time.Millisecond, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				time.Sleep(test.delay)
				_, _ = writer.Write(make([]byte, 16<<10))
			}))
			defer server.Close()
			fast, _, err := DownloadFastEnough(context.Background(), server.URL, 100<<10)
			if err != nil {
				t.Fatal(err)
			}
			if fast != test.want {
				t.Fatalf("fast = %v, want %v", fast, test.want)
			}
		})
	}
}
