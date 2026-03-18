package backend

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// R2Backend uploads images to Cloudflare R2 via the S3-compatible API.
type R2Backend struct {
	endpoint  string
	bucket    string
	publicURL string
	prefix    string
	accessKey string
	secretKey string
}

// NewR2Backend creates an R2Backend using credentials from environment variables
// R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY.
func NewR2Backend(bucket, publicURL, endpoint, prefix string) (*R2Backend, error) {
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY must be set")
	}

	return &R2Backend{
		endpoint:  strings.TrimRight(endpoint, "/"),
		bucket:    bucket,
		publicURL: publicURL,
		prefix:    prefix,
		accessKey: accessKey,
		secretKey: secretKey,
	}, nil
}

// Save uploads data to R2 and returns the public URL of the uploaded object.
func (b *R2Backend) Save(ctx context.Context, data []byte, filename string) (string, error) {
	key := filename
	if b.prefix != "" {
		key = path.Join(b.prefix, filename)
	}

	if err := b.putObject(ctx, key, data, mimeType(filename)); err != nil {
		return "", fmt.Errorf("upload to R2: %w", err)
	}

	return strings.TrimRight(b.publicURL, "/") + "/" + key, nil
}

// putObject sends a single S3 PutObject request signed with AWS Signature Version 4.
// Only path-style addressing is used, matching Cloudflare R2 requirements.
func (b *R2Backend) putObject(ctx context.Context, key string, body []byte, contentType string) error {
	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	dateTimeStr := now.Format("20060102T150405Z")

	rawURL := b.endpoint + "/" + b.bucket + "/" + key
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	bodyHashBytes := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(bodyHashBytes[:])

	// Canonical headers must be sorted alphabetically by header name.
	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + u.Host + "\n" +
		"x-amz-content-sha256:" + bodyHash + "\n" +
		"x-amz-date:" + dateTimeStr + "\n"
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := "PUT\n" +
		u.EscapedPath() + "\n" +
		"\n" + // query string (empty)
		canonicalHeaders + "\n" +
		signedHeaders + "\n" +
		bodyHash

	region := "auto" // Cloudflare R2 uses "auto" as the region
	credentialScope := dateStr + "/" + region + "/s3/aws4_request"

	crHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := "AWS4-HMAC-SHA256\n" +
		dateTimeStr + "\n" +
		credentialScope + "\n" +
		hex.EncodeToString(crHash[:])

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256([]byte("AWS4"+b.secretKey), []byte(dateStr)),
				[]byte(region)),
			[]byte("s3")),
		[]byte("aws4_request"))

	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))
	authHeader := "AWS4-HMAC-SHA256 Credential=" + b.accessKey + "/" + credentialScope +
		",SignedHeaders=" + signedHeaders +
		",Signature=" + signature

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Content-Sha256", bodyHash)
	req.Header.Set("X-Amz-Date", dateTimeStr)
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("R2 upload failed: %s", resp.Status)
	}
	return nil
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// mimeType returns the MIME type for a filename based on its extension.
func mimeType(filename string) string {
	switch path.Ext(filename) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	default:
		return "image/webp"
	}
}
