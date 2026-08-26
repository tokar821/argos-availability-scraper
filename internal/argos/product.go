package argos

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	productIDFromURL = regexp.MustCompile(`(?i)/product/(\d{5,})`)
	digitsOnly       = regexp.MustCompile(`^\d{5,}$`)
	titleTag         = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	ogTitle          = regexp.MustCompile(`(?is)property=["']og:title["']\s+content=["']([^"']+)["']`)
	itemPropPrice    = regexp.MustCompile(`(?is)itemprop=["']price["']\s+content=["']([0-9]+(?:\.[0-9]+)?)["']`)
	jsonLDBlocks     = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
)

func ResolveProductID(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("product is required")
	}
	if digitsOnly.MatchString(input) {
		return input, nil
	}
	if m := productIDFromURL.FindStringSubmatch(input); len(m) == 2 {
		return m[1], nil
	}
	if strings.Contains(input, "argos.co.uk") && !strings.Contains(input, "://") {
		return ResolveProductID("https://" + input)
	}
	u, err := url.Parse(input)
	if err == nil && u.Host != "" {
		if m := productIDFromURL.FindStringSubmatch(u.Path); len(m) == 2 {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("could not parse product ID from %q (expected Argos product ID or URL)", input)
}

func ProductURL(productID string) string {
	return BaseURL + "/product/" + productID
}

func ParseProductHTML(html, productID string) (ProductInfo, error) {
	info := ProductInfo{
		ID:  productID,
		URL: ProductURL(productID),
	}
	if strings.Contains(html, "Access Denied") && len(html) < 2000 {
		return info, fmt.Errorf("blocked: Argos returned Access Denied for product page")
	}
	if strings.Contains(strings.ToLower(html), "page not found") || strings.Contains(html, "We can't find this page") {
		return info, fmt.Errorf("product not found: %s", productID)
	}

	if m := jsonLDBlocks.FindAllStringSubmatch(html, -1); len(m) > 0 {
		for _, block := range m {
			title, price, ok := parseJSONLDProduct(block[1], productID)
			if !ok {
				continue
			}
			if title != "" {
				info.Title = title
			}
			if price != nil {
				info.Price = price
			}
		}
	}

	if info.Title == "" {
		if m := ogTitle.FindStringSubmatch(html); len(m) == 2 {
			info.Title = cleanTitle(m[1])
		}
	}
	if info.Title == "" {
		if m := titleTag.FindStringSubmatch(html); len(m) == 2 {
			info.Title = cleanTitle(m[1])
		}
	}
	if info.Price == nil {
		if m := itemPropPrice.FindStringSubmatch(html); len(m) == 2 {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				info.Price = &v
			}
		}
	}

	if info.Title == "" {
		return info, fmt.Errorf("could not parse product title for %s", productID)
	}
	return info, nil
}

func cleanTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&apos;", "'")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.TrimPrefix(s, "Buy ")
	if i := strings.Index(s, " | "); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func parseJSONLDProduct(raw, productID string) (title string, price *float64, ok bool) {
	raw = strings.TrimSpace(raw)
	var root any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return "", nil, false
	}
	nodes := flattenJSONLD(root)
	for _, node := range nodes {
		m, isMap := node.(map[string]any)
		if !isMap {
			continue
		}
		t, _ := m["@type"].(string)
		if !strings.EqualFold(t, "Product") {
			if arr, ok := m["@type"].([]any); ok {
				found := false
				for _, x := range arr {
					if s, _ := x.(string); strings.EqualFold(s, "Product") {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			} else if t == "" {
				continue
			} else {
				continue
			}
		}
		if name, _ := m["name"].(string); name != "" {
			title = strings.TrimSpace(name)
		}
		price = extractOffersPrice(m["offers"])
		ok = title != "" || price != nil
		if ok {
			return title, price, true
		}
	}
	return "", nil, false
}

func flattenJSONLD(v any) []any {
	switch t := v.(type) {
	case []any:
		out := make([]any, 0, len(t))
		for _, x := range t {
			out = append(out, flattenJSONLD(x)...)
		}
		return out
	case map[string]any:
		if graph, ok := t["@graph"]; ok {
			return flattenJSONLD(graph)
		}
		return []any{t}
	default:
		return nil
	}
}

func extractOffersPrice(offers any) *float64 {
	switch o := offers.(type) {
	case map[string]any:
		return priceFromOfferMap(o)
	case []any:
		for _, item := range o {
			if m, ok := item.(map[string]any); ok {
				if p := priceFromOfferMap(m); p != nil {
					return p
				}
			}
		}
	}
	return nil
}

func priceFromOfferMap(m map[string]any) *float64 {
	switch v := m["price"].(type) {
	case float64:
		return &v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}
