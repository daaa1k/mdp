// verify-webp-roundtrip verifies that a WebP file can be decoded and is not all-white.
// Usage: go run verify-webp-roundtrip.go <output.webp>
package main

import (
	"fmt"
	"image"
	"os"

	_ "golang.org/x/image/webp"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <output.webp>\n", os.Args[0])
		os.Exit(2)
	}

	img, err := decodeFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}

	b := img.Bounds()
	total := b.Dx() * b.Dy()
	white := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r >= 0xfefe && g >= 0xfefe && bl >= 0xfefe {
				white++
			}
		}
	}

	if total > 1 && white == total {
		fmt.Fprintf(os.Stderr, "FAIL: all %d pixels are white\n", total)
		os.Exit(1)
	}
	fmt.Printf("ok: %dx%d decoded, %d/%d non-white pixels\n", b.Dx(), b.Dy(), total-white, total)
}

func decodeFile(path string) (image.Image, error) {
	f, err := os.Open(path) //nolint:gosec // CLI tool, path from user args
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only file
	img, _, err := image.Decode(f)
	return img, err
}
