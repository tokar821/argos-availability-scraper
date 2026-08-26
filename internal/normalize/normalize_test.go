package normalize_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tokar821/argos-availability-scraper/internal/argos"
	"github.com/tokar821/argos-availability-scraper/internal/normalize"
)

func TestNormalizeCollectionFromFixture(t *testing.T) {
	raw := readTestdata(t, "availability_collection_london.json")
	var resp argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	out := normalize.NormalizeCollection("7885338", &resp, nil)
	if out.Status != "available" {
		t.Fatalf("status=%s message=%s", out.Status, out.Message)
	}
	if len(out.Stores) < 2 {
		t.Fatalf("expected stores, got %d", len(out.Stores))
	}
	// First store in fixture is OOS; second is available.
	if out.Stores[0].Status != "unavailable" {
		t.Fatalf("store0 status=%s", out.Stores[0].Status)
	}
	if out.Stores[1].Status != "available" {
		t.Fatalf("store1 status=%s msg=%s", out.Stores[1].Status, out.Stores[1].Message)
	}
	if out.Stores[1].Name == "" || out.Stores[1].DistanceMiles == nil {
		t.Fatalf("expected name/distance on available store: %#v", out.Stores[1])
	}
	if out.EarliestDate == "" {
		t.Fatal("expected earliest collection date")
	}
}

func TestNormalizeDeliveryFromFixture(t *testing.T) {
	raw := readTestdata(t, "availability_delivery_sw1a.json")
	var resp argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	out := normalize.NormalizeDelivery("7885338", &resp, nil)
	if out.Status != "available" {
		t.Fatalf("status=%s message=%s", out.Status, out.Message)
	}
	if out.Message == "" {
		t.Fatal("expected delivery message")
	}
	if out.EarliestDate == "" {
		t.Fatal("expected earliest delivery date")
	}
}

func TestNormalizeDeliveryEmpty(t *testing.T) {
	resp := &argos.AvailabilityResponse{Delivery: json.RawMessage("null")}
	out := normalize.NormalizeDelivery("7885338", resp, nil)
	if out.Status != "unavailable" {
		t.Fatalf("status=%s", out.Status)
	}
}

func TestBuildResultBoth(t *testing.T) {
	price := 10.0
	product := argos.ProductInfo{ID: "7885338", Title: "Hot Wheels", Price: &price, URL: "https://www.argos.co.uk/product/7885338"}
	var collection argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(readTestdata(t, "availability_collection_london.json")), &collection); err != nil {
		t.Fatal(err)
	}
	var delivery argos.AvailabilityResponse
	if err := json.Unmarshal([]byte(readTestdata(t, "availability_delivery_sw1a.json")), &delivery); err != nil {
		t.Fatal(err)
	}
	res := normalize.BuildResult(product, "SW1A 1AA", "both", time.Date(2026, 8, 26, 21, 0, 0, 0, time.UTC), &collection, &delivery, nil, nil)
	if res.Collection == nil || res.Delivery == nil {
		t.Fatal("expected both modes")
	}
	if res.Price == nil || res.Price.Display != "£10.00" {
		t.Fatalf("price=%#v", res.Price)
	}
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
