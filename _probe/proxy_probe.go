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
	proxy := os.Getenv("HTTP_PROXY")
	if proxy == "" {
		panic("HTTP_PROXY not set")
	}
	fmt.Println("proxy host:", proxy[strings.LastIndex(proxy, "@")+1:])

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithTimeoutSeconds(45),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithDisableHttp3(),
		tls_client.WithProxyUrl(proxy),
	)
	if err != nil {
		panic(err)
	}

	for i := 1; i <= 2; i++ {
		req, _ := http.NewRequest("GET", "https://api.ipify.org", nil)
		req.Header.Set("user-agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("ipify", i, err)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Println("exit ip", i, strings.TrimSpace(string(b)))
	}

	req, _ := http.NewRequest("GET", "https://www.argos.co.uk/product/7885338", nil)
	req.Header = http.Header{
		"sec-ch-ua":                 {`"Google Chrome";v="133", "Chromium";v="133", "Not A(Brand";v="24"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-GB,en;q=0.9"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"upgrade-insecure-requests", "user-agent", "accept",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
			"accept-encoding", "accept-language",
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println("argos status", resp.StatusCode, "len", len(b), "denied", strings.Contains(string(b), "Access Denied"))
	if len(b) > 200 && !strings.Contains(string(b), "Access Denied") {
		fmt.Println("argos ok snippet:", strings.ReplaceAll(string(b)[:120], "\n", " "))
	} else {
		fmt.Println(string(b))
	}
}
