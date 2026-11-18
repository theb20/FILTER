package domain

import "testing"

func TestAdresseDImage(t *testing.T) {
	for u, ok := range map[string]bool{
		"/v1/files/0123456789abcdef0123456789abcdef.jpg": true, "https://cdn.exemple.com/a.png": true,
		"http://exemple.com/a.png": false, "javascript:alert(1)": false, "data:image/png;base64,AAA": false, "//evil.com/a.png": false, "/v1/files/../secret": false, "": false,
	} {
		if mediaURL(u) != ok {
			t.Errorf("mediaURL(%q) = %v", u, !ok)
		}
	}
}
