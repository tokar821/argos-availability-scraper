package model

import "time"

// Result is the normalized availability payload returned by the CLI.
type Result struct {
	ProductID   string    `json:"product_id"`
	Title       string    `json:"title"`
	Price       *Price    `json:"price"`
	Location    string    `json:"location"`
	Mode        string    `json:"mode"`
	CheckedAt   time.Time `json:"checked_at"`
	ProductURL  string    `json:"product_url"`
	Collection  *ModeResult `json:"collection,omitempty"`
	Delivery    *ModeResult `json:"delivery,omitempty"`
	Error       *ErrorInfo  `json:"error,omitempty"`
}

type Price struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Display  string  `json:"display"`
}

type ModeResult struct {
	Status           string       `json:"status"` // available | unavailable | unknown | error
	Message          string       `json:"message,omitempty"`
	EarliestDate     string       `json:"earliest_date,omitempty"`
	EarliestWindow   string       `json:"earliest_window,omitempty"`
	Fee              *Fee         `json:"fee,omitempty"`
	Stores           []Store      `json:"stores,omitempty"`
	RawSummary       string       `json:"raw_summary,omitempty"`
	Error            *ErrorInfo   `json:"error,omitempty"`
}

type Fee struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Display  string  `json:"display"`
}

type Store struct {
	ID           string  `json:"id,omitempty"`
	Name         string  `json:"name"`
	Address      string  `json:"address,omitempty"`
	Postcode     string  `json:"postcode,omitempty"`
	Town         string  `json:"town,omitempty"`
	DistanceMiles *float64 `json:"distance_miles,omitempty"`
	Status       string  `json:"status"`
	Message      string  `json:"message,omitempty"`
	EarliestDate string  `json:"earliest_date,omitempty"`
	EarliestWindow string `json:"earliest_window,omitempty"`
	QuantityAvailable int `json:"quantity_available,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
