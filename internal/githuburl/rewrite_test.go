package githuburl

import "testing"

func TestRewrite(t *testing.T) {
	tests := []struct {
		mirror string
		want   string
	}{
		{mirror: "github.uzfdafw.cc", want: "https://github.uzfdafw.cc/XTLS/Xray-core/releases/download/v1/x.zip"},
		{want: "https://github.com/XTLS/Xray-core/releases/download/v1/x.zip"},
	}
	for _, test := range tests {
		rewriter, err := New(test.mirror)
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

func TestRejectsMirrorPath(t *testing.T) {
	if _, err := New("https://mirror.example/prefix"); err == nil {
		t.Fatal("expected mirror path validation error")
	}
}
