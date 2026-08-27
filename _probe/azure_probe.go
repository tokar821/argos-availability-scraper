package main

import (
	"fmt"

	"github.com/Noooste/azuretls-client"
)

func main() {
	session := azuretls.NewSession()
	session.Browser = azuretls.Chrome
	session.OrderedHeaders = azuretls.OrderedHeaders{
		{"sec-ch-ua", `"Google Chrome";v="133", "Chromium";v="133", "Not A(Brand";v="24"`},
		{"sec-ch-ua-mobile", "?0"},
		{"sec-ch-ua-platform", `"Windows"`},
		{"upgrade-insecure-requests", "1"},
		{"user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"},
		{"accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		{"sec-fetch-site", "none"},
		{"sec-fetch-mode", "navigate"},
		{"sec-fetch-user", "?1"},
		{"sec-fetch-dest", "document"},
		{"accept-encoding", "gzip, deflate, br, zstd"},
		{"accept-language", "en-GB,en;q=0.9"},
		{"priority", "u=0, i"},
	}
	resp, err := session.Get("https://www.argos.co.uk/product/7885338")
	if err != nil {
		panic(err)
	}
	fmt.Println("status", resp.StatusCode, "len", len(resp.Body))
	body := string(resp.Body)
	if len(body) > 300 {
		body = body[:300]
	}
	fmt.Println(body)
}
