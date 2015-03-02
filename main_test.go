package main

import (
	"testing"
	"bytes"
)

func beforeEach() {
	var outb, inb, errb bytes.Buffer
	outW = &outb
	inR = &inb
	errW = &errb
}

func TestHelp(t *testing.T) {
	beforeEach()
	err := wks([]string{"wks", "help"})
	if err != nil {
		t.Error("wks help returned error")
	}
	//println("output", out.String())
}

