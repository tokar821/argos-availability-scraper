package argos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

type Client struct {
	Timeout  time.Duration
	ProxyURL string

	mu           sync.Mutex
	httpClient   tls_client.HttpClient
	warmedProduct string
}

func NewClient() *Client {
	return &Client{
		Timeout: 60 * time.Second,
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient = nil
	c.warmedProduct = ""
}

func (c *Client) ensureHTTP() (tls_client.HttpClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.httpClient != nil {
		return c.httpClient, nil
	}

	timeoutSec := int(c.Timeout.Seconds())
	if timeoutSec <= 0 {
		timeoutSec = 60
	}

	opts := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(chrome152Profile()),
		tls_client.WithTimeoutSeconds(timeoutSec),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithDisableHttp3(),
		tls_client.WithRandomTLSExtensionOrder(),
	}
	proxy := strings.TrimSpace(c.ProxyURL)
	if proxy == "" {
		proxy = strings.TrimSpace(os.Getenv("HTTPS_PROXY"))
	}
	if proxy == "" {
		proxy = strings.TrimSpace(os.Getenv("HTTP_PROXY"))
	}
	if proxy != "" {
		opts = append(opts, tls_client.WithProxyUrl(proxy))
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create tls client: %w", err)
	}
	c.httpClient = client
	return c.httpClient, nil
}

func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, err := c.ensureHTTP()
	if err != nil {
		return nil, err
	}
	type result struct {
		resp *http.Response
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		resp, err := client.Do(req)
		ch <- result{resp, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.resp, r.err
	}
}

func (c *Client) FetchProduct(ctx context.Context, productID string) (ProductInfo, error) {
	html, err := c.loadProductPage(ctx, productID)
	if err != nil {
		return ProductInfo{}, err
	}
	return ParseProductHTML(html, productID)
}

func (c *Client) loadProductPage(ctx context.Context, productID string) (string, error) {
	productURL := ProductURL(productID)
	req, err := http.NewRequest("GET", productURL, nil)
	if err != nil {
		return "", fmt.Errorf("build product request: %w", err)
	}
	req.Header = navigateHeaders()

	resp, err := c.do(ctx, req)
	if err != nil {
		return "", fmt.Errorf("load product page: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read product page: %w", err)
	}
	html := string(body)

	if resp.StatusCode == 403 || (strings.Contains(html, "Access Denied") && len(html) < 8000) {
		return "", fmt.Errorf("blocked: access denied to product page")
	}
	if resp.StatusCode == 404 {
		return "", fmt.Errorf("not found: product page returned 404")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("product page HTTP %d: %s", resp.StatusCode, truncate(html, 180))
	}

	c.mu.Lock()
	c.warmedProduct = productID
	c.mu.Unlock()
	return html, nil
}

func (c *Client) warmSession(ctx context.Context, productID string) error {
	c.mu.Lock()
	warmed := c.warmedProduct == productID
	c.mu.Unlock()
	if warmed {
		return nil
	}
	_, err := c.loadProductPage(ctx, productID)
	return err
}

func (c *Client) FetchCollection(ctx context.Context, productID, location string) (*AvailabilityResponse, error) {
	q := url.Values{}
	q.Set("origin", strings.TrimSpace(location))
	q.Set("skuQty", productID+"_1")
	q.Set("maxResults", "10")
	q.Set("maxDistance", "50")
	q.Set("save", "pdp-ss:/")
	q.Set("ssm", "true")
	return c.fetchAvailability(ctx, productID, AvailabilityPath+"?"+q.Encode())
}

func (c *Client) FetchDelivery(ctx context.Context, productID, location string) (*AvailabilityResponse, error) {
	q := url.Values{}
	q.Set("postcode", strings.TrimSpace(location))
	q.Set("skuQty", productID+"_1")
	q.Set("maxResults", "10")
	q.Set("maxDistance", "50")
	return c.fetchAvailability(ctx, productID, AvailabilityPath+"?"+q.Encode())
}

func (c *Client) fetchAvailability(ctx context.Context, productID, pathAndQuery string) (*AvailabilityResponse, error) {
	if err := c.warmSession(ctx, productID); err != nil {
		return nil, err
	}

	apiURL := BaseURL + pathAndQuery
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build availability request: %w", err)
	}
	req.Header = apiHeaders(ProductURL(productID))

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("availability request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read availability response: %w", err)
	}
	text := string(body)

	if resp.StatusCode == 403 || strings.Contains(text, "Access Denied") {
		return nil, fmt.Errorf("blocked: availability API access denied (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found: availability API returned 404")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("availability API HTTP %d: %s", resp.StatusCode, truncate(text, 180))
	}

	var out AvailabilityResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse availability JSON: %w", err)
	}
	return &out, nil
}
