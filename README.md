# Argos Collection & Delivery Scraper

Go CLI that retrieves **live** Argos collection and home-delivery availability for a UK postcode or town using **programmatic HTTPS only** (no browser automation).

Runtime: custom Chrome 152 TLS/HTTP2 profile ([bogdanfinn/tls-client](https://github.com/bogdanfinn/tls-client)), cookie-jar session, Chrome-like headers, optional HTTP(S) proxy via `.env` / `--proxy`.

---

## Requirements

- Go **1.24+** (`go.mod`)
- Network access to `https://www.argos.co.uk`
- Optional: HTTP(S) proxy when the host egress IP is blocked (common on datacenter IPs)

Chrome/Chromium is **not** required. The TLS profile is compiled into the binary.

## Installation

```bash
go mod download
```

Optional local config (gitignored — never commit secrets):

```bash
cp .env.example .env
# edit HTTP_PROXY / HTTPS_PROXY if needed
```

`.env` is loaded automatically on startup. Existing process environment variables are not overwritten.

## Build

```bash
go mod tidy
go build -o argos ./cmd/argos        # Linux/macOS
go build -o argos.exe ./cmd/argos    # Windows
```

## Usage

Inputs are always taken from the CLI — product IDs, locations, and availability are never hardcoded.

```bash
# collection + delivery (prefer a UK postcode for delivery)
./argos --product <product-id> --location "<postcode>" --mode both

# collection only (town or postcode)
./argos --product https://www.argos.co.uk/product/<id> --location "<town>" --mode collection

# delivery only (UK postcode)
./argos --product <product-id> --location "<postcode>" --mode delivery

# positional shorthand
./argos <product-id-or-url> "<location>" [mode]
```

Example:

```bash
./argos --product 7885338 --location "M1 1AE" --mode both
```

- **stdout** → structured JSON  
- **stderr** → human-readable summary  

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--product` | | Argos product ID or product URL |
| `--location` | | UK postcode or town |
| `--mode` | `both` | `collection`, `delivery`, or `both` |
| `--json` | `true` | Print JSON to stdout |
| `--quiet` | `false` | Suppress stderr summary |
| `--timeout` | `75s` | Per-request timeout |
| `--proxy` | | HTTP(S) proxy URL (else `HTTP_PROXY` / `HTTPS_PROXY` / `.env`) |

### Proxy setup (optional)

If direct requests return `blocked` / HTTP 403 on the availability API:

1. Copy `.env.example` → `.env`
2. Set a UK residential proxy, e.g.:

```env
HTTP_PROXY=http://user:pass@proxy-host:port
HTTPS_PROXY=http://user:pass@proxy-host:port
```

3. Rebuild is not required; run `./argos ...` again from the repo root (`.env` is auto-loaded).

Or pass once:

```bash
./argos --product 7885338 --location "M1 1AE" --mode both --proxy "http://user:pass@proxy-host:port"
```

Development used BottingTools UK residential with a sticky session in the username. See [External Services](#external-services).

---

## Output

### Fields

| Field | JSON key | Notes |
|-------|----------|-------|
| Product title | `title` | Product page JSON-LD / meta |
| Product ID | `product_id` | From URL or `--product` |
| Price | `price` | `null` if Argos does not expose it |
| Location | `location` | Echo of `--location` |
| Timestamp | `checked_at` | UTC RFC3339 |
| Collection | `collection` | Present when mode includes collection |
| Delivery | `delivery` | Present when mode includes delivery |

**Collection** (when Argos provides data): `status`, `message`, `earliest_date`, `earliest_window`, `stores[]` (`name`, `distance_miles`, …).

**Delivery** (when Argos provides data): `status`, `message`, `earliest_date`, `earliest_window`, `fee`.

Optional fields are omitted or `null` when Argos does not return them — never fabricated.

Status values: `available` | `unavailable` | `unknown` | `error`.  
A transport/session failure uses `error` / top-level `error` — **not** reported as stock unavailability.

### Examples

- Live JSON sample: [`samples/sample_output.json`](samples/sample_output.json)
- Console-style sample: [`samples/sample_run_stderr.txt`](samples/sample_run_stderr.txt)

### Errors

```json
{
  "error": { "code": "blocked", "message": "blocked: access denied to product page" }
}
```

| Exit | Meaning |
|------|---------|
| 0 | Success |
| 2 | Invalid input |
| 3 | Product not found |
| 4 | Blocked / access denied |
| 5 | Timeout |
| 1 | Other request/parse error |

---

## Architecture

```text
CLI / .env
    ↓
HTTP client (Chrome 152 TLS + cookie jar + optional proxy)
    ↓
Argos product page + locator API
    ↓
Parser / normalizer
    ↓
JSON (stdout) + console summary (stderr)
```

| Package | Role |
|---------|------|
| `cmd/argos` | Flags, `.env` load, orchestration, exit codes |
| `internal/argos` | TLS client, product HTML, collection/delivery requests |
| `internal/normalize` | Argos JSON → stable schema |
| `internal/model` | Output types |

### Request implementation

1. Build tls-client with custom Chrome 152 ClientHello, HTTP/2 settings, cookie jar, optional proxy.
2. `GET` product page → parse title/price; store `Set-Cookie`.
3. `GET` `/stores/api/orchestrator/v0/locator/availability`  
   - Collection: `origin=<location>`  
   - Delivery: `postcode=<location>`  
   with Chrome-like headers, cookies, and product `Referer`.
4. Normalize live JSON; map failures to structured errors.

| Concern | Handling |
|---------|----------|
| Sessions | One shared HTTP client + cookie jar for the process run |
| Cookies | Automatic jar after product-page response |
| Tokens | Session cookies from Argos `Set-Cookie` are the only credentials used; no CAPTCHA/bearer/API-key token is required for this first-party flow |
| Headers / UA | Chrome 152 `User-Agent`, `sec-ch-ua*`, `sec-fetch-*`, API `x-newrelic-id` |
| Redirects | Followed by the HTTP client |
| TLS | Custom JA4-aligned profile in `internal/argos/chrome152.go` |
| Proxy | `--proxy` or `HTTP_PROXY` / `HTTPS_PROXY` (including `.env`) |

### Availability API

```text
GET /stores/api/orchestrator/v0/locator/availability
```

Same first-party endpoint the Argos website uses. Undocumented; may change.

---

## Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/bogdanfinn/tls-client` | TLS/HTTP2 client |
| `github.com/bogdanfinn/fhttp` | HTTP types used by tls-client |
| `github.com/bogdanfinn/utls` | ClientHello construction |

See `go.mod` / `go.sum` for exact versions.

## External services

### bogdanfinn/tls-client (required)

| | |
|--|--|
| Purpose | Programmatic Chrome-like TLS/HTTP2 |
| Integration | `internal/argos/client.go` + `chrome152.go` |
| Cost | Free (open-source) |
| Limits | Chrome TLS drift can break JA4 match |
| Risks | Upgrade breakage; temporary Argos 403 until profile update |
| Setup | `go mod download` |

### HTTP(S) residential proxy (optional)

| | |
|--|--|
| Provider (dev) | [BottingTools](https://bottingtools.com) UK residential (sticky session in username) |
| Purpose | Trusted UK egress when datacenter IPs are denied |
| Integration | `.env` `HTTP_PROXY`/`HTTPS_PROXY` or `--proxy` → tls-client `WithProxyUrl` |
| Cost | Provider plan / bandwidth (not included in this repo). Direct residential/home IPs often need **no** proxy (**$0**) |
| Limits | Sticky IP burn, latency, still depends on Argos accepting the exit |
| Risks | Outage, credential leak if `.env` is committed, cost if traffic is high |
| Setup | `cp .env.example .env` and set proxy URLs; run from repo root |

### Evaluated, not shipped

| Service | Cost if used | Why not shipped |
|---------|--------------|-----------------|
| Hyper Solutions (Akamai sensors) | ~€3 / 1 000 sensors | Not needed after Chrome 152 TLS match |
| chromedp / headed Chrome | Free software | Forbidden as final runtime |

**No paid extraction API and no CAPTCHA solver in the final runtime.**

---

## Testing

```bash
go test ./...
```

Offline tests cover product ID/URL parsing, HTML title/price, collection/delivery normalization, store/distance/date/fee extraction, missing optional fields, unavailable stock, malformed payloads, and error mapping. Fixtures are embedded in `*_test.go` — not used at runtime.

---

## Assumptions

- Reviewer supplies a real Argos product ID/URL and a UK location.
- Delivery mode should use a **UK postcode**; towns work best for collection (`origin=`).
- Argos continues to expose the locator orchestrator endpoint used by the public site.
- A correct Chrome-major TLS fingerprint is sufficient for access on a good IP; proxy is only needed when IP reputation fails.
- Session state for this flow is cookie-based; Argos does not require a separate developer API token for the public locator calls.
- Availability values always come from live Argos responses for that run.
- Delivery `fee` is included only when present in the Argos payload; otherwise it is omitted (never fabricated).

## Known limitations

- Locator API is undocumented and can change shape or path.
- Chrome major upgrades may require updating `chrome152.go`.
- Delivery fee appears only when Argos includes it.
- Rapid repeats from one IP can cause temporary `blocked` responses.
- Town-as-delivery-location may return empty delivery (`postcode=` expected).
- Location validation only rejects empty/too-short/non-alphanumeric input; Argos still decides whether a place resolves.

## Troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| `blocked` / HTTP 403 on product or API | Bad egress IP or burned sticky session | Set UK residential `HTTP_PROXY` in `.env`, or change `_session-…` id |
| Product works, API 403 | IP soft-block on locator | Use proxy; wait; rotate session |
| `invalid_input` / exit 2 | Bad product string or mode | Pass numeric ID or full `/product/<id>` URL; mode ∈ collection\|delivery\|both |
| `not_found` / exit 3 | Unknown SKU or product page 400/404 | Confirm the product page opens in a normal browser |
| Empty delivery with a town | API expects postcode | Re-run with e.g. `SW1A 1AA` |
| Timeout / exit 5 | Slow proxy or network | Raise `--timeout`, check proxy |
| Build fails | Old Go | Use Go 1.24+ |
| Proxy ignored | Ran outside repo / empty env | Run from repo root so `.env` loads, or pass `--proxy` |

---

## Reproduce (reviewer checklist)

```bash
go mod tidy
go build -o argos ./cmd/argos    # Windows: argos.exe
# optional: configure .env proxy if direct IP is blocked
./argos --product <reviewer-product> --location "<reviewer-postcode>" --mode both
```

Expect exit `0` with truthful `available` / `unavailable`. On deny: structured `blocked`, exit `4` — not fake stock status.
