// WSL2 clipboard tests.
//
// Unit tests (TestIsWSL, TestResolvePowerShell_*) always run.
// Integration tests (TestGetWSL2Images_*) require an actual WSL2 environment
// with PowerShell accessible; they are skipped automatically otherwise.
package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ─── unit tests ───────────────────────────────────────────────────────────────

func TestIsWSL_DoesNotPanic(t *testing.T) {
	// isWSL reads /proc/version; just ensure it never panics.
	_ = isWSL()
}

func TestResolvePowerShell_PassThrough(t *testing.T) {
	// A non-empty configured path must be returned as-is.
	got := resolvePowerShell("/custom/powershell.exe")
	if got != "/custom/powershell.exe" {
		t.Errorf("got %q, want /custom/powershell.exe", got)
	}
}

func TestResolvePowerShell_FallbackNonEmpty(t *testing.T) {
	// When no path is configured the function must return something non-empty.
	got := resolvePowerShell("")
	if got == "" {
		t.Error("resolvePowerShell(\"\") returned empty string")
	}
}

// ─── WSL2 integration helpers ────────────────────────────────────────────────

// skipIfNotWSL2 skips the test when not running inside WSL2 or when
// PowerShell is not reachable.
func skipIfNotWSL2(t *testing.T) {
	t.Helper()
	if !isWSL() {
		t.Skip("not running in WSL2")
	}
	ps := resolvePowerShell("")
	if err := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", "exit 0").Run(); err != nil {
		t.Skipf("PowerShell not available (%s): %v", ps, err)
	}
}

// windowsPath converts a Linux path to the Windows UNC path via wslpath.
func windowsPath(t *testing.T, linuxPath string) string {
	t.Helper()
	out, err := exec.Command("wslpath", "-w", linuxPath).Output()
	if err != nil {
		t.Fatalf("wslpath -w %q: %v", linuxPath, err)
	}
	return strings.TrimSpace(string(out))
}

// psQuote wraps a path in PowerShell single quotes, escaping any embedded
// single quotes (” is the PowerShell escape sequence).
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// setWSL2Image puts a small red bitmap on the Windows clipboard as an image.
func setWSL2Image(t *testing.T, ps string) {
	t.Helper()
	script := `
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms
$bmp = New-Object System.Drawing.Bitmap(4, 4)
$g   = [System.Drawing.Graphics]::FromImage($bmp)
$g.Clear([System.Drawing.Color]::Red)
$g.Dispose()
[System.Windows.Forms.Clipboard]::SetImage($bmp)
$bmp.Dispose()
`
	if _, err := runPowerShell(ps, script); err != nil {
		t.Fatalf("setWSL2Image: %v", err)
	}
}

// setWSL2FileDrop puts the given Windows paths on the Windows clipboard as a
// file-drop list.
func setWSL2FileDrop(t *testing.T, ps string, winPaths []string) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("Add-Type -AssemblyName System.Windows.Forms\n")
	sb.WriteString("$f = New-Object System.Collections.Specialized.StringCollection\n")
	for _, p := range winPaths {
		fmt.Fprintf(&sb, "$f.Add(%s)\n", psQuote(p))
	}
	sb.WriteString("[System.Windows.Forms.Clipboard]::SetFileDropList($f)\n")
	if _, err := runPowerShell(ps, sb.String()); err != nil {
		t.Fatalf("setWSL2FileDrop: %v", err)
	}
}

// ─── WSL2 integration tests ───────────────────────────────────────────────────

// TestGetWSL2Images_Image copies a bitmap to the Windows clipboard and verifies
// that getWSL2Images returns exactly one image.
func TestGetWSL2Images_Image(t *testing.T) {
	skipIfNotWSL2(t)
	ps := resolvePowerShell("")

	setWSL2Image(t, ps)

	imgs, err := getWSL2Images(ps)
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

// TestGetWSL2Images_FileDropSingle copies a single file path to the Windows
// clipboard as a file-drop and verifies that getWSL2Images reads it.
func TestGetWSL2Images_FileDropSingle(t *testing.T) {
	skipIfNotWSL2(t)
	ps := resolvePowerShell("")

	linuxPath := makeTestPNGFile(t)
	defer os.Remove(linuxPath)
	winPath := windowsPath(t, linuxPath)

	setWSL2FileDrop(t, ps, []string{winPath})

	imgs, err := getWSL2Images(ps)
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

// TestGetWSL2Images_FileDropNonASCII verifies that getWSL2Images handles
// Windows paths containing non-ASCII characters (e.g. Japanese) correctly.
// This exercises the [Console]::OutputEncoding=UTF8 fix: without it,
// PowerShell outputs paths in the system OEM code page (e.g. Shift-JIS on
// Japanese Windows), which corrupts the path before wslpath sees it.
func TestGetWSL2Images_FileDropNonASCII(t *testing.T) {
	skipIfNotWSL2(t)
	ps := resolvePowerShell("")

	// Create a temp dir with a Japanese name so the full path is non-ASCII.
	dir, err := os.MkdirTemp("", "mdp-テスト-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)

	linuxPath := filepath.Join(dir, "画像.png")
	if err := os.WriteFile(linuxPath, makeTestPNG(t), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	winPath := windowsPath(t, linuxPath)

	setWSL2FileDrop(t, ps, []string{winPath})

	imgs, err := getWSL2Images(ps)
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

// TestGetWSL2Images_FileDropMultiple copies multiple file paths to the Windows
// clipboard and verifies that getWSL2Images returns all of them.
func TestGetWSL2Images_FileDropMultiple(t *testing.T) {
	skipIfNotWSL2(t)
	ps := resolvePowerShell("")

	linuxPath1 := makeTestPNGFile(t)
	linuxPath2 := makeTestPNGFile(t)
	defer os.Remove(linuxPath1)
	defer os.Remove(linuxPath2)

	winPath1 := windowsPath(t, linuxPath1)
	winPath2 := windowsPath(t, linuxPath2)

	setWSL2FileDrop(t, ps, []string{winPath1, winPath2})

	imgs, err := getWSL2Images(ps)
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
