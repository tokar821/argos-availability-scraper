# Argos Collection & Delivery Scraper

Go CLI that retrieves **live** Argos product availability for **collection** and/or **delivery** using a UK postcode or town.

**Trial:** Tokar / Adventure AIO — Argos availability scraper (Go)

---

## Quick start

### Prerequisites

- Go 1.22+
- Google Chrome, Chromium, or Microsoft Edge installed
- Network access to `www.argos.co.uk`

### Build

```bash
go mod tidy
go build -o argos ./cmd/argos        # Linux/macOS
go build -o argos.exe ./cmd/argos    # Windows
```

### Run

Works with **any** valid Argos product ID or product URL — nothing is hardcoded at runtime.

```bash
# Both modes — any product ID + UK postcode (best for delivery)
./argos --product <product-id> --location "<postcode>" --mode both

# Any product URL + town — collection
./argos --product https://www.argos.co.uk/product/<product-id> --location "<town>" --mode collection

# Positional shorthand
./argos <product-id-or-url> "<location>" both
```

Example (replace with whatever the reviewer gives you):

```bash
./argos --product 12345678 --location "M1 1AE" --mode both
```

**Output:** human-readable summary on **stderr**, normalized JSON on **stdout**.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--product` | | Argos product ID or product URL |
| `--location` | | UK postcode or town |
| `--mode` | `both` | `collection`, `delivery`, or `both` |
| `--json` | `true` | Print normalized JSON to stdout |
| `--quiet` | `false` | Suppress stderr console summary |
| `--timeout` | `75s` | Per-request timeout |
| `--headless` | `false` | Headless Chrome (usually blocked by Akamai) |
| `--chrome` | | Optional path to Chrome/Edge binary |

---

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────────────────┐
│  CLI        │────▶│  Chrome session  │────▶│  Argos locator JSON API     │
│  cmd/argos  │     │  (chromedp)      │     │  /stores/api/orchestrator/  │
└─────────────┘     └────────┬─────────┘     └─────────────────────────────┘
                             │
                             ▼
                    ┌──────────────────┐     ┌──────────────────┐
                    │  Parse PDP HTML  │     │  Normalize JSON  │
                    │  title / price   │     │  internal/       │
                    └──────────────────┘     │  normalize       │
                                             └────────┬─────────┘
                                                      ▼
                                             ┌──────────────────┐
                                             │  Result JSON +   │
                                             │  console summary │
                                             └──────────────────┘
```

### Why Chrome?

Argos is protected by **Akamai bot management**. Plain `net/http`, curl, and TLS-fingerprint clients return `403 Access Denied` in testing.

This tool uses **headed Chrome** (via open-source [chromedp](https://github.com/chromedp/chromedp)) only to:

1. Load the product page like a normal browser
2. Call the **same JSON HTTP APIs** the website uses via in-page `fetch()`

It does **not** automate the UI (no typing postcode, no clicking Check, no scraping the modal). It does **not** use paid anti-bot APIs, CAPTCHA solvers, or credential bypass.

### API discovery (from live HAR + DevTools)

Both collection and delivery use one endpoint with different query params:

| Mode | Query param | Example |
|------|-------------|---------|
| Collection | `origin=` | `.../availability?origin=<location>&skuQty=<product-id>_1&...` |
| Delivery | `postcode=` | `.../availability?postcode=<postcode>&skuQty=<product-id>_1&...` |

Using only `origin=` returns `"delivery": null` even when home delivery exists — delivery requires `postcode=`.

### Project layout

```
cmd/argos/           CLI entrypoint
internal/argos/      Chrome client, product parsing, API types
internal/normalize/  Availability → normalized JSON
internal/model/      Output structs
samples/             Example JSON from a real run
```

All product and location values come from CLI arguments at runtime — nothing is hardcoded in the application code.

### Package responsibilities

| Package | Role |
|---------|------|
| `cmd/argos` | CLI flags, orchestration, JSON output, exit codes |
| `internal/argos` | Chrome session, Argos API calls, product HTML parsing |
| `internal/normalize` | Maps raw Argos JSON → stable output schema |
| `internal/model` | Output contract (decoupled from Argos API shapes) |

Data flow: **CLI → fetch product → fetch availability → normalize → stdout JSON + stderr summary**.

---

## Output

### Required fields (assignment)

| Field | JSON key | Notes |
|-------|----------|-------|
| Product title | `title` | From PDP JSON-LD / meta |
| Product ID | `product_id` | Parsed from URL or flag |
| Price | `price.amount`, `price.display` | When present on PDP |
| Location | `location` | User-supplied postcode/town |
| Timestamp | `checked_at` | UTC RFC3339 |
| Collection status | `collection.status` | `available` / `unavailable` / `error` |
| Delivery status | `delivery.status` | Same |
| Store name / distance | `collection.stores[]` | When collection checked |
| Earliest window | `*.earliest_date`, `*.earliest_window` | When Argos provides it |
| Delivery fee | `delivery.fee` | Only when API includes it |

### Example (abbreviated)

See [`samples/sample_output.json`](samples/sample_output.json) — one example live capture for submission. The reviewer will use **their own** product URLs during the review.

Regenerate with **any** product you choose:

```bash
go build -o argos ./cmd/argos
./argos --product <product-id> --location "<location>" --mode both --quiet > samples/sample_output.json
./argos --product <product-id> --location "<location>" --mode both 2> samples/sample_run_stderr.txt
```

```json
{
  "product_id": "<from-live-run>",
  "title": "<from-live-run>",
  "location": "<your-input>",
  "mode": "both",
  "collection": { "status": "available", "stores": [] },
  "delivery": { "status": "available" }
}
```

### Structured errors

Failures return JSON with an `error` object or per-mode `error` — never silent failure or fabricated stock:

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

## Tests

```bash
go test ./...
```

Unit tests cover core parsing and normalization (no network, no Chrome):

- Product ID / URL resolution (multiple arbitrary IDs and URL formats)
- Product HTML parsing (title, price, not-found, blocked page)
- Collection normalization (available, unavailable, API errors)
- Delivery normalization (available with fee, empty delivery, full result build)

Test fixtures are **inline synthetic JSON/HTML** inside test files — they mirror Argos response shape only and are never used at runtime.

---

## Assumptions & limitations

- **Chrome required** on the machine running the scraper (headed mode).
- **Delivery works best with UK postcodes** (`postcode=`). Town names (e.g. `London`) work well for collection but may return no delivery options.
- **Delivery fee** is included only when Argos returns it in the API payload (often omitted for small items).
- **Akamai rate limiting**: repeated rapid requests can trigger temporary `403`. Wait and retry on a normal desktop network.
- **No hard-coded availability** at runtime.

---

## Dependencies & external services

| Dependency | Purpose |
|------------|---------|
| [chromedp](https://github.com/chromedp/chromedp) | Chrome DevTools Protocol (open-source) |
| Google Chrome / Edge | Browser runtime |
| `www.argos.co.uk` | Product pages + locator orchestrator API |

**Not used:** paid extraction APIs, CAPTCHA solvers, anti-bot bypass services.

---

## Tradeoffs (for review)

| Choice | Rationale |
|--------|-----------|
| Chrome + `fetch()` vs pure HTTP | Pure HTTP blocked by Akamai; Chrome matches assignment allowance and avoids paid bypass |
| API calls vs UI automation | Faster, stable, same data source as the website's Check stock flow |
| `origin=` vs `postcode=` | Discovered from HAR; delivery requires `postcode=` |
| Headed vs headless | Headed passes Akamai; headless usually blocked |
| Overall run timeout | `3×` per-request timeout when mode is `both` (product + collection + delivery) |
| Inline unit tests | Covers parsing/normalization without network or fake live stock |

---

## Live review talking points

Be ready to explain these on the call (5% of score):

| Topic | Answer |
|-------|--------|
| Why Chrome? | Akamai blocks plain HTTP; headed browser + same-origin `fetch()` to Argos JSON API |
| Why not UI automation? | Same data as Check stock modal, but faster and more stable |
| `origin=` vs `postcode=` | Same endpoint; collection uses town/postcode as origin, delivery requires postcode |
| Error strategy | Structured JSON + stderr summary + exit codes — never silent failure or fabricated stock |
| Package layout | Thin CLI, integration in `argos`, normalization in `normalize`, output types in `model` |

---

## Scoring coverage

| Area | Weight | Coverage |
|------|--------|----------|
| Functional correctness | 40% | Live CLI, both modes, all required JSON fields, any product URL/ID |
| Resilience & error handling | 25% | Structured errors, exit codes, unavailable stock, run-level timeout |
| Idiomatic Go / design | 20% | `cmd` + `internal` layout, separated fetch / normalize / model |
| Tests & documentation | 10% | `go test ./...` + README + `samples/sample_output.json` |
| Explanation & tradeoffs | 5% | README tradeoffs + live review talking points above |

---

## Assignment compliance checklist

| Requirement | Status |
|-------------|--------|
| Accept product URL/ID, location, mode | ✅ CLI flags + positional args |
| Live runtime fetch (no fake stock) | ✅ Chrome + Argos API |
| Title, ID, price, location, timestamp, status | ✅ `Result` JSON |
| Earliest date/window, fee, store/distance | ✅ When API provides |
| JSON + console summary | ✅ stdout JSON, stderr summary |
| Invalid / unavailable / timeout / block handling | ✅ Structured errors + exit codes |
| go.mod / go.sum / build instructions | ✅ This README |
| README: approach, assumptions, limits, services | ✅ Above |
| Automated parsing/normalization tests | ✅ `go test ./...` |
| Sample JSON from real run | ✅ `samples/sample_output.json` |

---

## How the reviewer will test

The assignment says a **live code review + functional review** happens after submission. Expect this flow:

### 1. Build & tests

```bash
go mod tidy
go test ./...
go build -o argos ./cmd/argos
```

Unit tests run offline (no Argos calls). Then they run live against Argos.

### 2. Run live against Argos (pass condition)

From the README, they run your CLI with **real product URLs/IDs** and a **postcode or town**:

```bash
./argos --product <id-or-url> --location "<postcode-or-town>" --mode both
```

They will likely supply **two product pages themselves** during the review (not just your sample). You should be ready to run those live on the call.

**Pass condition:** truthful live JSON (or a clear structured error like `blocked`, `not_found`, `timeout`) — not hard-coded or fake stock.

### 3. What they check in the output

| Check | Where |
|-------|--------|
| Product title, ID, price | Top-level JSON |
| Location + timestamp | `location`, `checked_at` |
| Collection status + stores | `collection` |
| Delivery status + window | `delivery` |
| Console summary | stderr (unless `--quiet`) |
| Bad input / blocked handling | `error.code`, exit codes |

### 4. Code review questions

Be ready to explain (5% of score):

- Why Chrome + `fetch()` instead of plain HTTP
- Why `origin=` vs `postcode=` for collection vs delivery
- How errors are normalized
- Tradeoffs (Chrome dependency, postcode for delivery, etc.)

### 5. Sample JSON in repo

They may compare [`samples/sample_output.json`](samples/sample_output.json) to a fresh run you do in the meeting — timestamps will differ; structure and live statuses should match Argos.

---

## Live review commands

For the reviewer demo, run any two Argos product pages:

```bash
go build -o argos ./cmd/argos
./argos --product <reviewer-product-id-or-url> --location "<reviewer-location>" --mode both
```

Ensure Chrome is installed and visible (headed window may flash briefly).
