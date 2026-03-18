package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daaa1k/mdp/internal/xdg"
)

// NodeBBBackend uploads images to a NodeBB forum instance.
type NodeBBBackend struct {
	baseURL    string
	client     *http.Client
	jar        *cookiejar.Jar
	cookieFile string
}

// NewNodeBBBackend creates a NodeBBBackend for the given base URL.
func NewNodeBBBackend(baseURL string) (*NodeBBBackend, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	b := &NodeBBBackend{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		jar:        jar,
		cookieFile: nodeBBCookieFile(),
	}
	// Load persisted cookies (best-effort).
	_ = b.loadCookies()
	return b, nil
}

// apiConfig holds the fields we read from NodeBB's /api/config endpoint.
type apiConfig struct {
	UID       float64 `json:"uid"`
	CSRFToken string  `json:"csrf_token"`
}

// fetchAPIConfig calls /api/config and returns uid + csrf_token.
func (b *NodeBBBackend) fetchAPIConfig(ctx context.Context) (*apiConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/api/config", nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var cfg apiConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save authenticates (if needed) and uploads an image to NodeBB, returning the URL.
func (b *NodeBBBackend) Save(ctx context.Context, data []byte, filename string) (string, error) {
	cfg, err := b.fetchAPIConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("get api config: %w", err)
	}

	if cfg.UID == 0 {
		if err := b.login(ctx, cfg.CSRFToken); err != nil {
			return "", fmt.Errorf("nodebb login: %w", err)
		}
		_ = b.saveCookies()
		// Re-fetch to get fresh CSRF token for the upload.
		cfg, err = b.fetchAPIConfig(ctx)
		if err != nil {
			return "", fmt.Errorf("get api config after login: %w", err)
		}
	}

	imageURL, err := b.upload(ctx, data, filename, cfg.CSRFToken)
	if err != nil {
		return "", fmt.Errorf("nodebb upload: %w", err)
	}
	return imageURL, nil
}

// login authenticates using NODEBB_USERNAME and NODEBB_PASSWORD env vars.
func (b *NodeBBBackend) login(ctx context.Context, csrfToken string) error {
	username := os.Getenv("NODEBB_USERNAME")
	password := os.Getenv("NODEBB_PASSWORD")
	if username == "" || password == "" {
		return fmt.Errorf("NODEBB_USERNAME and NODEBB_PASSWORD must be set")
	}

	form := url.Values{
		"username": {username},
		"password": {password},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/api/v3/utilities/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-csrf-token", csrfToken)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (%d): %s", resp.StatusCode, body)
	}
	return nil
}

// upload sends the image as a multipart form to /api/post/upload.
func (b *NodeBBBackend) upload(ctx context.Context, data []byte, filename, csrfToken string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("files[]", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/api/post/upload", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("x-csrf-token", csrfToken)

	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, body)
	}

	return parseUploadResponse(body, b.baseURL)
}

// parseUploadResponse handles both NodeBB v3 and legacy response formats.
func parseUploadResponse(body []byte, baseURL string) (string, error) {
	// Try v3 format: {"response": {"images": [{"url": "..."}]}}
	var v3 struct {
		Response struct {
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &v3); err == nil {
		if len(v3.Response.Images) > 0 && v3.Response.Images[0].URL != "" {
			return resolveURL(v3.Response.Images[0].URL, baseURL), nil
		}
	}

	// Try legacy format: {"images": [{"url": "..."}]}
	var legacy struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.Unmarshal(body, &legacy); err == nil {
		if len(legacy.Images) > 0 && legacy.Images[0].URL != "" {
			return resolveURL(legacy.Images[0].URL, baseURL), nil
		}
	}

	return "", fmt.Errorf("could not parse upload response: %s", body)
}

// resolveURL makes a relative URL absolute using the base URL.
func resolveURL(imageURL, baseURL string) string {
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + imageURL
	}
	ref, err := url.Parse(imageURL)
	if err != nil {
		return baseURL + imageURL
	}
	return base.ResolveReference(ref).String()
}

// nodeBBCookieFile returns the XDG-compliant cache file path for cookies.
func nodeBBCookieFile() string {
	dir, err := xdg.CacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "mdp", "nodebb_cookies.json")
}

type cookieEntry struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Domain  string `json:"domain"`
	Path    string `json:"path"`
	Secure  bool   `json:"secure"`
	Expires int64  `json:"expires"`
}

// saveCookies persists session cookies to disk.
func (b *NodeBBBackend) saveCookies() error {
	if b.cookieFile == "" {
		return nil
	}
	u, err := url.Parse(b.baseURL)
	if err != nil {
		return err
	}
	cookies := b.jar.Cookies(u)
	entries := make([]cookieEntry, 0, len(cookies))
	for _, c := range cookies {
		e := cookieEntry{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
			Secure: c.Secure,
		}
		if !c.Expires.IsZero() {
			e.Expires = c.Expires.Unix()
		}
		entries = append(entries, e)
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(b.cookieFile), 0o700); err != nil {
		return err
	}
	return os.WriteFile(b.cookieFile, data, 0o600)
}

// loadCookies restores persisted cookies from disk.
func (b *NodeBBBackend) loadCookies() error {
	if b.cookieFile == "" {
		return nil
	}
	data, err := os.ReadFile(b.cookieFile)
	if err != nil {
		return err
	}
	var entries []cookieEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	u, err := url.Parse(b.baseURL)
	if err != nil {
		return err
	}
	cookies := make([]*http.Cookie, 0, len(entries))
	for _, e := range entries {
		c := &http.Cookie{
			Name:   e.Name,
			Value:  e.Value,
			Domain: e.Domain,
			Path:   e.Path,
			Secure: e.Secure,
		}
		if e.Expires != 0 {
			c.Expires = time.Unix(e.Expires, 0)
		}
		cookies = append(cookies, c)
	}
	b.jar.SetCookies(u, cookies)
	return nil
}
