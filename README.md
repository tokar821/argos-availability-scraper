# Argos Collection & Delivery Scraper

Go CLI that queries Argos at runtime for live product fulfilment availability, including collection and home delivery, for a supplied UK postcode or town.

This is a **Chromium-backed Argos API client**: the tool loads the public Argos page in headed Chromium and performs availability requests from that page context via Argos's first-party JSON endpoint.

---

## Prerequisites

- Go 1.22+
- Google Chrome, Chromium, or Microsoft Edge
- Network access to `www.argos.co.uk`

## Installation

Clone the repository and download dependencies:

```bash
go mod download
```

## Build

```bash
go mod tidy
go build -o argos ./cmd/argos        # Linux/macOS
go build -o argos.exe ./cmd/argos    # Windows
```

## Usage

Accepts any Argos product ID or product URL. Product and location values are always taken from the command line — nothing is hardcoded.

```bash
# Both collection and delivery (supply a UK postcode for delivery)
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
| `--proxy` | | Optional HTTP(S) proxy URL (else `HTTP_PROXY` / `HTTPS_PROXY`) |

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

A full example is in [`samples/sample_output.json`](samples/sample_output.json). The sample was captured from a real Argos request during development; availability values are not hardcoded into the application.

### Errors

Failures return structured JSON — never silent errors or fabricated availability:

```json
{
  "error": { "code": "blocked", "message": "blocked: access denied to product page" }
}
```

| Exit code | Meaning |
|-----------|---------|
| 0 | Success |
| 2 | Invalid input |
| 3 | Product not found |
| 4 | Blocked (access denied) |
| 5 | Timeout |
| 1 | Other error |

Exit codes are intended for scripting and CI integration; the JSON `error` object provides the detailed failure reason.

If Argos does not permit the TLS session or API call, the program returns a structured `blocked` error rather than attempting to work around the protection.

---

## Approach

Plain Go `net/http` requests to Argos returned `403 Access Denied` because Akamai fingerprints TLS (JA4). Stock browser profiles in tls-client (through Chrome 146) also mismatched real Chrome 152.

This tool uses **direct programmatic HTTPS** via [bogdanfinn/tls-client](https://github.com/bogdanfinn/tls-client) with a **custom Chrome 152 ClientHello** (ML-DSA signature schemes + trust_anchors) that matches the local Chrome JA4, then:

1. GETs the public Argos product page (cookie jar)
2. GETs Argos's first-party availability JSON with Chrome-like headers

It does **not**:

- Automate a browser UI
- Solve CAPTCHAs
- Bypass authentication
- Route through a paid extraction or bypass service

If the session is blocked, the program reports a structured failure.

### Architecture

```
     tls-client (Chrome 152 JA4)
                 │
                 ▼
       Argos first-party HTTP
                 │
          ┌──────┴──────┐
          ▼             ▼
      Collection     Delivery
       origin=       postcode=
          │             │
          └──────┬──────┘
                 ▼
            Normalizer
                 │
                 ▼
          Stable JSON model
```

| Package | Responsibility |
|---------|----------------|
| `cmd/argos` | CLI, orchestration, output, exit codes |
| `internal/argos` | Chrome-152 TLS HTTP client, first-party API calls, HTML parsing |
| `internal/normalize` | Raw Argos JSON → stable output schema |
| `internal/model` | Output types (decoupled from Argos API shapes) |

### Availability API

**Endpoint** (discovered from Argos's own network traffic in browser DevTools):

```
GET /stores/api/orchestrator/v0/locator/availability
```

The endpoint was identified by observing network requests made by the Argos website during a normal stock-check interaction; the implementation does not invent or emulate a separate availability data source.

This is Argos's internal locator orchestrator API — used by the public website but not formally documented as a public integration surface. It may change without notice. If it becomes unavailable or changes shape, the program returns a structured error rather than fabricated data.

| Mode | Query param | Meaning |
|------|-------------|---------|
| Collection | `origin=` | Location anchor accepted by the Argos locator API; town names and postcodes are supported where Argos resolves them. The value from `--location` is passed directly as `origin`. |
| Delivery | `postcode=` | Delivery destination. A UK postcode should be supplied; the underlying API expects a delivery postcode. |

Both modes also pass `skuQty=<product-id>_1` (and related locator params observed in site traffic).

**Important:** using only `origin=` returns `"delivery": null` even when home delivery exists. Delivery requires `postcode=`.

### Design decisions

| Decision | Reason |
|----------|--------|
| Custom Chrome 152 TLS profile over stock `net/http` | Plain TLS / older Chrome profiles JA4-mismatch Chrome 152 and get 403 |
| First-party API calls over UI automation | Same data source, faster and more stable |
| Separate `normalize` package | Keeps Argos API shapes out of the output contract |
| Run-level timeout | When mode is `both`, steps run sequentially (product page → collection → delivery) with an overall context timeout of `3×` `--timeout` (e.g. 225s at default 75s). Each step also has its own per-request timeout. |
| Structured `blocked` errors | No CAPTCHA solving or circumvention when access is denied |

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

Fixtures are synthetic JSON/HTML embedded in test files. They are not used at runtime. The fixtures only test parser and normalization behaviour; runtime availability always comes from Argos.

---

## Limitations

- Delivery availability is queried using Argos's `postcode=` parameter. A UK postcode should be supplied for delivery checks; a town name may produce no delivery result because the underlying API expects a delivery postcode.
- Delivery fee is included only when Argos returns it in the API response.
- The availability endpoint is undocumented and may change; the program depends on Argos's current website behaviour.
- Chrome major-version drift can break the JA4 match; the ClientHello in `internal/argos/chrome152.go` may need updating when Chrome adds new TLS features.
- Optional residential proxy (`--proxy` / `HTTP_PROXY`) can help on datacenter IPs; not required when the TLS fingerprint already matches.

---

## Access & Compliance

The scraper does not attempt to defeat Argos access controls.

It does not:

- solve or bypass CAPTCHAs
- bypass authentication
- use CAPTCHA-solving or anti-bot services
- use paid extraction APIs
- rotate credentials or otherwise circumvent access restrictions

If Argos denies access to the TLS session or availability request, the scraper returns a structured `blocked` error.

---

## Dependencies

| Dependency | Purpose |
|------------|---------|
| [bogdanfinn/tls-client](https://github.com/bogdanfinn/tls-client) | Browser-like TLS/HTTP2 client (open-source) |
| Custom Chrome 152 profile (`internal/argos/chrome152.go`) | JA4 match for current Chrome |
| `www.argos.co.uk` | Public product pages and first-party availability API |

No paid extraction APIs, CAPTCHA solvers, or third-party bypass services are used.
