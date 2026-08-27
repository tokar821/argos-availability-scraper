package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type cfg struct {
	name    string
	profile profiles.ClientProfile
	uaVer   string
	opts    []tls_client.HttpClientOption
}

func main() {
	proxy := os.Getenv("HTTP_PROXY")
	cfgs := []cfg{
		{"146", profiles.Chrome_146, "146", []tls_client.HttpClientOption{tls_client.WithDisableHttp3()}},
		{"146_psk", profiles.Chrome_146_PSK, "146", []tls_client.HttpClientOption{tls_client.WithDisableHttp3()}},
		{"146_norand", profiles.Chrome_146, "146", []tls_client.HttpClientOption{tls_client.WithDisableHttp3()}},
		{"144", profiles.Chrome_144, "144", []tls_client.HttpClientOption{tls_client.WithDisableHttp3()}},
		{"144_psk", profiles.Chrome_144_PSK, "144", []tls_client.HttpClientOption{tls_client.WithDisableHttp3()}},
		{"133", profiles.Chrome_133, "133", []tls_client.HttpClientOption{tls_client.WithDisableHttp3()}},
		{"146_h3", profiles.Chrome_146, "146", nil},
		{"146_h1", profiles.Chrome_146, "146", []tls_client.HttpClientOption{tls_client.WithForceHttp1()}},
	}

	for _, c := range cfgs {
		opts := []tls_client.HttpClientOption{
			tls_client.WithClientProfile(c.profile),
			tls_client.WithTimeoutSeconds(30),
			tls_client.WithCookieJar(tls_client.NewCookieJar()),
		}
		if c.name != "146_norand" {
			opts = append(opts, tls_client.WithRandomTLSExtensionOrder())
		}
		opts = append(opts, c.opts...)
		if proxy != "" {
			opts = append(opts, tls_client.WithProxyUrl(proxy))
		}
		client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
		if err != nil {
			fmt.Println(c.name, "create", err)
			continue
		}

		// fingerprint check
		req, _ := http.NewRequest("GET", "https://tls.peet.ws/api/all", nil)
		applyNav(req, c.uaVer)
		resp, err := client.Do(req)
		ja4 := "?"
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			if tls, ok := m["tls"].(map[string]any); ok {
				if j, ok := tls["ja4"].(string); ok {
					ja4 = j
				}
			}
			if j, ok := m["ja4"].(string); ok {
				ja4 = j
			}
		}

		req2, _ := http.NewRequest("GET", "https://www.argos.co.uk/", nil)
		applyNav(req2, c.uaVer)
		resp2, err := client.Do(req2)
		if err != nil {
			fmt.Printf("%-12s ja4=%s argos=ERR %v\n", c.name, ja4, err)
			continue
		}
		b2, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		denied := strings.Contains(string(b2), "Access Denied")
		fmt.Printf("%-12s ja4=%s status=%d len=%d denied=%v\n", c.name, ja4, resp2.StatusCode, len(b2), denied)
	}
}

func applyNav(req *http.Request, ver string) {
	sec := fmt.Sprintf(`"Google Chrome";v="%s", "Chromium";v="%s", "Not A(Brand";v="24"`, ver, ver)
	ua := fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36", ver)
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
			"upgrade-insecure-requests", "user-agent", "accept",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
			"accept-encoding", "accept-language", "priority",
		},
		http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	}
}
