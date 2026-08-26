package argos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Client fetches Argos product and availability data using a real Chrome
// session. Argos sits behind Akamai bot management; plain HTTP clients are
// routinely blocked, so we load the PDP in Chrome and perform same-origin
// fetch() calls for the availability JSON APIs.
type Client struct {
	Timeout     time.Duration
	Headless    bool // default false — headed Chrome is required to pass Akamai
	ChromePath  string
	UserDataDir string

	mu         sync.Mutex
	allocCtx   context.Context
	allocCancel context.CancelFunc
	browserCtx context.Context
	browserCancel context.CancelFunc
}

func NewClient() *Client {
	return &Client{
		Timeout:  60 * time.Second,
		Headless: false,
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.browserCancel != nil {
		c.browserCancel()
		c.browserCancel = nil
	}
	if c.allocCancel != nil {
		c.allocCancel()
		c.allocCancel = nil
	}
}

func (c *Client) ensureBrowser(parent context.Context) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.browserCtx != nil {
		return c.browserCtx, nil
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", c.Headless),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.WindowSize(1280, 900),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"),
	)
	if c.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(c.ChromePath))
	}
	if c.UserDataDir != "" {
		opts = append(opts, chromedp.UserDataDir(c.UserDataDir))
	} else {
		dir, err := os.MkdirTemp("", "argos-chrome-*")
		if err != nil {
			return nil, fmt.Errorf("create chrome profile dir: %w", err)
		}
		c.UserDataDir = dir
		opts = append(opts, chromedp.UserDataDir(dir))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(parent, opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	c.allocCtx = allocCtx
	c.allocCancel = allocCancel
	c.browserCtx = browserCtx
	c.browserCancel = browserCancel

	// Warm the browser process.
	if err := chromedp.Run(browserCtx); err != nil {
		browserCancel()
		allocCancel()
		c.browserCtx = nil
		return nil, fmt.Errorf("start chrome: %w", err)
	}
	return c.browserCtx, nil
}

// FetchProduct loads the product page and parses title/price.
func (c *Client) FetchProduct(ctx context.Context, productID string) (ProductInfo, error) {
	browserCtx, err := c.ensureBrowser(ctx)
	if err != nil {
		return ProductInfo{}, err
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(browserCtx, timeout)
	defer cancel()

	productURL := ProductURL(productID)
	var html string
	var title string
	err = chromedp.Run(runCtx,
		chromedp.Navigate(productURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(1500*time.Millisecond),
		chromedp.Title(&title),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return ProductInfo{}, fmt.Errorf("load product page: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(title), "Access Denied") || strings.Contains(html, "Access Denied") && len(html) < 5000 {
		return ProductInfo{}, fmt.Errorf("blocked: Argos/Akamai denied access to product page (try again, or use a normal desktop network)")
	}
	info, err := ParseProductHTML(html, productID)
	if err != nil {
		// Fallback to document title if JSON-LD missing but page loaded.
		if info.Title == "" && title != "" && !strings.EqualFold(title, "Access Denied") {
			info.Title = cleanTitle(title)
			info.ID = productID
			info.URL = productURL
			return info, nil
		}
		return info, err
	}
	return info, nil
}

// FetchCollection queries nearby collection availability using origin=.
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

// FetchDelivery queries home delivery availability using postcode=.
// Argos returns delivery options when the locator API is called with postcode=
// (not origin=). Town names may return empty delivery; UK postcodes work best.
func (c *Client) FetchDelivery(ctx context.Context, productID, location string) (*AvailabilityResponse, error) {
	q := url.Values{}
	q.Set("postcode", strings.TrimSpace(location))
	q.Set("skuQty", productID+"_1")
	q.Set("maxResults", "10")
	q.Set("maxDistance", "50")
	return c.fetchAvailability(ctx, productID, AvailabilityPath+"?"+q.Encode())
}

func (c *Client) fetchAvailability(ctx context.Context, productID, pathAndQuery string) (*AvailabilityResponse, error) {
	browserCtx, err := c.ensureBrowser(ctx)
	if err != nil {
		return nil, err
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	runCtx, cancel := context.WithTimeout(browserCtx, timeout)
	defer cancel()

	// Ensure we are on the product origin so fetch is same-origin.
	productURL := ProductURL(productID)
	var href string
	_ = chromedp.Run(runCtx, chromedp.Location(&href))
	if !strings.Contains(href, "/product/"+productID) {
		err = chromedp.Run(runCtx,
			chromedp.Navigate(productURL),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Sleep(800*time.Millisecond),
		)
		if err != nil {
			return nil, fmt.Errorf("navigate before availability fetch: %w", err)
		}
	}

	js := fmt.Sprintf(`(async () => {
  const url = %q;
  const res = await fetch(url, {
    method: 'GET',
    credentials: 'include',
    headers: {
      'accept': 'application/json,*/*',
      'content-type': 'application/json',
      'x-newrelic-id': 'VQEPU15SARAGV1hVDgMBUVY='
    }
  });
  const text = await res.text();
  return JSON.stringify({ status: res.status, body: text });
})()`, pathAndQuery)

	var raw string
	err = chromedp.Run(runCtx, chromedp.Evaluate(js, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithAwaitPromise(true)
	}))
	if err != nil {
		return nil, fmt.Errorf("availability fetch evaluate: %w", err)
	}

	var wrap struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw), &wrap); err != nil {
		return nil, fmt.Errorf("decode fetch wrapper: %w (raw=%s)", err, truncate(raw, 180))
	}
	if wrap.Status == 403 || strings.Contains(wrap.Body, "Access Denied") {
		return nil, fmt.Errorf("blocked: Argos/Akamai denied availability API (HTTP %d)", wrap.Status)
	}
	if wrap.Status == 404 {
		return nil, fmt.Errorf("not found: availability API returned 404")
	}
	if wrap.Status < 200 || wrap.Status >= 300 {
		return nil, fmt.Errorf("availability API HTTP %d: %s", wrap.Status, truncate(wrap.Body, 180))
	}

	var resp AvailabilityResponse
	if err := json.Unmarshal([]byte(wrap.Body), &resp); err != nil {
		return nil, fmt.Errorf("parse availability JSON: %w", err)
	}
	return &resp, nil
}
