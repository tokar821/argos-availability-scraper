# Argos Collection & Delivery Scraper

Go CLI that fetches **live** Argos product availability for **collection** and/or **delivery** for a UK postcode or town, then returns normalized JSON plus a readable console summary.

## Requirements

- Go 1.22+
- Google Chrome (or Chromium / Edge) installed
- Network access to `www.argos.co.uk`

> Argos sits behind Akamai bot management. Plain `net/http` / curl clients are typically blocked (`403 Access Denied`). This tool uses a **real headed Chrome** session and issues the same **HTTP JSON APIs** the website uses (not UI clicking). Headless Chrome is usually blocked.

## Build

```bash
go mod tidy
go build -o argos.exe ./cmd/argos
```

## Run

```bash
# Product ID + postcode, both modes
./argos.exe --product 7885338 --location "SW1A 1AA" --mode both

# Product URL + town, collection only
./argos.exe --product https://www.argos.co.uk/product/7885338 --location London --mode collection

# Positional args
./argos.exe 7885338 "SW1A 1AA" both
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--product` | | Argos product ID or product URL |
| `--location` | | UK postcode or town |
| `--mode` | `both` | `collection`, `delivery`, or `both` |
| `--json` | `true` | Print normalized JSON to stdout |
| `--quiet` | `false` | Suppress stderr console summary |
| `--timeout` | `75s` | Per-request timeout |
| `--headless` | `false` | Headless Chrome (usually blocked) |
| `--chrome` | | Optional path to Chrome/Edge binary |

Human-readable summary goes to **stderr**; JSON goes to **stdout** (easy to pipe).

## Approach

1. Resolve product ID from URL or bare ID.
2. Open the Argos PDP in headed Chrome and parse title/price (JSON-LD / meta / `itemprop`).
3. Call Argos locator HTTP API via same-origin `fetch()`:
   - **Collection**: `GET /stores/api/orchestrator/v0/locator/availability?origin=<location>&skuQty=<id>_1&maxResults=10&maxDistance=50&ssm=true`
   - **Delivery**: `GET /stores/api/orchestrator/v0/locator/availability?postcode=<location>&skuQty=<id>_1&maxResults=10&maxDistance=50`
4. Normalize stores / delivery windows into structured JSON.
5. Surface structured failures for invalid input, missing products, timeouts, and Akamai blocks.

Discovered from live HAR / Chrome DevTools: collection uses `origin=`, delivery uses `postcode=`. Calling with only `origin=` often returns `"delivery": null` even when home delivery is available.

## Output shape (abbreviated)

```json
{
  "product_id": "7885338",
  "title": "Hot Wheels Formula 1 5-Pack, Set of 5 Die-Cast Toy",
  "price": { "amount": 10.0, "currency": "GBP", "display": "£10.00" },
  "location": "SW1A 1AA",
  "mode": "both",
  "checked_at": "2026-08-26T21:30:00Z",
  "product_url": "https://www.argos.co.uk/product/7885338",
  "collection": {
    "status": "available",
    "message": "Order now, collect from 5pm tomorrow",
    "earliest_date": "2026-08-27T16:00:00Z",
    "stores": [
      {
        "name": "Farringdon Sainsbury's (Argos Collection Point)",
        "distance_miles": 0.416,
        "status": "available",
        "message": "Order now, collect from 5pm tomorrow"
      }
    ]
  },
  "delivery": {
    "status": "available",
    "message": "Next day delivery available",
    "earliest_date": "2026-08-27T23:00:00Z"
  }
}
```

Sample captured live output: [`samples/sample_output.json`](samples/sample_output.json).

## Tests

```bash
go test ./...
```

Tests cover product ID/URL parsing, HTML product extraction, and collection/delivery normalization against fixtures under `testdata/` (no live network required).

## Assumptions & limitations

- **Chrome required**: Akamai blocks non-browser TLS fingerprints from this environment. The tool does not bypass CAPTCHA or paid anti-bot services.
- **Delivery prefers postcodes**: Town names (e.g. `London`) work well for collection (`origin=`). Delivery (`postcode=`) is reliable with UK postcodes; towns may return no delivery options.
- **Delivery fee**: Included only when present in the Argos API payload. Many small-item responses omit an explicit fee field.
- **Live data only**: No hard-coded availability; fixtures are for unit tests only.
- **Rate / bot pressure**: Aggressive parallel probing can trigger temporary `403 Access Denied`. Retry with headed Chrome on a normal desktop network.

## Dependencies / external services

- Open-source: [`chromedp`](https://github.com/chromedp/chromedp) (Chrome DevTools Protocol)
- External: Argos public website / locator orchestrator API only
- No paid extraction APIs

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Invalid input |
| 3 | Not found |
| 4 | Blocked by Argos/Akamai |
| 5 | Timeout |
| 1 | Other request/parse error |
