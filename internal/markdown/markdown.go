package markdown

import "fmt"

// Generate wraps a URL in Markdown image syntax: ![](url)
func Generate(url string) string {
	return fmt.Sprintf("![](%s)", url)
}
