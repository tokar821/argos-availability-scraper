package normalize_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/tokar821/argos-availability-scraper/internal/argos"
	"github.com/tokar821/argos-availability-scraper/internal/normalize"
)

const testProductID = "12345678"

const collectionFixture = `{
  "stores": [
    {
      "distance": "2.5",
      "storeinfo": {
        "store_id": "100",
        "legacy_name": "Example Store A",
        "town": "Example Town",
        "postcode": "AB1 2CD",
        "address_line": ["1 High Street"]
      },
      "availability": [{
        "sku": "12345678",
        "quantityAvailable": 0,
        "customerAvailableBy": "2026-08-28T17:00:00Z"
      }],
      "messages": {"12345678": {"text": "Out of stock"}},
      "store_id": "100"
    },
    {
      "distance": "1.2",
      "storeinfo": {
        "store_id": "200",
        "legacy_name": "Example Store B",
        "town": "Example Town",
        "postcode": "AB1 2CD",
        "address_line": ["2 High Street"]
      },
      "availability": [{
        "sku": "12345678",
        "quantityAvailable": 3,
        "customerAvailableBy": "2026-08-27T17:00:00Z"
      }],
      "messages": {"12345678": {"text": "Order now, collect from 5pm tomorrow"}},
      "store_id": "200"
    }
  ]
}`

const deliveryFixture = `{
  "delivery": [{
    "postcode": "AB1 2CD",
    "messages": {"12345678": {"text": "Next day delivery available"}},
    "availability": [{
      "sku": "12345678",
      "quantityAvailable": 5,
      "customerAvailableBy": "2026-08-28T23:00:00Z"
    }],
    "fee": {"amount": 3.95, "currency": "GBP", "display": "£3.95"}
  }]
}`

func TestNormalizeCollection(t *testing.T) {
	var resp argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(collectionFixture), &resp); err != nil {
		t.Fatal(err)
	}
	out := normalize.NormalizeCollection(testProductID, &resp, nil)
	if out.Status != "available" {
		t.Fatalf("status=%s", out.Status)
	}
	if len(out.Stores) != 2 {
		t.Fatalf("stores=%d", len(out.Stores))
	}
	if out.Stores[0].Status != "unavailable" || out.Stores[1].Status != "available" {
		t.Fatalf("store statuses: %s, %s", out.Stores[0].Status, out.Stores[1].Status)
	}
	if out.Message != "Order now, collect from 5pm tomorrow" {
		t.Fatalf("message=%q", out.Message)
	}
}

func TestNormalizeCollectionUnavailable(t *testing.T) {
	var resp argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(collectionFixture), &resp); err != nil {
		t.Fatal(err)
	}
	for i := range resp.Stores {
		if len(resp.Stores[i].Availability) > 0 {
			resp.Stores[i].Availability[0].QuantityAvailable = 0
		}
	}
	out := normalize.NormalizeCollection(testProductID, &resp, nil)
	if out.Status != "unavailable" {
		t.Fatalf("status=%s", out.Status)
	}
}

func TestNormalizeCollectionError(t *testing.T) {
	out := normalize.NormalizeCollection(testProductID, nil, fmt.Errorf("blocked: Access Denied"))
	if out.Status != "error" || out.Error == nil || out.Error.Code != "blocked" {
		t.Fatalf("got %#v", out)
	}
}

func TestNormalizeDelivery(t *testing.T) {
	var resp argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(deliveryFixture), &resp); err != nil {
		t.Fatal(err)
	}
	out := normalize.NormalizeDelivery(testProductID, &resp, nil)
	if out.Status != "available" {
		t.Fatalf("status=%s", out.Status)
	}
	if out.Fee == nil || out.Fee.Display != "£3.95" {
		t.Fatalf("fee=%#v", out.Fee)
	}
	if out.EarliestDate == "" {
		t.Fatal("expected earliest date")
	}
}

func TestNormalizeDeliveryEmpty(t *testing.T) {
	resp := &argos.AvailabilityResponse{Delivery: json.RawMessage("null")}
	out := normalize.NormalizeDelivery(testProductID, resp, nil)
	if out.Status != "unavailable" {
		t.Fatalf("status=%s", out.Status)
	}
}

func TestBuildResult(t *testing.T) {
	price := 19.99
	product := argos.ProductInfo{
		ID:    testProductID,
		Title: "Example Product",
		Price: &price,
		URL:   "https://www.argos.co.uk/product/" + testProductID,
	}
	var collection, delivery argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(collectionFixture), &collection); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(deliveryFixture), &delivery); err != nil {
		t.Fatal(err)
	}
	res := normalize.BuildResult(
		product, "AB1 2CD", "both",
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		&collection, &delivery, nil, nil,
	)
	if res.Collection == nil || res.Delivery == nil {
		t.Fatal("expected both modes")
	}
	if res.Price == nil || res.Price.Display != "£19.99" {
		t.Fatalf("price=%#v", res.Price)
	}
}
