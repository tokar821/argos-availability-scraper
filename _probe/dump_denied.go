package main

import (
	"fmt"
	"io"
	"os"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func main() {
	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithTimeoutSeconds(20),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithDisableHttp3(),
		tls_client.WithRandomTLSExtensionOrder(),
	)
	req, _ := http.NewRequest("GET", "https://www.argos.co.uk/product/7885338", nil)
	req.Header = http.Header{
		"user-agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"},
		"accept":     {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"accept-language": {"en-GB,en;q=0.9"},
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println("status", resp.StatusCode)
	fmt.Println(string(b))
	_ = os.WriteFile("denied.html", b, 0644)
}
