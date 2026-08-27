package main

import (
	"fmt"
	"io"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func main() {
	for _, name := range []string{"h2", "h1", "forceh1"} {
		opts := []tls_client.HttpClientOption{
			tls_client.WithClientProfile(profiles.Chrome_133),
			tls_client.WithTimeoutSeconds(20),
			tls_client.WithCookieJar(tls_client.NewCookieJar()),
			tls_client.WithRandomTLSExtensionOrder(),
			tls_client.WithDisableHttp3(),
		}
		if name == "forceh1" {
			opts = append(opts, tls_client.WithForceHttp1())
		}
		client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
		if err != nil {
			fmt.Println(name, err)
			continue
		}
		req, _ := http.NewRequest("GET", "https://www.argos.co.uk/", nil)
		req.Header = http.Header{
			"sec-ch-ua":                 {`"Google Chrome";v="133", "Chromium";v="133", "Not A(Brand";v="24"`},
			"sec-ch-ua-mobile":          {"?0"},
			"sec-ch-ua-platform":        {`"Windows"`},
			"upgrade-insecure-requests": {"1"},
			"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"},
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
			http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(name, "err", err)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Println(name, resp.StatusCode, len(b))
	}
}
