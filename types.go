package main

import (
	"bytes"
	"strconv"

	colorful "github.com/lucasb-eyer/go-colorful"
)

type pixel struct {
	colorful.Color
}

func (p pixel) MarshalJSON() ([]byte, error) {
	out := &bytes.Buffer{}
	r, g, b := p.RGB255()
	var rgb int
	rgb = (int(r) << 16) + (int(g) << 8) + int(b)
	out.WriteString(strconv.Itoa(rgb))
	return out.Bytes(), nil
}
