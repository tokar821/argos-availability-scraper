package argos_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tokar821/argos-availability-scraper/internal/argos"
)

func TestResolveProductID(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"7885338", "7885338", true},
		{"https://www.argos.co.uk/product/7885338", "7885338", true},
		{"https://www.argos.co.uk/product/7885338?utm_source=x", "7885338", true},
		{"www.argos.co.uk/product/7885338", "7885338", true},
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
	html := readTestdata(t, "product_snippet.html")
	info, err := argos.ParseProductHTML(html, "7885338")
	if err != nil {
		t.Fatal(err)
	}
	if info.Title == "" {
		t.Fatal("expected title")
	}
	if info.Price == nil || *info.Price != 10.00 {
		t.Fatalf("expected price 10.00, got %#v", info.Price)
	}
	if info.ID != "7885338" {
		t.Fatalf("id=%s", info.ID)
	}
}

func TestParseProductHTMLBlocked(t *testing.T) {
	_, err := argos.ParseProductHTML(`<HTML><HEAD><TITLE>Access Denied</TITLE></HEAD><BODY><H1>Access Denied</H1></BODY></HTML>`, "1")
	if err == nil {
		t.Fatal("expected blocked error")
	}
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
