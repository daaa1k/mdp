// Integration tests for clipboard image reading on Linux (Wayland and X11).
// Tests are skipped automatically when the required display server and tools
// are not available, so they run safely in any environment.
package clipboard

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ─── Wayland helpers ──────────────────────────────────────────────────────────

func skipIfNoWayland(t *testing.T) {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("WAYLAND_DISPLAY not set")
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		t.Skip("wl-copy not found in PATH")
	}
	if _, err := exec.LookPath("wl-paste"); err != nil {
		t.Skip("wl-paste not found in PATH")
	}
}

// setWaylandClipboard writes data to the Wayland clipboard with the given MIME type.
func setWaylandClipboard(t *testing.T, data []byte, mime string) {
	t.Helper()
	cmd := exec.Command("wl-copy", "--type", mime)
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		t.Fatalf("wl-copy --type %s: %v", mime, err)
	}
}

// ─── Wayland: image ──────────────────────────────────────────────────────────

func TestLinuxWayland_Image(t *testing.T) {
	skipIfNoWayland(t)

	setWaylandClipboard(t, makeTestPNG(t), "image/png")

	imgs, err := getLinuxWayland()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgs))
	}
	if imgs[0].Ext != "webp" {
		t.Errorf("expected ext webp, got %s", imgs[0].Ext)
	}
	if len(imgs[0].Data) == 0 {
		t.Error("image data is empty")
	}
}

// ─── Wayland: FileDrop single ─────────────────────────────────────────────────

func TestLinuxWayland_FileDropSingle(t *testing.T) {
	skipIfNoWayland(t)

	path := makeTestPNGFile(t)
	defer os.Remove(path)

	uriList := "file://" + path + "\r\n"
	setWaylandClipboard(t, []byte(uriList), "text/uri-list")

	imgs, err := getLinuxWayland()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgs))
	}
	if imgs[0].Ext != "png" {
		t.Errorf("expected ext png, got %s", imgs[0].Ext)
	}
	if len(imgs[0].Data) == 0 {
		t.Error("image data is empty")
	}
}

// ─── Wayland: FileDrop multiple ───────────────────────────────────────────────

func TestLinuxWayland_FileDropMultiple(t *testing.T) {
	skipIfNoWayland(t)

	path1 := makeTestPNGFile(t)
	path2 := makeTestPNGFile(t)
	defer os.Remove(path1)
	defer os.Remove(path2)

	uriList := "file://" + path1 + "\r\nfile://" + path2 + "\r\n"
	setWaylandClipboard(t, []byte(uriList), "text/uri-list")

	imgs, err := getLinuxWayland()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 2 {
		t.Fatalf("expected 2 images, got %d", len(imgs))
	}
	for i, img := range imgs {
		if img.Ext != "png" {
			t.Errorf("imgs[%d]: expected ext png, got %s", i, img.Ext)
		}
		if len(img.Data) == 0 {
			t.Errorf("imgs[%d]: image data is empty", i)
		}
	}
}

// ─── X11 helpers ─────────────────────────────────────────────────────────────

func skipIfNoX11(t *testing.T) {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY not set")
	}
	if _, err := exec.LookPath("xclip"); err != nil {
		t.Skip("xclip not found in PATH")
	}
}

// setX11Clipboard writes data to the X11 clipboard with the given MIME type.
func setX11Clipboard(t *testing.T, data []byte, mime string) {
	t.Helper()
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", mime)
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		t.Fatalf("xclip -t %s: %v", mime, err)
	}
}

// ─── X11: image ──────────────────────────────────────────────────────────────

func TestLinuxX11_Image(t *testing.T) {
	skipIfNoX11(t)

	setX11Clipboard(t, makeTestPNG(t), "image/png")

	imgs, err := getLinuxX11()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgs))
	}
	if imgs[0].Ext != "webp" {
		t.Errorf("expected ext webp, got %s", imgs[0].Ext)
	}
	if len(imgs[0].Data) == 0 {
		t.Error("image data is empty")
	}
}

// ─── X11: FileDrop single ─────────────────────────────────────────────────────

func TestLinuxX11_FileDropSingle(t *testing.T) {
	skipIfNoX11(t)

	path := makeTestPNGFile(t)
	defer os.Remove(path)

	uriList := "file://" + path + "\r\n"
	setX11Clipboard(t, []byte(uriList), "text/uri-list")

	imgs, err := getLinuxX11()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgs))
	}
	if imgs[0].Ext != "png" {
		t.Errorf("expected ext png, got %s", imgs[0].Ext)
	}
}

// ─── X11: FileDrop multiple ───────────────────────────────────────────────────

func TestLinuxX11_FileDropMultiple(t *testing.T) {
	skipIfNoX11(t)

	path1 := makeTestPNGFile(t)
	path2 := makeTestPNGFile(t)
	defer os.Remove(path1)
	defer os.Remove(path2)

	uriList := strings.Join([]string{
		"file://" + path1,
		"file://" + path2,
		"",
	}, "\r\n")
	setX11Clipboard(t, []byte(uriList), "text/uri-list")

	imgs, err := getLinuxX11()
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 2 {
		t.Fatalf("expected 2 images, got %d", len(imgs))
	}
}
