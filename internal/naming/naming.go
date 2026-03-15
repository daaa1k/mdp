package naming

import (
	"fmt"
	"time"
)

// Generate returns a timestamp-based filename like "20260312_120233.webp".
func Generate(ext string) string {
	return time.Now().Local().Format("20060102_150405") + "." + ext
}

// GenerateN returns a timestamp-based filename with a sequence number like "20260312_120233_2.webp".
// n is 1-indexed; index 1 uses the plain format, 2+ appends the sequence number.
func GenerateN(n int, ext string) string {
	ts := time.Now().Local().Format("20060102_150405")
	if n <= 1 {
		return ts + "." + ext
	}
	return fmt.Sprintf("%s_%d.%s", ts, n, ext)
}
