package main

import (
	"os"
	"wks/core"
)

func main() {
	core.Priam(os.Args, os.Stdin, os.Stdout, os.Stderr)
}
