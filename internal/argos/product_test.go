package argos_test

import (
	"testing"

	"github.com/tokar821/argos-availability-scraper/internal/argos"
)

func TestResolveProductID(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"12345678", "12345678", true},
		{"9876543", "9876543", true},
		{"https://www.argos.co.uk/product/12345678", "12345678", true},
		{"https://www.argos.co.uk/product/9876543?ref=abc", "9876543", true},
		{"www.argos.co.uk/product/12345678", "12345678", true},
		{"", "", false},
		{"not-a-product", "", false},
	}
	for _, tc := range cases {
		got, err := argos.ResolveProductID(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("%q: unexpected err %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseProductHTML(t *testing.T) {
	html := `<html><head>
<script type="application/ld+json">{"@type":"Product","name":"Example Product","offers":{"price":"19.99"}}</script>
<title>Example Product | Argos</title>
</head><body></body></html>`

	info, err := argos.ParseProductHTML(html, "12345678")
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Example Product" {
		t.Fatalf("title=%q", info.Title)
	}
	if info.Price == nil || *info.Price != 19.99 {
		t.Fatalf("price=%v", info.Price)
	}
	if info.ID != "12345678" {
		t.Fatalf("id=%s", info.ID)
	}
}

func TestParseProductHTMLNotFound(t *testing.T) {
	_, err := argos.ParseProductHTML(`<html><body>We can't find this page</body></html>`, "12345678")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestParseProductHTMLBlocked(t *testing.T) {
	_, err := argos.ParseProductHTML(`<html><head><title>Access Denied</title></head><body>Access Denied</body></html>`, "12345678")
	if err == nil {
		t.Fatal("expected blocked error")
	}
}
