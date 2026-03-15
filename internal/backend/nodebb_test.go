package backend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── parseUploadResponse ──────────────────────────────────────────────────────

func TestParseUploadResponse_V3Format(t *testing.T) {
	body := []byte(`{"response":{"images":[{"url":"/assets/uploads/img.png"}]}}`)
	url, err := parseUploadResponse(body, "https://forum.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://forum.example.com/assets/uploads/img.png" {
		t.Errorf("got %s", url)
	}
}

func TestParseUploadResponse_LegacyFormat(t *testing.T) {
	body := []byte(`{"images":[{"url":"https://cdn.example.com/img.png"}]}`)
	url, err := parseUploadResponse(body, "https://forum.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://cdn.example.com/img.png" {
		t.Errorf("got %s", url)
	}
}

func TestParseUploadResponse_AbsoluteURL(t *testing.T) {
	body := []byte(`{"response":{"images":[{"url":"https://cdn.example.com/img.png"}]}}`)
	url, err := parseUploadResponse(body, "https://forum.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://cdn.example.com/img.png" {
		t.Errorf("got %s", url)
	}
}

func TestParseUploadResponse_Invalid(t *testing.T) {
	_, err := parseUploadResponse([]byte(`{"status":"ok"}`), "https://forum.example.com")
	if err == nil {
		t.Error("expected error for response with no image URL")
	}
}

// ─── resolveURL ───────────────────────────────────────────────────────────────

func TestResolveURL_AlreadyAbsolute(t *testing.T) {
	got := resolveURL("https://cdn.example.com/img.png", "https://forum.example.com")
	if got != "https://cdn.example.com/img.png" {
		t.Errorf("got %s", got)
	}
}

func TestResolveURL_RelativePath(t *testing.T) {
	got := resolveURL("/assets/img.png", "https://forum.example.com")
	if got != "https://forum.example.com/assets/img.png" {
		t.Errorf("got %s", got)
	}
}

// ─── NodeBBBackend integration with mock server ────────────────────────────────

// nodeBBServer builds a test HTTP server that mimics the NodeBB API.
// If loggedIn is true, /api/config reports uid > 0 from the first call.
func nodeBBServer(t *testing.T, loggedIn bool) (*httptest.Server, *struct{ LoginCalls, UploadCalls int }) {
	t.Helper()
	counters := &struct{ LoginCalls, UploadCalls int }{}
	configCallCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/config":
			configCallCount++
			uid := 0.0
			if loggedIn || configCallCount > 1 {
				uid = 42.0
			}
			json.NewEncoder(w).Encode(map[string]any{
				"uid":        uid,
				"csrf_token": "csrf-token-test",
			})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/utilities/login":
			counters.LoginCalls++
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPost && r.URL.Path == "/api/post/upload":
			counters.UploadCalls++
			// Drain the multipart body.
			io.Copy(io.Discard, r.Body)
			json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"images": []any{
						map[string]any{"url": "/assets/uploads/result.png"},
					},
				},
			})

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, counters
}

// TestNodeBBBackend_Save_SessionValid tests upload when already authenticated.
func TestNodeBBBackend_Save_SessionValid(t *testing.T) {
	srv, counters := nodeBBServer(t, true)

	b, err := NewNodeBBBackend(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	b.cookieFile = "" // prevent disk I/O during tests

	url, err := b.Save(context.Background(), []byte("fake png"), "test.png")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if counters.LoginCalls != 0 {
		t.Errorf("expected no login calls when session is valid, got %d", counters.LoginCalls)
	}
	if counters.UploadCalls != 1 {
		t.Errorf("expected 1 upload call, got %d", counters.UploadCalls)
	}
	want := srv.URL + "/assets/uploads/result.png"
	if url != want {
		t.Errorf("expected URL %s, got %s", want, url)
	}
}

// TestNodeBBBackend_Save_SessionExpired tests that login is triggered when
// the session is not valid (uid == 0).
func TestNodeBBBackend_Save_SessionExpired(t *testing.T) {
	srv, counters := nodeBBServer(t, false)

	t.Setenv("NODEBB_USERNAME", "testuser")
	t.Setenv("NODEBB_PASSWORD", "testpass")

	b, err := NewNodeBBBackend(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	b.cookieFile = ""

	_, err = b.Save(context.Background(), []byte("fake png"), "test.png")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if counters.LoginCalls != 1 {
		t.Errorf("expected 1 login call, got %d", counters.LoginCalls)
	}
	if counters.UploadCalls != 1 {
		t.Errorf("expected 1 upload call, got %d", counters.UploadCalls)
	}
}

// TestNodeBBBackend_Save_MissingCredentials verifies that a missing username/password
// returns an error rather than hanging or panicking.
func TestNodeBBBackend_Save_MissingCredentials(t *testing.T) {
	srv, _ := nodeBBServer(t, false)

	t.Setenv("NODEBB_USERNAME", "")
	t.Setenv("NODEBB_PASSWORD", "")

	b, err := NewNodeBBBackend(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	b.cookieFile = ""

	_, err = b.Save(context.Background(), []byte("data"), "img.png")
	if err == nil {
		t.Error("expected error when NODEBB_USERNAME/PASSWORD are not set")
	}
}

// TestNodeBBBackend_Save_UploadError verifies that a non-2xx upload response returns an error.
func TestNodeBBBackend_Save_UploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/config":
			json.NewEncoder(w).Encode(map[string]any{
				"uid":        42.0,
				"csrf_token": "csrf",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/post/upload":
			io.Copy(io.Discard, r.Body)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b, err := NewNodeBBBackend(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	b.cookieFile = ""

	_, err = b.Save(context.Background(), []byte("data"), "img.png")
	if err == nil {
		t.Error("expected error for upload failure")
	}
}
