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

func TestCandidatesUseBuiltInMirrorOnlyAsFallback(t *testing.T) {
	candidates, err := Candidates("")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].Mirror != "" || candidates[1].Mirror != DefaultMirror {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}

	candidates, err = Candidates("https://mirror.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Mirror != "https://mirror.example" {
		t.Fatalf("unexpected configured candidates: %#v", candidates)
	}

	candidates, err = Candidates(DefaultMirror)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].Mirror != DefaultMirror || candidates[1].Mirror != "" {
		t.Fatalf("built-in mirror should fall back to direct GitHub: %#v", candidates)
	}
}
