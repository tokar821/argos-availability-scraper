package argos

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURL          = "https://www.argos.co.uk"
	AvailabilityPath = "/stores/api/orchestrator/v0/locator/availability"
)

type ProductInfo struct {
	ID    string
	Title string
	Price *float64
	URL   string
}

type AvailabilityResponse struct {
	Stores      []StoreEntry       `json:"stores"`
	Suggestions json.RawMessage    `json:"suggestions"`
	Delivery    json.RawMessage    `json:"delivery"`
	Products    map[string]Product `json:"products"`
	SafeMode    *SafeMode          `json:"safeMode"`
	Empty       bool               `json:"empty"`
}

type SafeMode struct {
	StockCheckDisabled bool `json:"stockCheckDisabled"`
}

type Product struct {
	Data ProductData `json:"data"`
}

type ProductData struct {
	ID         string            `json:"id"`
	Attributes ProductAttributes `json:"attributes"`
}

type ProductAttributes struct {
	Deliverable         bool `json:"deliverable"`
	Collectable         bool `json:"collectable"`
	GloballyOutOfStock  bool `json:"globallyOutOfStock"`
	EndOfLineOutOfStock bool `json:"endOfLineOutOfStock"`
}

type StoreEntry struct {
	Distance     string            `json:"distance"`
	StoreInfo    StoreInfo         `json:"storeinfo"`
	Availability []SKUAvailability `json:"availability"`
	Messages     map[string]Message `json:"messages"`
	StoreID      string            `json:"store_id"`
}

type StoreInfo struct {
	StoreID     string   `json:"store_id"`
	LegacyName  string   `json:"legacy_name"`
	Town        string   `json:"town"`
	County      string   `json:"county"`
	Postcode    string   `json:"postcode"`
	AddressLine []any    `json:"address_line"`
}

type SKUAvailability struct {
	Requested           string             `json:"requested"`
	NodeID              string             `json:"nodeId"`
	Postcode            string             `json:"postcode"`
	SKU                 string             `json:"sku"`
	QuantityRequested   int                `json:"quantityRequested"`
	QuantityAvailable   int                `json:"quantityAvailable"`
	CustomerAvailableBy string             `json:"customerAvailableBy"`
	Cutoff              string             `json:"cutoff"`
	Excluded            bool               `json:"excluded"`
	LeadTime            string             `json:"leadTime"`
	FulfillmentNodes    []FulfillmentNode  `json:"fulfillmentNodes"`
}

type FulfillmentNode struct {
	NodeID            string `json:"nodeId"`
	QuantityFulfilled int    `json:"quantityFulfilled"`
}

type Message struct {
	MessageKey    string `json:"messageKey"`
	Text          string `json:"text"`
	Icon          string `json:"icon"`
	AnalyticsCode string `json:"analyticsCode"`
	Fasttrack     string `json:"fasttrack"`
}

type DeliveryEntry struct {
	Postcode     string             `json:"postcode"`
	Messages     map[string]Message `json:"messages"`
	Availability []SKUAvailability  `json:"availability"`
	Fee          *DeliveryFee       `json:"fee"`
	Cost         *DeliveryFee       `json:"cost"`
	DeliveryFee  *DeliveryFee       `json:"deliveryFee"`
}

type DeliveryFee struct {
	Amount   any    `json:"amount"`
	Currency string `json:"currency"`
	Value    any    `json:"value"`
	Display  string `json:"display"`
}

func ParseDeliveryPayload(raw json.RawMessage) ([]DeliveryEntry, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var arr []DeliveryEntry
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var one DeliveryEntry
	if err := json.Unmarshal(raw, &one); err == nil {
		if one.Postcode == "" && len(one.Availability) == 0 && len(one.Messages) == 0 && one.Fee == nil && one.Cost == nil && one.DeliveryFee == nil {
			return nil, fmt.Errorf("unexpected delivery payload: %s", truncate(string(raw), 120))
		}
		return []DeliveryEntry{one}, nil
	}
	return nil, fmt.Errorf("unexpected delivery payload: %s", truncate(string(raw), 120))
}

func FormatAddress(lines []any, postcode string) string {
	parts := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		s, ok := line.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			parts = append(parts, s)
		}
	}
	pc := strings.TrimSpace(postcode)
	if pc != "" {
		parts = append(parts, pc)
	}
	return strings.Join(parts, ", ")
}

func ParseDistanceMiles(raw string) *float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &v
}

func ParseTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t.UTC(), true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
