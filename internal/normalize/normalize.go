package normalize

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tokar821/argos-availability-scraper/internal/argos"
	"github.com/tokar821/argos-availability-scraper/internal/model"
)

func BuildResult(product argos.ProductInfo, location, mode string, checkedAt time.Time, collection, delivery *argos.AvailabilityResponse, collectionErr, deliveryErr error) model.Result {
	res := model.Result{
		ProductID:  product.ID,
		Title:      product.Title,
		Location:   location,
		Mode:       mode,
		CheckedAt:  checkedAt.UTC(),
		ProductURL: product.URL,
	}
	if product.Price != nil {
		res.Price = &model.Price{
			Amount:   round2(*product.Price),
			Currency: "GBP",
			Display:  fmt.Sprintf("£%.2f", *product.Price),
		}
	}

	wantCollection := mode == "collection" || mode == "both"
	wantDelivery := mode == "delivery" || mode == "both"

	if wantCollection {
		res.Collection = NormalizeCollection(product.ID, collection, collectionErr)
	}
	if wantDelivery {
		res.Delivery = NormalizeDelivery(product.ID, delivery, deliveryErr)
	}
	return res
}

func NormalizeCollection(productID string, resp *argos.AvailabilityResponse, err error) *model.ModeResult {
	if err != nil {
		return &model.ModeResult{
			Status: "error",
			Error:  mapError(err),
		}
	}
	if resp == nil {
		return &model.ModeResult{Status: "unknown", Message: "no collection response"}
	}

	out := &model.ModeResult{
		Stores: make([]model.Store, 0, len(resp.Stores)),
	}

	var earliest time.Time
	var earliestSet bool
	var bestMsg string
	availableCount := 0

	for _, s := range resp.Stores {
		store := normalizeStore(productID, s)
		out.Stores = append(out.Stores, store)
		if store.Status != "available" {
			continue
		}
		availableCount++
		t, hasTime := argos.ParseTime(store.EarliestDate)
		window := store.EarliestWindow
		if window == "" {
			window = store.Message
		}
		if !earliestSet || (hasTime && t.Before(earliest)) || (!hasTime && bestMsg == "") {
			if hasTime {
				earliest = t
				earliestSet = true
			}
			if bestMsg == "" || hasTime {
				if store.Message != "" {
					bestMsg = store.Message
				}
				if window != "" {
					out.EarliestWindow = window
				}
			}
		}
	}

	switch {
	case availableCount > 0:
		out.Status = "available"
		out.Message = bestMsg
		if out.Message == "" {
			out.Message = fmt.Sprintf("Available for collection at %d nearby store(s)", availableCount)
		}
		if earliestSet {
			out.EarliestDate = earliest.Format(time.RFC3339)
		}
	case len(resp.Stores) > 0:
		out.Status = "unavailable"
		out.Message = firstStoreMessage(productID, resp.Stores)
		if out.Message == "" {
			out.Message = "No nearby stores have stock for collection"
		}
	default:
		out.Status = "unavailable"
		out.Message = "No collection stores returned for this location"
	}
	return out
}

func NormalizeDelivery(productID string, resp *argos.AvailabilityResponse, err error) *model.ModeResult {
	if err != nil {
		return &model.ModeResult{
			Status: "error",
			Error:  mapError(err),
		}
	}
	if resp == nil {
		return &model.ModeResult{Status: "unknown", Message: "no delivery response"}
	}

	entries, parseErr := argos.ParseDeliveryPayload(resp.Delivery)
	if parseErr != nil {
		return &model.ModeResult{
			Status:  "error",
			Message: parseErr.Error(),
			Error:   &model.ErrorInfo{Code: "parse_error", Message: parseErr.Error()},
		}
	}
	out := &model.ModeResult{}
	if len(entries) == 0 {
		// Town searches often return stores with origin= but delivery=null.
		out.Status = "unavailable"
		out.Message = "No delivery options returned for this location (a UK postcode usually works best)"
		return out
	}

	entry := entries[0]
	msg := messageForSKU(productID, entry.Messages)
	if msg == nil {
		if m, ok := entry.Messages["delivery_summary"]; ok {
			msg = &m
		}
	}
	avail := firstAvailability(entry.Availability)

	fee := extractFee(entry)
	if fee != nil {
		out.Fee = fee
	}

	qty := 0
	if avail != nil {
		qty = avail.QuantityAvailable
	}

	switch {
	case qty > 0 || (msg != nil && !isOutOfStock(msg.MessageKey, msg.Text)):
		out.Status = "available"
		if msg != nil {
			out.Message = msg.Text
			out.EarliestWindow = msg.Text
			out.RawSummary = msg.Text
		} else {
			out.Message = "Delivery available"
		}
		if avail != nil {
			if t, ok := argos.ParseTime(avail.CustomerAvailableBy); ok {
				out.EarliestDate = t.Format(time.RFC3339)
			} else if t, ok := argos.ParseTime(avail.LeadTime); ok {
				out.EarliestDate = t.Format(time.RFC3339)
			}
		}
	default:
		out.Status = "unavailable"
		if msg != nil {
			out.Message = msg.Text
		} else {
			out.Message = "Delivery not available for this location"
		}
	}
	return out
}

func normalizeStore(productID string, s argos.StoreEntry) model.Store {
	info := s.StoreInfo
	name := info.LegacyName
	if name == "" {
		name = s.StoreID
	}
	msg := messageForSKU(productID, s.Messages)
	if msg == nil {
		if m, ok := s.Messages["collection_summary"]; ok {
			msg = &m
		}
	}
	avail := firstAvailability(s.Availability)
	store := model.Store{
		ID:            firstNonEmpty(info.StoreID, s.StoreID),
		Name:          name,
		Address:       argos.FormatAddress(info.AddressLine, info.Postcode),
		Postcode:      info.Postcode,
		Town:          info.Town,
		DistanceMiles: argos.ParseDistanceMiles(s.Distance),
	}
	qty := 0
	if avail != nil {
		qty = avail.QuantityAvailable
		store.QuantityAvailable = qty
		if t, ok := argos.ParseTime(avail.CustomerAvailableBy); ok {
			store.EarliestDate = t.Format(time.RFC3339)
		}
	}
	if msg != nil {
		store.Message = msg.Text
		store.EarliestWindow = msg.Text
		if qty > 0 || !isOutOfStock(msg.MessageKey, msg.Text) {
			store.Status = "available"
		} else {
			store.Status = "unavailable"
		}
	} else if qty > 0 {
		store.Status = "available"
		store.Message = "In stock for collection"
	} else {
		store.Status = "unavailable"
		store.Message = "Not in stock here"
	}
	return store
}

func messageForSKU(productID string, messages map[string]argos.Message) *argos.Message {
	if messages == nil {
		return nil
	}
	if m, ok := messages[productID]; ok {
		return &m
	}
	return nil
}

func firstAvailability(items []argos.SKUAvailability) *argos.SKUAvailability {
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func firstStoreMessage(productID string, stores []argos.StoreEntry) string {
	for _, s := range stores {
		if m := messageForSKU(productID, s.Messages); m != nil && m.Text != "" {
			return m.Text
		}
	}
	return ""
}

func isOutOfStock(key, text string) bool {
	k := strings.ToUpper(key)
	t := strings.ToLower(text)
	if strings.Contains(k, "OUT_OF_STOCK") || strings.Contains(k, "UNAVAILABLE") {
		return true
	}
	if strings.Contains(t, "not in stock") || strings.Contains(t, "out of stock") || strings.Contains(t, "unavailable") {
		return true
	}
	return false
}

func extractFee(entry argos.DeliveryEntry) *model.Fee {
	candidates := []*argos.DeliveryFee{entry.Fee, entry.Cost, entry.DeliveryFee}
	for _, c := range candidates {
		if c == nil {
			continue
		}
		amount, ok := anyToFloat(c.Amount)
		if !ok {
			amount, ok = anyToFloat(c.Value)
		}
		if !ok && c.Display != "" {
			return &model.Fee{Display: c.Display, Currency: "GBP"}
		}
		if ok {
			cur := c.Currency
			if cur == "" {
				cur = "GBP"
			}
			display := c.Display
			if display == "" {
				display = fmt.Sprintf("£%.2f", amount)
			}
			return &model.Fee{Amount: round2(amount), Currency: cur, Display: display}
		}
	}
	return nil
}

func anyToFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		var f float64
		_, err := fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(t, "£")), "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

func mapError(err error) *model.ErrorInfo {
	msg := err.Error()
	code := "request_error"
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "access denied") || strings.Contains(lower, "blocked"):
		code = "blocked"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		code = "timeout"
	case strings.Contains(lower, "not found"):
		code = "not_found"
	case strings.Contains(lower, "invalid"):
		code = "invalid_input"
	}
	return &model.ErrorInfo{Code: code, Message: msg}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
