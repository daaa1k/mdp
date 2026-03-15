package markdown

import "testing"

func TestGenerate(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://example.com/image.webp", "![](https://example.com/image.webp)"},
		{"./images/20260312_120233.webp", "![](./images/20260312_120233.webp)"},
		{"", "![]()"},
		{"file with spaces.webp", "![](file with spaces.webp)"},
	}
	for _, c := range cases {
		got := Generate(c.url)
		if got != c.want {
			t.Errorf("Generate(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}
