package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func main() {
	for _, p := range []struct{ name string; profile profiles.ClientProfile }{
		{"133", profiles.Chrome_133},
		{"133_PSK", profiles.Chrome_133_PSK},
		{"144", profiles.Chrome_144},
		{"144_PSK", profiles.Chrome_144_PSK},
	} {
		jar := tls_client.NewCookieJar()
		client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
			tls_client.WithTimeoutSeconds(20),
			tls_client.WithClientProfile(p.profile),
			tls_client.WithCookieJar(jar),
		)
		if err != nil {
			fmt.Println(p.name, "create err", err)
			continue
		}
		req, _ := http.NewRequest(http.MethodGet, "https://www.argos.co.uk/product/7885338", nil)
		ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
		req.Header = http.Header{
			"accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
			"accept-language": {"en-GB,en;q=0.9"},
			"user-agent": {ua},
			"sec-ch-ua": {"Chromium";v="144", "Not_A Brand";v="8", "Google Chrome";v="144"},
			"sec-ch-ua-mobile": {"?0"},
			"sec-ch-ua-platform": {"Windows"},
			"upgrade-insecure-requests": {"1"},
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(p.name, "err", err)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		abck := ""
		for _, c := range jar.Cookies(req.URL) {
			if c.Name == "_abck" {
				abck = c.Value
			}
		}
		fmt.Printf("%s status=%d len=%d abck_has_0=%v denied=%v\n", p.name, resp.StatusCode, len(b), strings.Contains(abck, "~0~"), strings.Contains(string(b), "Access Denied"))
	}
	os.Exit(0)
}
