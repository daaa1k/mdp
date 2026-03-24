package webpenc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"testing"

	_ "golang.org/x/image/webp"
)

func TestRoundtripGradient256(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := range 256 {
		for x := range 256 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8((x + y) / 2), A: 255})
		}
	}
	testRoundtrip(t, img)
}

func TestRoundtripClipboardPNG(t *testing.T) {
	path := os.Getenv("CLIPBOARD_PNG")
	if path == "" {
		t.Skip("CLIPBOARD_PNG not set")
	}
	f, err := os.Open(path) //nolint:gosec // test file path from env var
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close() //nolint:errcheck // test cleanup
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	testRoundtrip(t, img)
}

func testRoundtrip(t *testing.T, img image.Image) {
	t.Helper()
	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	t.Logf("encoded: %d bytes", buf.Len())

	decoded, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode webp: %v", err)
	}

	bounds := img.Bounds()
	mismatches := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, a1 := img.At(x, y).RGBA()
			r2, g2, b2, a2 := decoded.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				if mismatches < 3 {
					t.Logf("pixel (%d,%d): orig=(%d,%d,%d,%d) got=(%d,%d,%d,%d)",
						x, y, r1>>8, g1>>8, b1>>8, a1>>8, r2>>8, g2>>8, b2>>8, a2>>8)
				}
				mismatches++
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d pixel mismatches out of %d", mismatches, bounds.Dx()*bounds.Dy())
	}
}

func TestRoundtripSyntheticSameSize(t *testing.T) {
	// Same size as clipboard image: 382x154
	img := image.NewNRGBA(image.Rect(0, 0, 382, 154))
	for y := range 154 {
		for x := range 382 {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*7 + y*3) % 256),
				G: uint8((x*11 + y*5) % 256),
				B: uint8((x*13 + y*17) % 256),
				A: 255,
			})
		}
	}
	testRoundtrip(t, img)
}

func TestRoundtripLargeFlat(t *testing.T) {
	// Large image with many same-color areas (like screenshots)
	img := image.NewNRGBA(image.Rect(0, 0, 400, 200))
	// White background
	for y := range 200 {
		for x := range 400 {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// Some colored regions
	for y := 10; y < 50; y++ {
		for x := 10; x < 100; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 50, G: 100, B: 200, A: 255})
		}
	}
	for y := 60; y < 150; y++ {
		for x := 50; x < 350; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 30, G: 30, B: 30, A: 255})
		}
	}
	testRoundtrip(t, img)
}

func TestRoundtripThreeColors(t *testing.T) {
	// Minimal 3-color image to isolate Huffman code issue
	img := image.NewNRGBA(image.Rect(0, 0, 6, 1))
	// 3 pixels of color A, 2 pixels of color B, 1 pixel of color C
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(4, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(5, 0, color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	testRoundtrip(t, img)
}

func TestRoundtripFourColors(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 40, G: 50, B: 60, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 70, G: 80, B: 90, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 100, G: 110, B: 120, A: 255})
	testRoundtrip(t, img)
}

func TestRoundtripTwoPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 40, G: 50, B: 60, A: 255})
	testRoundtrip(t, img)
}

func TestRoundtripDebugThreeColors(t *testing.T) {
	// Encode and write to temp file for external verification
	img := image.NewNRGBA(image.Rect(0, 0, 6, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(4, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(5, 0, color.NRGBA{R: 50, G: 50, B: 50, A: 255})

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Dump hex
	data := buf.Bytes()
	t.Logf("total bytes: %d", len(data))
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		hex := ""
		for _, b := range data[i:end] {
			hex += fmt.Sprintf("%02x ", b)
		}
		t.Logf("%04x: %s", i, hex)
	}
}

func totalBits(bw *bitWriter) int {
	return bw.buf.Len()*8 + int(bw.nbits) //nolint:gosec // nbits is always 0..7
}

func TestEncoderBitCounts(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(4, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(5, 0, color.NRGBA{R: 50, G: 50, B: 50, A: 255})

	// Manually encode step by step, tracking bit positions
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	npix := width * height
	gPix := make([]byte, npix)
	rPix := make([]byte, npix)
	bPix := make([]byte, npix)
	aPix := make([]byte, npix)

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			gPix[idx] = c.G
			rPix[idx] = c.R
			bPix[idx] = c.B
			aPix[idx] = c.A
			idx++
		}
	}

	bw := &bitWriter{}

	// Signature
	bw.writeBits(0x2f, 8)
	posAfterSig := totalBits(bw)

	// Size header
	hdr := uint32(width-1) | uint32(height-1)<<14 //nolint:gosec // test values are small
	bw.writeBits(hdr, 32)
	posAfterHdr := totalBits(bw)

	// No transforms, cache, meta
	bw.writeBits(0, 1)
	bw.writeBits(0, 1)
	bw.writeBits(0, 1)
	posAfterFlags := totalBits(bw)

	const greenAlphaSize = 256 + 24

	gCodes := buildAndWriteHuffman(bw, byteFreq(gPix), greenAlphaSize)
	posAfterGreen := totalBits(bw)

	rCodes := buildAndWriteHuffman(bw, byteFreq(rPix), 256)
	posAfterRed := totalBits(bw)

	bCodes := buildAndWriteHuffman(bw, byteFreq(bPix), 256)
	posAfterBlue := totalBits(bw)

	aCodes := buildAndWriteHuffman(bw, byteFreq(aPix), 256)
	posAfterAlpha := totalBits(bw)

	// Distance
	bw.writeBits(1, 1)
	bw.writeBits(0, 1)
	bw.writeBits(0, 1)
	bw.writeBits(0, 1)
	posAfterDist := totalBits(bw)

	t.Logf("Bit positions:")
	t.Logf("  After sig:    %d", posAfterSig)
	t.Logf("  After hdr:    %d", posAfterHdr)
	t.Logf("  After flags:  %d", posAfterFlags)
	t.Logf("  After green:  %d (green=%d bits)", posAfterGreen, posAfterGreen-posAfterFlags)
	t.Logf("  After red:    %d (red=%d bits)", posAfterRed, posAfterRed-posAfterGreen)
	t.Logf("  After blue:   %d (blue=%d bits)", posAfterBlue, posAfterBlue-posAfterRed)
	t.Logf("  After alpha:  %d (alpha=%d bits)", posAfterAlpha, posAfterAlpha-posAfterBlue)
	t.Logf("  After dist:   %d (dist=%d bits)", posAfterDist, posAfterDist-posAfterAlpha)
	t.Logf("  Total header: %d bits", posAfterDist)

	// Pixel data
	for i := range npix {
		emit(bw, gCodes[gPix[i]])
		emit(bw, rCodes[rPix[i]])
		emit(bw, bCodes[bPix[i]])
		emit(bw, aCodes[aPix[i]])
	}

	posAfterPixels := totalBits(bw)
	t.Logf("  After pixels: %d (pixels=%d bits)", posAfterPixels, posAfterPixels-posAfterDist)
	t.Logf("  Total:        %d bits = %d bytes + %d bits", posAfterPixels, posAfterPixels/8, posAfterPixels%8)

	// Print the codes
	for _, sym := range []byte{50, 100, 200} {
		t.Logf("  G code[%d]: code=%d len=%d", sym, gCodes[sym].code, gCodes[sym].length)
		t.Logf("  R code[%d]: code=%d len=%d", sym, rCodes[sym].code, rCodes[sym].length)
		t.Logf("  B code[%d]: code=%d len=%d", sym, bCodes[sym].code, bCodes[sym].length)
	}
	t.Logf("  A code[255]: code=%d len=%d", aCodes[255].code, aCodes[255].length)
}
