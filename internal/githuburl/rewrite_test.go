package githuburl

import "testing"

func TestRewrite(t *testing.T) {
	tests := []struct {
		mirror string
		host   string
		want   string
	}{
		{mirror: "https://mirror.example/", want: "https://mirror.example/https://github.com/XTLS/Xray-core/releases/download/v1/x.zip"},
		{host: "downloads.example.com", want: "https://downloads.example.com/XTLS/Xray-core/releases/download/v1/x.zip"},
		{want: "https://github.com/XTLS/Xray-core/releases/download/v1/x.zip"},
	}
	for _, test := range tests {
		rewriter, err := New(test.mirror, test.host)
		if err != nil {
			t.Fatal(err)
		}
		got, err := rewriter.Rewrite("https://github.com/XTLS/Xray-core/releases/download/v1/x.zip")
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Errorf("Rewrite() = %q, want %q", got, test.want)
		}
	}
}

func TestRejectsMirrorAndHostTogether(t *testing.T) {
	if _, err := New("https://mirror.example", "downloads.example.com"); err == nil {
		t.Fatal("expected mutually exclusive settings error")
	}
}
