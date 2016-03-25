package main

import (
	"os"
	"priam/core"
)

func main() {
	core.Priam(os.Args, os.Stdin, os.Stdout, os.Stderr)
}
