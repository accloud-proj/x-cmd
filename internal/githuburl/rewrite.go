package githuburl

import (
	"fmt"
	"net/url"
	"strings"
)

const DefaultMirror = "https://github.uzfdafw.cc"

type Rewriter struct {
	Mirror string
}

func Candidates(mirror string) ([]Rewriter, error) {
	configured, err := New(mirror)
	if err != nil {
		return nil, err
	}
	if configured.Mirror != "" {
		if configured.Mirror == DefaultMirror {
			return []Rewriter{configured, {}}, nil
		}
		return []Rewriter{configured}, nil
	}
	fallback, err := New(DefaultMirror)
	if err != nil {
		return nil, err
	}
	return []Rewriter{{}, fallback}, nil
}

func New(mirror string) (Rewriter, error) {
	mirror = strings.TrimRight(mirror, "/")
	if mirror == "" {
		return Rewriter{}, nil
	}
	if !strings.Contains(mirror, "://") {
		mirror = "https://" + mirror
	}
	parsed, err := url.Parse(mirror)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return Rewriter{}, fmt.Errorf("无效 GitHub 镜像地址 %q", mirror)
	}
	return Rewriter{Mirror: mirror}, nil
}

func (r Rewriter) Rewrite(target string) (string, error) {
	if r.Mirror == "" {
		return target, nil
	}
	parsedTarget, err := url.Parse(target)
	if err != nil || parsedTarget.Host == "" || (parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https") {
		return "", fmt.Errorf("无效 GitHub URL %q", target)
	}
	return r.Mirror + "/" + target, nil
}
