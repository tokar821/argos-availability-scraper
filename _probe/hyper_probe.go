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
	"time"

	hyper "github.com/Hyper-Solutions/hyper-sdk-go/v2"
	"github.com/Hyper-Solutions/hyper-sdk-go/v2/akamai"
	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/cookiejar"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const (
	userAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	secChUA         = `"Google Chrome";v="133", "Chromium";v="133", "Not A(Brand";v="24"`
	secChUAPlatform = `"Windows"`
	acceptLanguage  = "en-GB,en;q=0.9"
)

var sbsdRegex = regexp.MustCompile(`(?i)([a-z\d/\-_\.]+)\?v=(.*?)(?:&.*?t=(.*?))?["']`)

func main() {
	apiKey := os.Getenv("HYPER_API_KEY")
	if apiKey == "" {
		panic("HYPER_API_KEY not set")
	}
	jar, _ := cookiejar.New(nil)
	opts := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithTimeoutSeconds(45),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(jar),
		tls_client.WithDisableHttp3(),
	}
	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		opts = append(opts, tls_client.WithProxyUrl(proxy))
		fmt.Println("using proxy")
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	hyperAPI := hyper.NewSession(apiKey)

	ip, err := getIP(ctx, client)
	if err != nil {
		panic(err)
	}
	fmt.Println("ip:", ip)

	productID := "7885338"
	pageURL := "https://www.argos.co.uk/product/" + productID

	html, status, err := get(ctx, client, pageURL, navHeaders())
	if err != nil {
		panic(err)
	}
	fmt.Println("page status:", status, "len:", len(html), "denied:", strings.Contains(html, "Access Denied"))

	if info := parseSbsd(html); info != nil {
		fmt.Println("sbsd:", info)
		if err := solveSbsd(ctx, client, hyperAPI, pageURL, info, ip); err != nil {
			fmt.Println("sbsd err:", err)
		}
		html, status, err = get(ctx, client, pageURL, navHeaders())
		if err != nil {
			panic(err)
		}
		fmt.Println("page after sbsd:", status, "len:", len(html), "denied:", strings.Contains(html, "Access Denied"))
	}

	scriptPath, err := akamai.ParseScriptPath(strings.NewReader(html))
	if err != nil {
		fmt.Println("parse script path:", err)
		os.Exit(1)
	}
	scriptURL := "https://www.argos.co.uk" + scriptPath
	fmt.Println("sensor endpoint:", scriptURL)

	scriptBody, _, err := get(ctx, client, scriptURL, scriptHeaders(pageURL))
	if err != nil {
		panic(err)
	}
	fmt.Println("script len:", len(scriptBody))

	var sensorCtx string
	for i := 0; i < 3; i++ {
		in := &hyper.SensorInput{
			Abck:           cookie(client, pageURL, "_abck"),
			Bmsz:           cookie(client, pageURL, "bm_sz"),
			Version:        "3",
			PageUrl:        pageURL,
			UserAgent:      userAgent,
			ScriptUrl:      scriptURL,
			AcceptLanguage: acceptLanguage,
			IP:             ip,
			Context:        sensorCtx,
		}
		if i == 0 {
			in.Script = scriptBody
		}
		sensorData, nextCtx, err := hyperAPI.GenerateSensorData(ctx, in)
		if err != nil {
			panic(err)
		}
		sensorCtx = nextCtx
		body, _ := json.Marshal(map[string]string{"sensor_data": sensorData})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, scriptURL, bytes.NewReader(body))
		req.Header = sensorPostHeaders(pageURL)
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		abck := cookie(client, pageURL, "_abck")
		fmt.Printf("sensor %d status=%d valid=%v abck=~0~=%v\n", i+1, resp.StatusCode, akamai.IsCookieValid(abck, i), strings.Contains(abck, "~0~"))
		if akamai.IsCookieValid(abck, i) || strings.Contains(abck, "~0~") {
			break
		}
	}

	html, status, err = get(ctx, client, pageURL, navHeaders())
	if err != nil {
		panic(err)
	}
	fmt.Println("page after sensors:", status, "len:", len(html), "denied:", strings.Contains(html, "Access Denied"))

	q := url.Values{}
	q.Set("origin", "London")
	q.Set("skuQty", productID+"_1")
	q.Set("maxResults", "10")
	q.Set("maxDistance", "50")
	q.Set("ssm", "true")
	apiURL := "https://www.argos.co.uk/stores/api/orchestrator/v0/locator/availability?" + q.Encode()
	apiBody, apiStatus, err := get(ctx, client, apiURL, apiHeaders(pageURL))
	if err != nil {
		panic(err)
	}
	fmt.Println("collection status:", apiStatus, "len:", len(apiBody))
	snip := apiBody
	if len(snip) > 250 {
		snip = snip[:250]
	}
	fmt.Println("collection body:", strings.ReplaceAll(snip, "\n", " "))

	q2 := url.Values{}
	q2.Set("postcode", "M1 1AE")
	q2.Set("skuQty", productID+"_1")
	q2.Set("maxResults", "10")
	q2.Set("maxDistance", "50")
	apiURL2 := "https://www.argos.co.uk/stores/api/orchestrator/v0/locator/availability?" + q2.Encode()
	apiBody2, apiStatus2, err := get(ctx, client, apiURL2, apiHeaders(pageURL))
	if err != nil {
		panic(err)
	}
	fmt.Println("delivery status:", apiStatus2, "len:", len(apiBody2))
	snip2 := apiBody2
	if len(snip2) > 250 {
		snip2 = snip2[:250]
	}
	fmt.Println("delivery body:", strings.ReplaceAll(snip2, "\n", " "))
}

type sbsdInfo struct {
	Path, Uuid, T string
}

func parseSbsd(html string) *sbsdInfo {
	m := sbsdRegex.FindStringSubmatch(html)
	if len(m) < 3 {
		return nil
	}
	info := &sbsdInfo{Path: m[1], Uuid: m[2]}
	if len(m) >= 4 {
		info.T = m[3]
	}
	return info
}

func solveSbsd(ctx context.Context, client tls_client.HttpClient, api *hyper.Session, pageURL string, info *sbsdInfo, ip string) error {
	u, _ := url.Parse(pageURL)
	scriptURL := fmt.Sprintf("%s://%s%s?v=%s", u.Scheme, u.Host, info.Path, info.Uuid)
	if info.T != "" {
		scriptURL += "&t=" + info.T
	}
	script, _, err := get(ctx, client, scriptURL, scriptHeaders(pageURL))
	if err != nil {
		return err
	}
	times := 2
	if info.T != "" {
		times = 1
	}
	for i := 0; i < times; i++ {
		o := cookie(client, pageURL, "bm_so")
		if o == "" {
			o = cookie(client, pageURL, "sbsd_o")
		}
		payload, err := api.GenerateSbsdData(ctx, &hyper.SbsdInput{
			Index: i, UserAgent: userAgent, Uuid: info.Uuid, PageUrl: pageURL,
			OCookie: o, Script: script, AcceptLanguage: acceptLanguage, IP: ip,
		})
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{"body": payload})
		postURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, info.Path)
		if info.T != "" {
			postURL += "?t=" + info.T
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(body))
		req.Header = sensorPostHeaders(pageURL)
		req.Header.Set("content-type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		fmt.Println("sbsd post", i, "status", resp.StatusCode)
	}
	return nil
}

func getIP(ctx context.Context, client tls_client.HttpClient) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ip.hypersolutions.co/ip", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("x-api-key", os.Getenv("HYPER_API_KEY"))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("decode ip response %q: %w", string(b), err)
	}
	if out.IP == "" {
		return "", fmt.Errorf("empty ip in response %q", string(b))
	}
	return out.IP, nil
}

func get(ctx context.Context, client tls_client.HttpClient, rawURL string, hdr http.Header) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header = hdr
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return string(b), resp.StatusCode, err
}

func cookie(client tls_client.HttpClient, pageURL, name string) string {
	u, _ := url.Parse(pageURL)
	for _, c := range client.GetCookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func navHeaders() http.Header {
	return http.Header{
		"sec-ch-ua":                 {secChUA},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {secChUAPlatform},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {userAgent},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {acceptLanguage},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"upgrade-insecure-requests", "user-agent", "accept", "sec-fetch-site",
			"sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest", "accept-encoding",
			"accept-language", "priority",
		},
	}
}

func scriptHeaders(referer string) http.Header {
	return http.Header{
		"sec-ch-ua":          {secChUA},
		"sec-ch-ua-mobile":   {"?0"},
		"user-agent":         {userAgent},
		"sec-ch-ua-platform": {secChUAPlatform},
		"accept":             {"*/*"},
		"sec-fetch-site":     {"same-origin"},
		"sec-fetch-mode":     {"no-cors"},
		"sec-fetch-dest":     {"script"},
		"referer":            {referer},
		"accept-encoding":    {"gzip, deflate, br, zstd"},
		"accept-language":    {acceptLanguage},
		"priority":           {"u=1"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "user-agent", "sec-ch-ua-platform",
			"accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest",
			"referer", "accept-encoding", "accept-language", "cookie", "priority",
		},
	}
}

func sensorPostHeaders(referer string) http.Header {
	u, _ := url.Parse(referer)
	return http.Header{
		"sec-ch-ua":          {secChUA},
		"sec-ch-ua-platform": {secChUAPlatform},
		"sec-ch-ua-mobile":   {"?0"},
		"user-agent":         {userAgent},
		"content-type":       {"text/plain;charset=UTF-8"},
		"accept":             {"*/*"},
		"origin":             {fmt.Sprintf("%s://%s", u.Scheme, u.Host)},
		"sec-fetch-site":     {"same-origin"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-dest":     {"empty"},
		"referer":            {referer},
		"accept-encoding":    {"gzip, deflate, br, zstd"},
		"accept-language":    {acceptLanguage},
		"priority":           {"u=1, i"},
		http.HeaderOrderKey: {
			"content-length", "sec-ch-ua", "sec-ch-ua-platform", "sec-ch-ua-mobile",
			"user-agent", "content-type", "accept", "origin",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-dest", "referer",
			"accept-encoding", "accept-language", "cookie", "priority",
		},
	}
}

func apiHeaders(referer string) http.Header {
	return http.Header{
		"accept":             {"application/json,*/*"},
		"accept-language":    {acceptLanguage},
		"content-type":       {"application/json"},
		"referer":            {referer},
		"sec-ch-ua":          {secChUA},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {secChUAPlatform},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-origin"},
		"user-agent":         {userAgent},
		"x-newrelic-id":      {"VQEPU15SARAGV1hVDgMBUVY="},
		"accept-encoding":    {"gzip, deflate, br, zstd"},
		http.HeaderOrderKey: {
			"accept", "accept-language", "content-type", "referer",
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site",
			"user-agent", "x-newrelic-id", "accept-encoding", "cookie",
		},
	}
}

var _ = time.Second
