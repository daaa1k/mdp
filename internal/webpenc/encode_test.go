package webpenc

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestEncodeRIFFStructure(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	data := buf.Bytes()
	if len(data) < 21 {
		t.Fatalf("output too short: %d bytes", len(data))
	}

	// RIFF header
	if string(data[:4]) != "RIFF" {
		t.Fatal("missing RIFF header")
	}
	riffSize := binary.LittleEndian.Uint32(data[4:8])
	if int(riffSize)+8 != len(data) {
		t.Fatalf("RIFF size mismatch: header says %d, file is %d", riffSize+8, len(data))
	}
	if string(data[8:12]) != "WEBP" {
		t.Fatal("missing WEBP tag")
	}

	// VP8L chunk
	if string(data[12:16]) != "VP8L" {
		t.Fatal("missing VP8L chunk")
	}
	chunkSize := binary.LittleEndian.Uint32(data[16:20])
	if int(chunkSize)+20 > len(data)+1 { // +1 for possible padding
		t.Fatalf("VP8L chunk size %d exceeds file", chunkSize)
	}

	// VP8L signature
	if data[20] != 0x2f {
		t.Fatalf("VP8L signature: got 0x%02x, want 0x2f", data[20])
	}

	// Image size header
	hdr := binary.LittleEndian.Uint32(data[21:25])
	w := int(hdr&0x3FFF) + 1
	h := int((hdr>>14)&0x3FFF) + 1
	if w != 4 || h != 4 {
		t.Fatalf("decoded size %dx%d, want 4x4", w, h)
	}
}

func TestEncodeGradient(t *testing.T) {
	w, h := 16, 16
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 16),
				G: uint8(y * 16),
				B: uint8((x + y) * 8),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	data := buf.Bytes()
	if string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || string(data[12:16]) != "VP8L" {
		t.Fatal("invalid WebP structure")
	}
}

func TestEncodeWithAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 255, B: 0, A: 128})
	img.SetNRGBA(0, 1, color.NRGBA{R: 0, G: 0, B: 255, A: 64})
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 0})

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Check alpha_is_used flag in header
	data := buf.Bytes()
	hdr := binary.LittleEndian.Uint32(data[21:25])
	alphaUsed := (hdr >> 28) & 1
	if alphaUsed != 1 {
		t.Fatal("alpha_is_used should be set")
	}
}

func TestEncodeSinglePixel(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 42, G: 100, B: 200, A: 255})

	var buf bytes.Buffer
	if err := Encode(&buf, img); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	data := buf.Bytes()
	if string(data[:4]) != "RIFF" {
		t.Fatal("invalid WebP")
	}

	hdr := binary.LittleEndian.Uint32(data[21:25])
	w := int(hdr&0x3FFF) + 1
	h := int((hdr>>14)&0x3FFF) + 1
	if w != 1 || h != 1 {
		t.Fatalf("decoded size %dx%d, want 1x1", w, h)
	}
}

func TestBitWriter(t *testing.T) {
	bw := &bitWriter{}
	bw.writeBits(0x2f, 8)
	bw.flush()

	if got := bw.buf.Bytes(); len(got) != 1 || got[0] != 0x2f {
		t.Fatalf("got %v, want [0x2f]", got)
	}
}

func TestCanonicalCodes(t *testing.T) {
	// Two symbols both with length 1: codes 0 and 1
	lengths := []uint{1, 1}
	codes := canonicalCodes(lengths)
	if codes[0].code != 0 || codes[0].length != 1 {
		t.Fatalf("symbol 0: %+v", codes[0])
	}
	if codes[1].code != 1 || codes[1].length != 1 {
		t.Fatalf("symbol 1: %+v", codes[1])
	}
}

func TestReverseBits(t *testing.T) {
	tests := []struct {
		v    uint32
		n    uint
		want uint32
	}{
		{0, 1, 0},
		{1, 1, 1},
		{0b10, 2, 0b01},
		{0b110, 3, 0b011},
		{0b10110, 5, 0b01101},
	}
	for _, tt := range tests {
		got := reverseBits(tt.v, tt.n)
		if got != tt.want {
			t.Errorf("reverseBits(%d, %d) = %d, want %d", tt.v, tt.n, got, tt.want)
		}
	}
}
