package main

import (
	"fmt"
	"os"

	"github.com/accloud-proj/x-cmd/internal/cli"
)

func main() {
	if err := cli.New().Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "[错误]", err)
		os.Exit(1)
	}
}
