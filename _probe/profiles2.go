package main

import (
	"fmt"
	"io"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func try(name string, profile profiles.ClientProfile, opts ...tls_client.HttpClientOption) {
	base := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profile),
		tls_client.WithTimeoutSeconds(20),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithRandomTLSExtensionOrder(),
	}
	base = append(base, opts...)
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), base...)
	if err != nil {
		fmt.Println(name, "create", err)
		return
	}
	req, _ := http.NewRequest("GET", "https://www.argos.co.uk/", nil)
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	sec := `"Google Chrome";v="133", "Chromium";v="133", "Not A(Brand";v="24"`
	if name == "144" || name == "144_psk" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
		sec = `"Google Chrome";v="144", "Chromium";v="144", "Not A(Brand";v="24"`
	}
	req.Header = http.Header{
		"sec-ch-ua":                 {sec},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {ua},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-GB,en;q=0.9"},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"upgrade-insecure-requests", "user-agent", "accept", "sec-fetch-site",
			"sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest", "accept-encoding",
			"accept-language", "priority",
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(name, "err", err)
		return
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("%s status=%d len=%d title_denied=%v\n", name, resp.StatusCode, len(b), len(b) < 2000)
}

func main() {
	try("133", profiles.Chrome_133, tls_client.WithDisableHttp3())
	try("133_psk", profiles.Chrome_133_PSK, tls_client.WithDisableHttp3())
	try("144", profiles.Chrome_144, tls_client.WithDisableHttp3())
	try("144_psk", profiles.Chrome_144_PSK, tls_client.WithDisableHttp3())
	try("133_h3", profiles.Chrome_133)
}
