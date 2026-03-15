package backend

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestR2Backend_Save verifies that Save uploads data to the correct path
// and returns the expected public URL.
func TestR2Backend_Save(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotContentType = r.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("ETag", `"test-etag"`)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	t.Setenv("R2_ACCESS_KEY_ID", "test-key-id")
	t.Setenv("R2_SECRET_ACCESS_KEY", "test-secret-key")

	b, err := NewR2Backend("my-bucket", "https://cdn.example.com", srv.URL, "")
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("fake png bytes")
	url, err := b.Save(context.Background(), data, "20240101_120000.png")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT request, got %s", gotMethod)
	}
	// Path-style URL: /bucket/key
	if !strings.HasSuffix(gotPath, "20240101_120000.png") {
		t.Errorf("unexpected request path: %s", gotPath)
	}
	if gotContentType != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", gotContentType)
	}
	if string(gotBody) != string(data) {
		t.Error("uploaded body mismatch")
	}
	if url != "https://cdn.example.com/20240101_120000.png" {
		t.Errorf("unexpected URL: %s", url)
	}
}

// TestR2Backend_SaveWithPrefix verifies that a configured prefix is prepended to the key.
func TestR2Backend_SaveWithPrefix(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			gotPath = r.URL.Path
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"test-etag"`)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	t.Setenv("R2_ACCESS_KEY_ID", "test-key-id")
	t.Setenv("R2_SECRET_ACCESS_KEY", "test-secret-key")

	b, err := NewR2Backend("my-bucket", "https://cdn.example.com", srv.URL, "uploads/images")
	if err != nil {
		t.Fatal(err)
	}

	url, err := b.Save(context.Background(), []byte("data"), "test.png")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !strings.HasSuffix(gotPath, "uploads/images/test.png") {
		t.Errorf("unexpected request path: %s", gotPath)
	}
	if url != "https://cdn.example.com/uploads/images/test.png" {
		t.Errorf("unexpected URL: %s", url)
	}
}

// TestR2Backend_MimeTypes verifies that content type is derived from file extension.
func TestR2Backend_MimeTypes(t *testing.T) {
	cases := []struct {
		filename string
		wantMime string
	}{
		{"img.png", "image/png"},
		{"img.jpg", "image/jpeg"},
		{"img.jpeg", "image/jpeg"},
		{"img.gif", "image/gif"},
		{"img.webp", "image/webp"},
	}
	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			got := mimeType(tc.filename)
			if got != tc.wantMime {
				t.Errorf("mimeType(%q) = %q, want %q", tc.filename, got, tc.wantMime)
			}
		})
	}
}

// TestR2Backend_MissingCredentials verifies that missing env vars return an error.
func TestR2Backend_MissingCredentials(t *testing.T) {
	t.Setenv("R2_ACCESS_KEY_ID", "")
	t.Setenv("R2_SECRET_ACCESS_KEY", "")

	_, err := NewR2Backend("bucket", "https://cdn.example.com", "https://r2.example.com", "")
	if err == nil {
		t.Error("expected error when credentials are missing")
	}
}
