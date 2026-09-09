package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/accloud-proj/x-cmd/internal/cli"
)

//go:embed banner.txt
var banner string

func main() {
	if err := cli.New(banner).Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "[错误]", err)
		os.Exit(1)
	}
}
