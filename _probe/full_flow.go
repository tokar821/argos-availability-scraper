package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	hyper "github.com/Hyper-Solutions/hyper-sdk-go/v2"
	"github.com/Hyper-Solutions/hyper-sdk-go/v2/akamai"
	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/cookiejar"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const (
	ua  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	sec = `"Google Chrome";v="133", "Chromium";v="133", "Not A(Brand";v="24"`
	al  = "en-GB,en;q=0.9"
)

var re = regexp.MustCompile(`(?i)([a-z\d/\-_\.]+)\?v=([0-9a-f\-]+)(?:&t=([^"'&]+))?`)

func main() {
	proxy := os.Getenv("HTTP_PROXY")
	key := os.Getenv("HYPER_API_KEY")
	jar, _ := cookiejar.New(nil)
	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(jar),
		tls_client.WithDisableHttp3(),
		tls_client.WithProxyUrl(proxy),
	)
	api := hyper.NewSession(key)
	ctx := context.Background()
	ip := mustIP(ctx, client, key)
	fmt.Println("ip", ip)

	// warm homepage first
	home := "https://www.argos.co.uk/"
	h1, hs := mustGet(ctx, client, home, nav())
	fmt.Println("home", hs, len(h1), "denied", strings.Contains(h1, "Access Denied"))

	pageURL := "https://www.argos.co.uk/product/7885338"
	html, st := mustGet(ctx, client, pageURL, nav())
	fmt.Println("product1", st, len(html), "denied", strings.Contains(html, "Access Denied"))

	for round := 1; round <= 3; round++ {
		if !strings.Contains(html, "Access Denied") && st == 200 {
			break
		}
		m := re.FindStringSubmatch(html)
		if m == nil {
			fmt.Println("no sbsd on round", round)
			break
		}
		path, uuid, t := m[1], m[2], ""
		if len(m) > 3 {
			t = m[3]
		}
		fmt.Printf("round %d uuid=%s t=%q\n", round, uuid, t)
		scriptURL := "https://www.argos.co.uk" + path + "?v=" + uuid
		if t != "" {
			scriptURL += "&t=" + t
		}
		script, _ := mustGet(ctx, client, scriptURL, scriptHdr(pageURL))
		times := 2
		if t != "" {
			times = 1
		}
		for i := 0; i < times; i++ {
			o := cookie(client, pageURL, "sbsd_o")
			if o == "" {
				o = cookie(client, pageURL, "bm_so")
			}
			payload, err := api.GenerateSbsdData(ctx, &hyper.SbsdInput{
				Index: i, UserAgent: ua, Uuid: uuid, PageUrl: pageURL,
				OCookie: o, Script: script, AcceptLanguage: al, IP: ip,
			})
			if err != nil {
				panic(err)
			}
			postURL := "https://www.argos.co.uk" + path
			if t != "" {
				postURL += "?t=" + t
			}
			body, _ := json.Marshal(map[string]string{"body": payload})
			req, _ := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader(body))
			req.Header = postHdr(pageURL)
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			fmt.Println("  post", i, resp.StatusCode)
		}
		html, st = mustGet(ctx, client, pageURL, nav())
		fmt.Println("product after round", round, st, len(html), "denied", strings.Contains(html, "Access Denied"))
	}

	if strings.Contains(html, "Access Denied") {
		fmt.Println("FAILED still denied")
		os.Exit(1)
	}
	fmt.Println("page unlocked, trying sensors + api")

	scriptPath, err := akamai.ParseScriptPath(strings.NewReader(html))
	if err != nil {
		fmt.Println("no sensor path:", err)
		// still try API
	} else {
		scriptURL := "https://www.argos.co.uk" + scriptPath
		scriptBody, _ := mustGet(ctx, client, scriptURL, scriptHdr(pageURL))
		var sensorCtx string
		for i := 0; i < 3; i++ {
			in := &hyper.SensorInput{
				Abck: cookie(client, pageURL, "_abck"), Bmsz: cookie(client, pageURL, "bm_sz"),
				Version: "3", PageUrl: pageURL, UserAgent: ua, ScriptUrl: scriptURL,
				AcceptLanguage: al, IP: ip, Context: sensorCtx,
			}
			if i == 0 {
				in.Script = scriptBody
			}
			sensorData, next, err := api.GenerateSensorData(ctx, in)
			if err != nil {
				panic(err)
			}
			sensorCtx = next
			body, _ := json.Marshal(map[string]string{"sensor_data": sensorData})
			req, _ := http.NewRequestWithContext(ctx, "POST", scriptURL, bytes.NewReader(body))
			req.Header = sensorPost(pageURL)
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			abck := cookie(client, pageURL, "_abck")
			fmt.Println("sensor", i+1, "valid", akamai.IsCookieValid(abck, i), "has0", strings.Contains(abck, "~0~"))
			if akamai.IsCookieValid(abck, i) || strings.Contains(abck, "~0~") {
				break
			}
		}
	}

	q := url.Values{"origin": {"London"}, "skuQty": {"7885338_1"}, "maxResults": {"10"}, "maxDistance": {"50"}, "ssm": {"true"}}
	apiURL := "https://www.argos.co.uk/stores/api/orchestrator/v0/locator/availability?" + q.Encode()
	body, st := mustGet(ctx, client, apiURL, apiHdr(pageURL))
	fmt.Println("collection", st, len(body), truncate(body, 200))
}

func mustIP(ctx context.Context, client tls_client.HttpClient, key string) string {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://ip.hypersolutions.co/ip", nil)
	req.Header.Set("x-api-key", key)
	req.Header.Set("user-agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out struct {
		IP string `json:"ip"`
	}
	json.Unmarshal(b, &out)
	return out.IP
}
func mustGet(ctx context.Context, client tls_client.HttpClient, u string, h http.Header) (string, int) {
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header = h
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), resp.StatusCode
}
func cookie(client tls_client.HttpClient, page, name string) string {
	u, _ := url.Parse(page)
	for _, c := range client.GetCookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
func nav() http.Header {
	return http.Header{
		"sec-ch-ua": {sec}, "sec-ch-ua-mobile": {"?0"}, "sec-ch-ua-platform": {`"Windows"`},
		"upgrade-insecure-requests": {"1"}, "user-agent": {ua},
		"accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site": {"none"}, "sec-fetch-mode": {"navigate"}, "sec-fetch-user": {"?1"}, "sec-fetch-dest": {"document"},
		"accept-encoding": {"gzip, deflate, br, zstd"}, "accept-language": {al}, "priority": {"u=0, i"},
		http.HeaderOrderKey: {"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "upgrade-insecure-requests", "user-agent", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest", "accept-encoding", "accept-language", "priority"},
	}
}
func scriptHdr(ref string) http.Header {
	return http.Header{
		"sec-ch-ua-platform": {`"Windows"`}, "user-agent": {ua}, "sec-ch-ua": {sec}, "sec-ch-ua-mobile": {"?0"},
		"accept": {"*/*"}, "sec-fetch-site": {"same-origin"}, "sec-fetch-mode": {"no-cors"}, "sec-fetch-dest": {"script"},
		"referer": {ref}, "accept-encoding": {"gzip, deflate, br, zstd"}, "accept-language": {al}, "priority": {"u=1"},
		http.HeaderOrderKey: {"sec-ch-ua-platform", "user-agent", "sec-ch-ua", "sec-ch-ua-mobile", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest", "referer", "accept-encoding", "accept-language", "cookie", "priority"},
	}
}
func postHdr(ref string) http.Header {
	u, _ := url.Parse(ref)
	return http.Header{
		"sec-ch-ua": {sec}, "content-type": {"application/json"}, "sec-ch-ua-mobile": {"?0"}, "user-agent": {ua},
		"sec-ch-ua-platform": {`"Windows"`}, "accept": {"*/*"}, "origin": {u.Scheme + "://" + u.Host},
		"sec-fetch-site": {"same-origin"}, "sec-fetch-mode": {"cors"}, "sec-fetch-dest": {"empty"},
		"referer": {ref}, "accept-encoding": {"gzip, deflate, br, zstd"}, "accept-language": {al}, "priority": {"u=1, i"},
		http.HeaderOrderKey: {"content-length", "sec-ch-ua", "content-type", "sec-ch-ua-mobile", "user-agent", "sec-ch-ua-platform", "accept", "origin", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest", "referer", "accept-encoding", "accept-language", "cookie", "priority"},
	}
}
func sensorPost(ref string) http.Header {
	u, _ := url.Parse(ref)
	return http.Header{
		"sec-ch-ua": {sec}, "sec-ch-ua-platform": {`"Windows"`}, "sec-ch-ua-mobile": {"?0"}, "user-agent": {ua},
		"content-type": {"text/plain;charset=UTF-8"}, "accept": {"*/*"}, "origin": {u.Scheme + "://" + u.Host},
		"sec-fetch-site": {"same-origin"}, "sec-fetch-mode": {"cors"}, "sec-fetch-dest": {"empty"},
		"referer": {ref}, "accept-encoding": {"gzip, deflate, br, zstd"}, "accept-language": {al}, "priority": {"u=1, i"},
		http.HeaderOrderKey: {"content-length", "sec-ch-ua", "sec-ch-ua-platform", "sec-ch-ua-mobile", "user-agent", "content-type", "accept", "origin", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest", "referer", "accept-encoding", "accept-language", "cookie", "priority"},
	}
}
func apiHdr(ref string) http.Header {
	return http.Header{
		"accept": {"application/json,*/*"}, "accept-language": {al}, "content-type": {"application/json"}, "referer": {ref},
		"sec-ch-ua": {sec}, "sec-ch-ua-mobile": {"?0"}, "sec-ch-ua-platform": {`"Windows"`},
		"sec-fetch-dest": {"empty"}, "sec-fetch-mode": {"cors"}, "sec-fetch-site": {"same-origin"},
		"user-agent": {ua}, "x-newrelic-id": {"VQEPU15SARAGV1hVDgMBUVY="}, "accept-encoding": {"gzip, deflate, br, zstd"},
		http.HeaderOrderKey: {"accept", "accept-language", "content-type", "referer", "sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site", "user-agent", "x-newrelic-id", "accept-encoding", "cookie"},
	}
}
