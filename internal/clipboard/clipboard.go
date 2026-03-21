// Package clipboard provides cross-platform clipboard image extraction.
// Supported platforms: macOS, Linux (native + WSL2), Windows.
package clipboard

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/daaa1k/mdp/internal/webpenc"
	_ "golang.org/x/image/webp"
)

// Image holds raw bytes and the file extension for a clipboard image.
type Image struct {
	Data []byte
	Ext  string
}

// GetImages returns all images currently on the clipboard.
// powerShellPath is used on WSL2/Windows to locate powershell.exe; leave empty for auto-detection.
func GetImages(powerShellPath string) ([]Image, error) {
	var imgs []Image
	var err error
	switch runtime.GOOS {
	case "darwin":
		imgs, err = getMacOSImages()
	case "linux":
		if isWSL() {
			imgs, err = getWSL2Images(powerShellPath)
		} else {
			imgs, err = getLinuxImages()
		}
	case "windows":
		imgs, err = getWindowsImages(powerShellPath)
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	if err != nil {
		return nil, err
	}
	return normalizeWebPImages(imgs)
}

// normalizeWebPImages ensures bytes match a .webp extension: if Ext is "webp" but the payload
// is not a WebP file (e.g. PNG bytes under a misnamed path, or file-URL taking precedence over
// a real clipboard image on macOS), re-encode via toWebP.
func normalizeWebPImages(imgs []Image) ([]Image, error) {
	for i := range imgs {
		if strings.ToLower(imgs[i].Ext) != "webp" {
			continue
		}
		if isWebP(imgs[i].Data) {
			continue
		}
		out, err := toWebP(imgs[i].Data)
		if err != nil {
			return nil, fmt.Errorf("clipboard: expected webp bytes: %w", err)
		}
		imgs[i].Data = out
	}
	return imgs, nil
}

func isWebP(data []byte) bool {
	return len(data) >= 12 &&
		bytes.Equal(data[:4], []byte("RIFF")) &&
		bytes.Equal(data[8:12], []byte("WEBP"))
}

// ─── macOS ───────────────────────────────────────────────────────────────────

func getMacOSImages() ([]Image, error) {
	// 1. Try AppleScript FileDrop (files copied from Finder).
	if imgs, err := getMacOSFileDropImages(); err == nil && len(imgs) > 0 {
		return imgs, nil
	}
	// 2. Try pngpaste.
	if imgs, err := getMacOSPngpaste(); err == nil && len(imgs) > 0 {
		return imgs, nil
	}
	// 3. AppleScript fallback (PNG from screen capture).
	return getMacOSAppleScript()
}

func getMacOSFileDropImages() ([]Image, error) {
	// Use JXA (JavaScript for Automation) to access NSPasteboard directly.
	// AppleScript «class furl» only returns the first file; pasteboardItems handles N files.
	// Finder copies files as persistent file-reference URLs (file:///.file/id=...).
	// url.filePathURL resolves these to regular file-path URLs before extracting the path.
	script := `
ObjC.import("AppKit");
var pb = $.NSPasteboard.generalPasteboard;
var items = pb.pasteboardItems;
var result = [];
if (items) {
	var count = items.count;
	for (var i = 0; i < count; i++) {
		var item = items.objectAtIndex(i);
		var urlStr = item.stringForType("public.file-url");
		if (urlStr) {
			var url = $.NSURL.URLWithString($(ObjC.unwrap(urlStr)));
			var pathURL = url.filePathURL;
			if (pathURL) {
				result.push(ObjC.unwrap(pathURL.path));
			}
		}
	}
}
result.join("\n");
`
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", script)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return nil, fmt.Errorf("no file drop")
	}
	var imgs []Image
	for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		ext := strings.TrimPrefix(filepath.Ext(p), ".")
		if ext == "" {
			ext = "png"
		}
		imgs = append(imgs, Image{Data: data, Ext: strings.ToLower(ext)})
	}
	return imgs, nil
}

func getMacOSPngpaste() ([]Image, error) {
	tmp, err := os.CreateTemp("", "mdp-*.png")
	if err != nil {
		return nil, err
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmp.Name()) }()

	if err := exec.Command("pngpaste", tmp.Name()).Run(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, err
	}
	out, err := toWebP(data)
	if err != nil {
		return nil, err
	}
	return []Image{{Data: out, Ext: "webp"}}, nil
}

func getMacOSAppleScript() ([]Image, error) {
	tmp, err := os.CreateTemp("", "mdp-*.png")
	if err != nil {
		return nil, err
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmp.Name()) }()

	script := fmt.Sprintf(`set imgData to the clipboard as «class PNGf»
set f to open for access POSIX file %q with write permission
write imgData to f
close access f`, tmp.Name())

	if _, err := runAppleScript(script); err != nil {
		return nil, fmt.Errorf("applescript PNG save: %w", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("empty clipboard image")
	}
	out, err := toWebP(data)
	if err != nil {
		return nil, err
	}
	return []Image{{Data: out, Ext: "webp"}}, nil
}

func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	return string(out), err
}

// ─── Linux (native) ──────────────────────────────────────────────────────────

func getLinuxImages() ([]Image, error) {
	// 1. Try Wayland wl-paste.
	if imgs, err := getLinuxWayland(); err == nil && len(imgs) > 0 {
		return imgs, nil
	}
	// 2. Try X11 xclip.
	return getLinuxX11()
}

var linuxMIMEs = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

func getLinuxWayland() ([]Image, error) {
	types, err := exec.Command("wl-paste", "--list-types").Output()
	if err != nil {
		return nil, err
	}
	available := string(types)

	if strings.Contains(available, "text/uri-list") {
		if imgs, err := getLinuxURIList("wl-paste", "--type", "text/uri-list"); err == nil && len(imgs) > 0 {
			return imgs, nil
		}
	}

	for _, mime := range linuxMIMEs {
		if !strings.Contains(available, mime) {
			continue
		}
		data, err := exec.Command("wl-paste", "--type", mime).Output()
		if err != nil {
			continue
		}
		img, err := rawToImage(data)
		if err != nil {
			continue
		}
		return []Image{img}, nil
	}
	return nil, fmt.Errorf("no image in wayland clipboard")
}

func getLinuxX11() ([]Image, error) {
	if imgs, err := getLinuxURIList("xclip", "-selection", "clipboard", "-t", "text/uri-list", "-o"); err == nil && len(imgs) > 0 {
		return imgs, nil
	}
	for _, mime := range linuxMIMEs {
		data, err := exec.Command("xclip", "-selection", "clipboard", "-t", mime, "-o").Output()
		if err != nil {
			continue
		}
		img, err := rawToImage(data)
		if err != nil {
			continue
		}
		return []Image{img}, nil
	}
	return nil, fmt.Errorf("no image in X11 clipboard")
}

func getLinuxURIList(args ...string) ([]Image, error) {
	data, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return nil, err
	}
	paths := parseURIList(string(data))
	var imgs []Image
	for _, p := range paths {
		fileData, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		ext := strings.TrimPrefix(filepath.Ext(p), ".")
		if ext == "" {
			ext = "webp"
		}
		imgs = append(imgs, Image{Data: fileData, Ext: strings.ToLower(ext)})
	}
	return imgs, nil
}

// parseURIList converts a text/uri-list to local file paths.
func parseURIList(raw string) []string {
	var paths []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			continue
		}
		if u.Scheme == "file" {
			decoded, err := url.PathUnescape(u.Path)
			if err != nil {
				decoded = u.Path
			}
			paths = append(paths, decoded)
		}
	}
	return paths
}

// ─── WSL2 ────────────────────────────────────────────────────────────────────

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

func getWSL2Images(powerShellPath string) ([]Image, error) {
	ps := resolvePowerShell(powerShellPath)

	// Try file drop list.
	// -Sta is required: System.Windows.Forms.Clipboard needs an STA thread.
	// [Console]::OutputEncoding=UTF8 is required for non-ASCII paths (e.g. Japanese).
	fileScript := `[Console]::OutputEncoding=[Text.Encoding]::UTF8;Add-Type -Assembly System.Windows.Forms;$files=[System.Windows.Forms.Clipboard]::GetFileDropList();if($files -eq $null -or $files.Count -eq 0){exit 1};foreach($f in $files){Write-Output $f}`
	out, err := runPowerShellSTA(ps, fileScript)
	if err == nil && strings.TrimSpace(out) != "" {
		var imgs []Image
		for _, winPath := range strings.Split(strings.TrimSpace(out), "\n") {
			winPath = strings.TrimSpace(winPath)
			if winPath == "" {
				continue
			}
			linuxPath, err := wslPath(winPath)
			if err != nil {
				continue
			}
			data, err := os.ReadFile(linuxPath)
			if err != nil {
				continue
			}
			ext := strings.TrimPrefix(filepath.Ext(linuxPath), ".")
			if ext == "" {
				ext = "webp"
			}
			imgs = append(imgs, Image{Data: data, Ext: strings.ToLower(ext)})
		}
		if len(imgs) > 0 {
			return imgs, nil
		}
	}

	return getPowerShellClipboardImage(ps, "WSL2")
}

// ─── Windows ─────────────────────────────────────────────────────────────────

func getWindowsImages(powerShellPath string) ([]Image, error) {
	ps := resolvePowerShell(powerShellPath)

	fileScript := `[Console]::OutputEncoding=[Text.Encoding]::UTF8;Add-Type -Assembly System.Windows.Forms;$files=[System.Windows.Forms.Clipboard]::GetFileDropList();if($files -eq $null -or $files.Count -eq 0){exit 1};foreach($f in $files){Write-Output $f}`
	out, err := runPowerShellSTA(ps, fileScript)
	if err == nil && strings.TrimSpace(out) != "" {
		var imgs []Image
		for _, p := range strings.Split(strings.TrimSpace(out), "\n") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			ext := strings.TrimPrefix(filepath.Ext(p), ".")
			if ext == "" {
				ext = "webp"
			}
			imgs = append(imgs, Image{Data: data, Ext: strings.ToLower(ext)})
		}
		if len(imgs) > 0 {
			return imgs, nil
		}
	}

	return getPowerShellClipboardImage(ps, "Windows")
}

// getPowerShellClipboardImage reads the clipboard image via PowerShell and re-encodes it as WebP.
// platform is used only in the error message.
func getPowerShellClipboardImage(ps, platform string) ([]Image, error) {
	imgScript := `Add-Type -AssemblyName System.Windows.Forms
$img = [System.Windows.Forms.Clipboard]::GetImage()
if ($img -ne $null) {
    $ms = New-Object System.IO.MemoryStream
    $img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    [Convert]::ToBase64String($ms.ToArray())
}`
	b64, err := runPowerShell(ps, imgScript)
	if err != nil || strings.TrimSpace(b64) == "" {
		return nil, fmt.Errorf("no image in %s clipboard", platform)
	}
	pngData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, err
	}
	webpData, err := toWebP(pngData)
	if err != nil {
		return nil, err
	}
	return []Image{{Data: webpData, Ext: "webp"}}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func resolvePowerShell(configured string) string {
	if configured != "" {
		return configured
	}
	candidates := []string{
		"powershell.exe",
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
		"/mnt/c/Program Files/PowerShell/7/pwsh.exe",
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "powershell.exe"
}

func runPowerShell(psPath, script string) (string, error) {
	cmd := exec.Command(psPath, "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	return string(out), err
}

// runPowerShellSTA runs a PowerShell script in STA mode.
// STA (Single-Threaded Apartment) is required by System.Windows.Forms.Clipboard.
// Note: -Sta is only supported by powershell.exe (Windows PowerShell), not pwsh.exe (PowerShell Core).
func runPowerShellSTA(psPath, script string) (string, error) {
	cmd := exec.Command(psPath, "-Sta", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	return string(out), err
}

func wslPath(winPath string) (string, error) {
	out, err := exec.Command("wslpath", "-u", strings.TrimSpace(winPath)).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// rawToImage converts raw image bytes to a clipboard Image, encoding as WebP (lossless).
func rawToImage(data []byte) (Image, error) {
	webpData, err := toWebP(data)
	if err != nil {
		return Image{}, err
	}
	return Image{Data: webpData, Ext: "webp"}, nil
}

// toWebP decodes any supported image and re-encodes as WebP (VP8L lossless).
func toWebP(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	var buf bytes.Buffer
	if err := webpenc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return buf.Bytes(), nil
}
