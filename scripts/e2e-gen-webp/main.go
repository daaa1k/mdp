package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/daaa1k/mdp/internal/webpenc"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := webpenc.Encode(&buf, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
