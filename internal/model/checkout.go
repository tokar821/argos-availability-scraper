package model

import "time"

type CheckoutResult struct {
	ProductID  string     `json:"product_id"`
	Postcode   string     `json:"postcode"`
	Fulfilment string     `json:"fulfilment"`
	SnapshotID string     `json:"snapshot_id,omitempty"`
	RedirectTo string     `json:"redirect_to,omitempty"`
	CheckedAt  time.Time  `json:"checked_at"`
	Error      *ErrorInfo `json:"error,omitempty"`
}
