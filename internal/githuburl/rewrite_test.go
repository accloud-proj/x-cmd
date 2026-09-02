package githuburl

import "testing"

func TestRewrite(t *testing.T) {
	tests := []struct {
		mirror string
		want   string
	}{
		{mirror: "github.uzfdafw.cc", want: "https://github.uzfdafw.cc/https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-linux-64.zip"},
		{want: "https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-linux-64.zip"},
	}
	for _, test := range tests {
		rewriter, err := New(test.mirror)
		if err != nil {
			t.Fatal(err)
		}
		got, err := rewriter.Rewrite("https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-linux-64.zip")
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Errorf("Rewrite() = %q, want %q", got, test.want)
		}
	}
}

func TestAcceptsMirrorPrefixPath(t *testing.T) {
	rewriter, err := New("https://mirror.example/prefix/")
	if err != nil {
		t.Fatal(err)
	}
	got, err := rewriter.Rewrite("https://github.com/example/project")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://mirror.example/prefix/https://github.com/example/project" {
		t.Fatalf("Rewrite() = %q", got)
	}
}
