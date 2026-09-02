package githuburl

import (
	"fmt"
	"net/url"
	"strings"
)

type Rewriter struct {
	Mirror string
	Host   string
}

func New(mirror, host string) (Rewriter, error) {
	if mirror != "" && host != "" {
		return Rewriter{}, fmt.Errorf("GitHub 镜像和替换主机不能同时设置")
	}
	rewriter := Rewriter{Mirror: strings.TrimRight(mirror, "/"), Host: strings.TrimRight(host, "/")}
	if rewriter.Host != "" && !strings.Contains(rewriter.Host, "://") {
		rewriter.Host = "https://" + rewriter.Host
	}
	if rewriter.Host != "" {
		parsed, err := url.Parse(rewriter.Host)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" {
			return Rewriter{}, fmt.Errorf("无效 GitHub 替换主机 %q", host)
		}
	}
	return rewriter, nil
}

func (r Rewriter) Rewrite(target string) (string, error) {
	if r.Mirror != "" {
		return r.Mirror + "/" + target, nil
	}
	if r.Host == "" {
		return target, nil
	}
	parsedTarget, err := url.Parse(target)
	if err != nil || parsedTarget.Host == "" {
		return "", fmt.Errorf("无效 GitHub URL %q", target)
	}
	parsedHost, _ := url.Parse(r.Host)
	parsedTarget.Scheme = parsedHost.Scheme
	parsedTarget.Host = parsedHost.Host
	return parsedTarget.String(), nil
}
