package clipboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// makeTestPNG returns bytes of a minimal valid 4x4 red PNG image.
func makeTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// makeTestPNGFile writes a test PNG to a temporary file and returns its path.
// The caller must defer os.Remove on the returned path.
func makeTestPNGFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "mdp-test-*.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(makeTestPNG(t)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// ─── parseURIList ─────────────────────────────────────────────────────────────

func TestParseURIList_SingleFile(t *testing.T) {
	got := parseURIList("file:///tmp/image.png\r\n")
	if len(got) != 1 || got[0] != "/tmp/image.png" {
		t.Errorf("got %v, want [/tmp/image.png]", got)
	}
}

func TestParseURIList_MultipleFiles(t *testing.T) {
	got := parseURIList("file:///tmp/a.png\r\nfile:///tmp/b.jpg\r\n")
	if len(got) != 2 || got[0] != "/tmp/a.png" || got[1] != "/tmp/b.jpg" {
		t.Errorf("got %v, want [/tmp/a.png /tmp/b.jpg]", got)
	}
}

func TestParseURIList_CommentsIgnored(t *testing.T) {
	got := parseURIList("# comment\r\nfile:///tmp/image.png\r\n")
	if len(got) != 1 || got[0] != "/tmp/image.png" {
		t.Errorf("got %v, want [/tmp/image.png]", got)
	}
}

func TestParseURIList_URLEncodedPath(t *testing.T) {
	got := parseURIList("file:///tmp/my%20image.png\r\n")
	if len(got) != 1 || got[0] != "/tmp/my image.png" {
		t.Errorf("got %v, want [/tmp/my image.png]", got)
	}
}

func TestParseURIList_SkipsNonFileURIs(t *testing.T) {
	got := parseURIList("https://example.com/img.png\r\nfile:///tmp/local.png\r\n")
	if len(got) != 1 || got[0] != "/tmp/local.png" {
		t.Errorf("got %v, want [/tmp/local.png]", got)
	}
}

func TestParseURIList_EmptyInput(t *testing.T) {
	got := parseURIList("")
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

func TestParseURIList_OnlyComments(t *testing.T) {
	got := parseURIList("# comment 1\r\n# comment 2\r\n")
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

// ─── toWebP ───────────────────────────────────────────────────────────────────

func TestToWebP_ReencodesPNG(t *testing.T) {
	data := makeTestPNG(t)
	out, err := toWebP(data)
	if err != nil {
		t.Fatal(err)
	}
	// Output should be a valid PNG (the function re-encodes as PNG despite the name).
	_, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not a valid image: %v", err)
	}
	if format != "png" {
		t.Errorf("expected png output, got %s", format)
	}
}

func TestToWebP_InvalidData(t *testing.T) {
	_, err := toWebP([]byte("not an image"))
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

func TestToWebP_EmptyData(t *testing.T) {
	_, err := toWebP([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

// ─── rawToImage ───────────────────────────────────────────────────────────────

func TestRawToImage_PNG(t *testing.T) {
	img, err := rawToImage(makeTestPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if img.Ext != "webp" {
		t.Errorf("expected ext webp, got %s", img.Ext)
	}
	if len(img.Data) == 0 {
		t.Error("expected non-empty image data")
	}
	// Output must decode as PNG.
	_, format, err := image.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil {
		t.Fatalf("output is not a valid image: %v", err)
	}
	if format != "png" {
		t.Errorf("expected png-encoded output, got %s", format)
	}
}

func TestRawToImage_InvalidData(t *testing.T) {
	_, err := rawToImage([]byte("garbage"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}
