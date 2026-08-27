package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func main() {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(jar),
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		panic(err)
	}

	productID := "7885338"
	productURL := "https://www.argos.co.uk/product/" + productID
	req, _ := http.NewRequest(http.MethodGet, productURL, nil)
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	secChUA := `"Chromium";v="133", "Not(A:Brand";v="99", "Google Chrome";v="133"`
	req.Header = http.Header{
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"accept-language":           {"en-GB,en;q=0.9"},
		"cache-control":             {"no-cache"},
		"pragma":                    {"no-cache"},
		"sec-ch-ua":                 {secChUA},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"sec-fetch-dest":            {"document"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-user":            {"?1"},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {ua},
		http.HeaderOrderKey: {
			"accept", "accept-language", "cache-control", "pragma",
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site", "sec-fetch-user",
			"upgrade-insecure-requests", "user-agent",
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("product err:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println("product status:", resp.StatusCode, "len:", len(body))
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	fmt.Println("product snippet:", strings.ReplaceAll(snippet, "\n", " "))
	for _, c := range resp.Cookies() {
		fmt.Println("cookie:", c.Name, "=", truncate(c.Value, 40))
	}

	q := url.Values{}
	q.Set("origin", "London")
	q.Set("skuQty", productID+"_1")
	q.Set("maxResults", "10")
	q.Set("maxDistance", "50")
	q.Set("ssm", "true")
	apiURL := "https://www.argos.co.uk/stores/api/orchestrator/v0/locator/availability?" + q.Encode()
	req2, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	req2.Header = http.Header{
		"accept":             {"application/json,*/*"},
		"accept-language":    {"en-GB,en;q=0.9"},
		"content-type":       {"application/json"},
		"referer":            {productURL},
		"sec-ch-ua":          {secChUA},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {`"Windows"`},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-origin"},
		"user-agent":         {ua},
		"x-newrelic-id":      {"VQEPU15SARAGV1hVDgMBUVY="},
		http.HeaderOrderKey: {
			"accept", "accept-language", "content-type", "referer",
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site",
			"user-agent", "x-newrelic-id",
		},
	}
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Println("api err:", err)
		os.Exit(1)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Println("api status:", resp2.StatusCode, "len:", len(body2))
	s2 := string(body2)
	if len(s2) > 300 {
		s2 = s2[:300]
	}
	fmt.Println("api body:", strings.ReplaceAll(s2, "\n", " "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
