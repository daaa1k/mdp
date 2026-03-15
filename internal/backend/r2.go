package backend

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Backend uploads images to Cloudflare R2 via the S3-compatible API.
type R2Backend struct {
	client    *s3.Client
	Bucket    string
	PublicURL string
	Prefix    string
}

// NewR2Backend creates an R2Backend using credentials from environment variables
// R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY.
func NewR2Backend(bucket, publicURL, endpoint, prefix string) (*R2Backend, error) {
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY must be set")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     "auto",
				HostnameImmutable: true,
			}, nil
		},
	)

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("load R2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &R2Backend{
		client:    client,
		Bucket:    bucket,
		PublicURL: publicURL,
		Prefix:    prefix,
	}, nil
}

// Save uploads data to R2 and returns the public URL of the uploaded object.
func (b *R2Backend) Save(data []byte, filename string) (string, error) {
	key := filename
	if b.Prefix != "" {
		key = path.Join(b.Prefix, filename)
	}

	contentType := mimeType(filename)

	_, err := b.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(b.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("upload to R2: %w", err)
	}

	url := b.PublicURL + "/" + key
	return url, nil
}

// mimeType returns the MIME type for a filename based on its extension.
func mimeType(filename string) string {
	ext := path.Ext(filename)
	switch ext {
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
