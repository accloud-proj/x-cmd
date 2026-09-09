//go:build !windows

package shellenv

func parentProcessName() string {
	return ""
}
