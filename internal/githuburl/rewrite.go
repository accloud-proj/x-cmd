package githuburl

import (
	"fmt"
	"net/url"
	"strings"
)

type Rewriter struct {
	Mirror string
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
