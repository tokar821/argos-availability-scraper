# Argos Collection & Delivery Scraper

Go CLI that retrieves **live** Argos product availability for **collection** and/or **delivery** using a UK postcode or town.

---

## Prerequisites

- Go 1.22+
- Google Chrome, Chromium, or Microsoft Edge
- Network access to `www.argos.co.uk`

## Build

```bash
go mod tidy
go build -o argos ./cmd/argos        # Linux/macOS
go build -o argos.exe ./cmd/argos    # Windows
```

## Usage

Accepts any Argos product ID or product URL. Product and location values are always taken from the command line — nothing is hardcoded.

```bash
# Both collection and delivery (use a UK postcode for best delivery results)
./argos --product <product-id> --location "<postcode>" --mode both

# Product URL with town (collection)
./argos --product https://www.argos.co.uk/product/<product-id> --location "<town>" --mode collection

# Positional shorthand
./argos <product-id-or-url> "<location>" [mode]
```

Example:

```bash
./argos --product 12345678 --location "M1 1AE" --mode both
```

**Output:** normalized JSON on **stdout**, human-readable summary on **stderr**.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--product` | | Argos product ID or product URL |
| `--location` | | UK postcode or town |
| `--mode` | `both` | `collection`, `delivery`, or `both` |
| `--json` | `true` | Print JSON to stdout |
| `--quiet` | `false` | Suppress stderr summary |
| `--timeout` | `75s` | Per-request timeout |
| `--headless` | `false` | Headless Chrome (usually blocked by Akamai) |
| `--chrome` | | Optional path to Chrome/Edge binary |

---

## Output

### Response fields

| Field | JSON key | Notes |
|-------|----------|-------|
| Product title | `title` | From product page JSON-LD / meta tags |
| Product ID | `product_id` | Parsed from URL or `--product` |
| Price | `price.amount`, `price.display` | When present on the product page |
| Location | `location` | Value passed via `--location` |
| Timestamp | `checked_at` | UTC (RFC3339) |
| Collection status | `collection.status` | `available`, `unavailable`, or `error` |
| Delivery status | `delivery.status` | Same values |
| Store name / distance | `collection.stores[]` | When collection is checked |
| Earliest window | `*.earliest_date`, `*.earliest_window` | When Argos provides them |
| Delivery fee | `delivery.fee` | When the API includes it |

A full example from a live run is in [`samples/sample_output.json`](samples/sample_output.json).

### Errors

Failures return structured JSON — never silent errors or fabricated availability:

```json
{
  "error": { "code": "blocked", "message": "blocked: Argos/Akamai denied access..." }
}
```

| Exit code | Meaning |
|-----------|---------|
| 0 | Success |
| 2 | Invalid input |
| 3 | Product not found |
| 4 | Blocked by Akamai |
| 5 | Timeout |
| 1 | Other error |

---

## Approach

Argos uses Akamai bot protection. Plain HTTP clients (curl, `net/http`) return `403 Access Denied` in testing.

This tool uses **headed Chrome** via [chromedp](https://github.com/chromedp/chromedp) to:

1. Load the product page in a normal browser session
2. Call the same JSON availability APIs the website uses via in-page `fetch()`

It does **not** automate the UI (no form filling or modal scraping). It does **not** use paid extraction APIs, CAPTCHA solvers, or credential bypass.

### Architecture

```
CLI (cmd/argos)
  → Chrome session (internal/argos)
  → Argos locator API + product page HTML
  → Normalize to output schema (internal/normalize)
  → JSON (stdout) + summary (stderr)
```

| Package | Responsibility |
|---------|----------------|
| `cmd/argos` | CLI, orchestration, output, exit codes |
| `internal/argos` | Browser session, API calls, HTML parsing |
| `internal/normalize` | Raw Argos JSON → stable output schema |
| `internal/model` | Output types (decoupled from Argos API shapes) |

### API

Collection and delivery share one endpoint with different query parameters:

| Mode | Parameter | Notes |
|------|-----------|-------|
| Collection | `origin=` | Town or postcode |
| Delivery | `postcode=` | UK postcode required for reliable results |

Endpoint: `/stores/api/orchestrator/v0/locator/availability`

Using only `origin=` returns `"delivery": null` even when home delivery exists.

### Design decisions

| Decision | Reason |
|----------|--------|
| Chrome + `fetch()` over plain HTTP | Akamai blocks non-browser clients |
| API calls over UI automation | Same data source, faster and more stable |
| Headed over headless Chrome | Headless is usually blocked by Akamai |
| Separate `normalize` package | Keeps Argos API shapes out of the output contract |
| Run-level timeout (`3×` per-request when mode is `both`) | Covers product + collection + delivery in one run |

---

## Tests

```bash
go test ./...
```

Offline unit tests cover:

- Product ID and URL parsing
- Product HTML parsing (title, price, not-found, blocked pages)
- Collection and delivery normalization
- Error mapping

Fixtures are synthetic JSON/HTML embedded in test files. They are not used at runtime.

---

## Limitations

- Chrome must be installed on the host machine (headed mode by default).
- Delivery works best with UK postcodes; town names are reliable for collection but may return no delivery options.
- Delivery fee is included only when Argos returns it in the API response.
- Repeated rapid requests may trigger temporary Akamai blocks; wait and retry on a normal desktop network.

---

## Dependencies

| Dependency | Purpose |
|------------|---------|
| [chromedp](https://github.com/chromedp/chromedp) | Chrome DevTools Protocol (open-source) |
| Google Chrome / Edge | Browser runtime |
| `www.argos.co.uk` | Product pages and availability API |

No paid extraction APIs, CAPTCHA solvers, or anti-bot bypass services are used.
