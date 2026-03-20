package main

import (
	"os"

	"impactable/internal/wtl"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(wtl.Run(os.Args[1:], cwd, os.Stdin, os.Stdout, os.Stderr))
}
