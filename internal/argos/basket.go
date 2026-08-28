package argos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
)

const (
	basketItemsPath    = "/basket-api/v1/basket/items"
	basketLocalisePath = "/basket-api/v2/basket:localise"
	basketCheckoutPath = "/basket-api/v3/basket:checkout"
)

type CheckoutResponse struct {
	SnapshotID string `json:"snapshotId"`
	RedirectTo string `json:"redirectTo"`
}

type localiseResponse struct {
	Items []struct {
		ProductID         string  `json:"productId"`
		Quantity          float64 `json:"quantity"`
		MonthlyCareIntent bool    `json:"monthlyCareIntent"`
	} `json:"items"`
	Products []struct {
		ProductID       string `json:"productId"`
		FulfilmentAgent string `json:"fulfilmentAgent"`
		ProductType     string `json:"productType"`
	} `json:"products"`
	Localisation struct {
		CollectFrom string `json:"collectFrom"`
		DeliverTo   string `json:"deliverTo"`
	} `json:"localisation"`
}

type checkoutItem struct {
	ProductID         string `json:"productId"`
	Quantity          int    `json:"quantity"`
	FulfilmentAgent   string `json:"fulfilmentAgent,omitempty"`
	ProductType       string `json:"productType,omitempty"`
	MonthlyCareIntent bool   `json:"monthlyCareIntent"`
}

type checkoutPayload struct {
	Items          []checkoutItem `json:"items"`
	CollectFrom    string         `json:"collectFrom"`
	DeliverTo      string         `json:"deliverTo"`
	FulfilmentType string         `json:"fulfilmentType"`
	Coupons        []any          `json:"coupons"`
	CheckoutHd     bool           `json:"checkoutHd"`
	SalesChannel   string         `json:"salesChannel"`
	UserAgent      string         `json:"userAgent"`
}

type basketSession struct {
	basketID  string
	sessionID string
	productID string
}

func (c *Client) CheckoutDelivery(ctx context.Context, productID, postcode string) (*CheckoutResponse, error) {
	postcode = strings.TrimSpace(postcode)
	if err := ValidateLocation(postcode); err != nil {
		return nil, err
	}

	if err := c.warmBasketSession(ctx, productID); err != nil {
		return nil, err
	}
	sess, err := c.addBasketItem(ctx, productID, 1)
	if err != nil {
		return nil, err
	}
	sess.productID = productID
	_ = c.getBasket(ctx, sess)
	loc, err := c.localiseBasket(ctx, postcode, "delivery", sess)
	if err != nil {
		return nil, err
	}
	payload, err := buildCheckoutPayload(loc, "delivery")
	if err != nil {
		return nil, err
	}

	body, status, err := c.postCheckout(ctx, payload, "delivery", sess)
	if err != nil {
		return nil, err
	}
	if status == 428 {
		key := strings.TrimSpace(os.Getenv("HYPER_API_KEY"))
		if key == "" {
			return nil, fmt.Errorf("blocked: checkout returned 428 adaptive challenge (set HYPER_API_KEY for Hyper SEC-CPT solve)")
		}
		if err := c.solveAdaptiveChallenge(ctx, key, body); err != nil {
			return nil, fmt.Errorf("adaptive challenge: %w", err)
		}
		body, status, err = c.postCheckout(ctx, payload, "delivery", sess)
		if err != nil {
			return nil, err
		}
	}
	if status == 428 {
		return nil, fmt.Errorf("blocked: checkout still returned 428 after adaptive solve")
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("checkout HTTP %d: %s", status, truncate(body, 240))
	}
	var out CheckoutResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return nil, fmt.Errorf("parse checkout response: %w", err)
	}
	if out.SnapshotID == "" {
		return nil, fmt.Errorf("checkout response missing snapshotId: %s", truncate(body, 240))
	}
	return &out, nil
}

func (c *Client) warmBasketSession(ctx context.Context, productID string) error {
	home, _ := http.NewRequestWithContext(ctx, "GET", BaseURL+"/", nil)
	home.Header = navigateHeaders()
	if resp, err := c.do(ctx, home); err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return c.warmSession(ctx, productID)
}

func (c *Client) addBasketItem(ctx context.Context, productID string, qty int) (basketSession, error) {
	body, _ := json.Marshal(map[string]any{
		"productId": productID,
		"quantity":  qty,
	})
	_, status, hdr, err := c.postBasketJSON(ctx, BaseURL+basketItemsPath, string(body), basketAPIHeaders(ProductURL(productID), nil))
	if err != nil {
		return basketSession{}, fmt.Errorf("add to basket: %w", err)
	}
	if status != 202 && (status < 200 || status >= 300) {
		return basketSession{}, fmt.Errorf("add to basket HTTP %d", status)
	}
	return basketSession{
		basketID:  hdr.Get("Basketid"),
		sessionID: hdr.Get("Sessionid"),
	}, nil
}

func (c *Client) getBasket(ctx context.Context, sess basketSession) error {
	req, err := http.NewRequestWithContext(ctx, "GET", BaseURL+"/basket-api/v1/basket", nil)
	if err != nil {
		return err
	}
	h := basketAPIHeaders(basketPageURL, nil)
	applyBasketSession(h, sess)
	req.Header = h
	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func (c *Client) localiseBasket(ctx context.Context, postcode, fulfilmentType string, sess basketSession) (*localiseResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"deliverTo":      postcode,
		"fulfilmentType": fulfilmentType,
	})
	raw, status, _, err := c.postBasketJSON(ctx, BaseURL+basketLocalisePath, string(body),
		basketAPIHeaders(basketPageURL, map[string]string{"baskettype": "all"}), sess)
	if err != nil || status < 200 || status >= 300 {
		return c.synthesizeLocalise(ctx, sess.productID, postcode)
	}
	var loc localiseResponse
	if err := json.Unmarshal([]byte(raw), &loc); err != nil {
		return nil, fmt.Errorf("parse localise response: %w", err)
	}
	if len(loc.Items) == 0 {
		return c.synthesizeLocalise(ctx, sess.productID, postcode)
	}
	return &loc, nil
}

func (c *Client) synthesizeLocalise(ctx context.Context, productID, postcode string) (*localiseResponse, error) {
	collectFrom := "661"
	if coll, err := c.FetchCollection(ctx, productID, postcode); err == nil && coll != nil && len(coll.Stores) > 0 {
		if id := strings.TrimSpace(coll.Stores[0].StoreID); id != "" {
			collectFrom = id
		}
	}
	agent, ptype := "ADSI", "FIRST_PARTY"
	return &localiseResponse{
		Items: []struct {
			ProductID         string  `json:"productId"`
			Quantity          float64 `json:"quantity"`
			MonthlyCareIntent bool    `json:"monthlyCareIntent"`
		}{{ProductID: productID, Quantity: 1}},
		Products: []struct {
			ProductID       string `json:"productId"`
			FulfilmentAgent string `json:"fulfilmentAgent"`
			ProductType     string `json:"productType"`
		}{{ProductID: productID, FulfilmentAgent: agent, ProductType: ptype}},
		Localisation: struct {
			CollectFrom string `json:"collectFrom"`
			DeliverTo   string `json:"deliverTo"`
		}{CollectFrom: collectFrom, DeliverTo: postcode},
	}, nil
}

func buildCheckoutPayload(loc *localiseResponse, fulfilmentType string) (*checkoutPayload, error) {
	prod := map[string]struct{ agent, ptype string }{}
	for _, p := range loc.Products {
		prod[p.ProductID] = struct{ agent, ptype string }{p.FulfilmentAgent, p.ProductType}
	}
	var items []checkoutItem
	for _, it := range loc.Items {
		ci := checkoutItem{
			ProductID:         it.ProductID,
			Quantity:          int(it.Quantity),
			MonthlyCareIntent: it.MonthlyCareIntent,
		}
		if p, ok := prod[it.ProductID]; ok {
			ci.FulfilmentAgent = p.agent
			ci.ProductType = p.ptype
		}
		if ci.FulfilmentAgent == "" || ci.ProductType == "" {
			return nil, fmt.Errorf("product %s missing fulfilmentAgent/productType after localise", it.ProductID)
		}
		items = append(items, ci)
	}
	return &checkoutPayload{
		Items:          items,
		CollectFrom:    loc.Localisation.CollectFrom,
		DeliverTo:      loc.Localisation.DeliverTo,
		FulfilmentType: fulfilmentType,
		Coupons:        []any{},
		CheckoutHd:     false,
		SalesChannel:   "WEB",
		UserAgent:      chrome152UA,
	}, nil
}

func (c *Client) postCheckout(ctx context.Context, payload *checkoutPayload, fulfilmentHeader string, sess basketSession) (string, int, error) {
	raw, _ := json.Marshal(payload)
	body, status, _, err := c.postBasketJSON(ctx, BaseURL+basketCheckoutPath, string(raw),
		basketAPIHeaders(basketPageURL, map[string]string{"fulfilment": fulfilmentHeader}), sess)
	return body, status, err
}

func applyBasketSession(hdr http.Header, sess basketSession) {
	if sess.basketID != "" {
		hdr.Set("Basketid", sess.basketID)
		hdr.Set("basketid", sess.basketID)
	}
	if sess.sessionID != "" {
		hdr.Set("Sessionid", sess.sessionID)
		hdr.Set("sessionid", sess.sessionID)
	}
}

func (c *Client) postBasketJSON(ctx context.Context, rawURL, body string, hdr http.Header, sess ...basketSession) (string, int, http.Header, error) {
	if len(sess) > 0 {
		applyBasketSession(hdr, sess[0])
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			c.resetHTTP()
		}
		req, err := http.NewRequestWithContext(ctx, "POST", rawURL, strings.NewReader(body))
		if err != nil {
			return "", 0, nil, err
		}
		req.Header = hdr
		resp, err := c.do(ctx, req)
		if err != nil {
			lastErr = err
			if strings.Contains(err.Error(), "INTERNAL_ERROR") || strings.Contains(err.Error(), "EOF") {
				continue
			}
			return "", 0, nil, err
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return string(b), resp.StatusCode, resp.Header, nil
	}
	return "", 0, nil, fmt.Errorf("post %s failed after retries: %w", pathOnly(rawURL), lastErr)
}

func pathOnly(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Path
}

func (c *Client) resetHTTP() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient = nil
}

func (c *Client) getCookie(name string) string {
	client, err := c.ensureHTTP()
	if err != nil {
		return ""
	}
	u, _ := url.Parse(BaseURL)
	for _, ck := range client.GetCookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func (c *Client) fetchGET(ctx context.Context, rawURL string, hdr http.Header) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header = hdr
	resp, err := c.do(ctx, req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), resp.StatusCode, nil
}

func (c *Client) fetchPOST(ctx context.Context, rawURL string, body []byte, hdr http.Header) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", rawURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header = hdr
	resp, err := c.do(ctx, req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), resp.StatusCode, nil
}
