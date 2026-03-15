package backend

import "context"

// Backend is the interface implemented by all storage backends.
type Backend interface {
	// Save stores data under the given filename and returns the URL or path to
	// the stored image.
	Save(ctx context.Context, data []byte, filename string) (string, error)
}
