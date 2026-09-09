//go:build !windows

package cli

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

func readKey(input *bufio.Reader) error {
	state, err := runStty("-g")
	if err != nil {
		_, err = input.ReadString('\n')
		return err
	}
	if _, err := runStty("-echo", "-icanon", "min", "1", "time", "0"); err != nil {
		return err
	}
	defer runStty(strings.TrimSpace(string(state)))
	_, err = input.ReadByte()
	return err
}

func runStty(args ...string) ([]byte, error) {
	command := exec.Command("stty", args...)
	command.Stdin = os.Stdin
	return command.Output()
}
