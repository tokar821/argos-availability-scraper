# 48-HOUR DEVELOPMENT ASSIGNMENT

## Argos Collection & Delivery Availability Scraper

**Prepared for Tokar • Adventure AIO**

| **WEBSITE** | **LANGUAGE** | **DEVELOPMENT WINDOW** |
| ----------- | ------------ | ---------------------- |
| [argos.co.uk](https://www.argos.co.uk/) | **Go** | **48 hours from receipt of follow-up clarification** |

---

## 1. Objective

Build a **working, request-based Go application** that retrieves **live Argos product availability** for both **Collection** and **Delivery** using a UK postcode or town.

The final runtime **must not depend on browser automation**.

The solution must perform the required availability requests programmatically and return **successful, usable, truthful live results** for reviewer-supplied Argos products and locations.

---

## 2. Required Inputs

The application must accept:

* An Argos **product URL or product ID**
* A UK **postcode or town**
* Availability mode:
  * `collection`
  * `delivery`
  * `both`

Example:

```bash
argos --product 1234567 --location "SW1A 1AA" --mode both
```

---

## 3. Required Runtime Behavior

The application must:

1. Resolve and validate the requested product.
2. Retrieve live product information from Argos.
3. Process the supplied UK postcode/town.
4. Perform live **Collection** availability requests.
5. Perform live **Delivery** availability requests.
6. Return structured results for each requested mode.
7. Include a timestamp for the live check.
8. Handle session/request state reliably.

The implementation may use normal HTTP request techniques such as:

* Cookies
* Sessions
* Required tokens
* Headers
* Redirect handling
* TLS configuration
* User-agent profiles
* Proxies where legitimately required
* An authorized/reputable third-party service where necessary

Any external service or dependency must be clearly documented and reproducible.

---

## 4. Required Output

For each request, return at minimum:

* Product title
* Product ID
* Price
* Requested location
* Timestamp
* Availability status

When Argos provides the information, also return:

### Collection

* Availability status
* Earliest collection date/window
* Nearby store name
* Store distance

### Delivery

* Availability status
* Earliest delivery date/window
* Delivery fee

Data must be returned in:

1. **Structured JSON**
2. **Readable console summary**

Optional information must not be fabricated. If Argos does not provide a particular field, it should be omitted or represented as `null` as appropriate.

---

## 5. Truthful Results & Failure Handling

The application must clearly distinguish between:

* Available
* Unavailable
* Unknown / unable to determine
* Request failure

It must handle failures cleanly, including:

* Invalid product
* Invalid location
* Product unavailable
* Collection unavailable
* Delivery unavailable
* HTTP errors
* Request timeouts
* Unexpected response formats
* Missing required data
* Session/token failures
* Blocking or access errors

A failed request must **not** be reported as product unavailability.

A `403 Forbidden` response alone is **not considered completion**. The implementation must make reasonable, legitimate efforts to establish the required request/session behavior so that live availability requests successfully work.

---

## 6. Request-Based Implementation Requirement

The final application must use **programmatic HTTP requests in Go**.

### Required

```text
Go application
    ↓
HTTP/session handling
    ↓
Argos requests
    ↓
Live availability data
    ↓
Parser/normalizer
    ↓
JSON + console output
```

### Not allowed as runtime dependencies

* Playwright
* Selenium
* Puppeteer
* Chrome/Chromium browser automation
* Browser-based scraping

The final runtime must operate without launching or controlling a browser.

---

## 7. Security & Access Constraints

The implementation must **not**:

* Bypass CAPTCHA
* Bypass authentication
* Circumvent access controls
* Defeat security mechanisms
* Use unauthorized methods to evade restrictions
* Fabricate or hard-code availability
* Depend on a manually prepared dataset

A legitimate request/session configuration may be implemented when required for normal access.

If an authorized third-party provider is used, the project must clearly document:

* Provider name
* Purpose
* Integration method
* Cost
* Limits
* Dependencies
* Operational risks
* Reproduction/setup instructions

Paid extraction APIs should not be used unless specifically justified as an authorized third-party solution.

---

## 8. Technical Expectations

The solution should be:

* Written in idiomatic Go
* Modular and maintainable
* Configurable
* Robust against normal request failures
* Clear about errors and state
* Suitable for live code review
* Reproducible from a clean environment

The implementation should clearly separate concerns such as:

```text
CLI/Input
    ↓
HTTP Client / Session
    ↓
Argos Request Layer
    ↓
Product / Location Resolution
    ↓
Collection / Delivery Availability
    ↓
Parsing & Normalization
    ↓
Output
```

The exact architecture is left to the developer.

---

## 9. Testing Requirements

Provide automated tests covering core parsing and normalization logic.

Tests should cover representative cases such as:

* Product parsing
* Price parsing
* Collection availability
* Delivery availability
* Store/distance extraction
* Date/window extraction
* Delivery fee extraction
* Missing optional fields
* Unavailable products
* Unexpected/malformed responses
* Error normalization

Live Argos requests do not need to be the primary unit-test mechanism. Core parsing/normalization should be testable independently using representative captured responses/fixtures where appropriate.

---

## 10. Documentation Requirements

Provide a clear `README.md` containing:

### Overview

What the application does and its intended use.

### Requirements

Go version and any system dependencies.

### Installation

How to install dependencies.

### Build

How to build the application.

### Usage

Examples for:

```text
collection
delivery
both
```

### Output

Example JSON and console output.

### Architecture

Explanation of the major components and request flow.

### Request Implementation

Explain how Argos requests are performed, including relevant:

* Sessions
* Cookies
* Tokens
* Headers
* Redirects
* TLS behavior
* User-agent configuration
* Proxy configuration, if applicable

### Dependencies

List all Go libraries and external dependencies.

### External Services

Clearly disclose any third-party services, including cost and limitations.

### Testing

Explain how to execute the test suite.

### Assumptions

Document important assumptions made during development.

### Known Limitations

Document known Argos/request/environment limitations.

### Troubleshooting

Explain common failures and how to diagnose them.

---

## 11. Required Submission

Submit a Git repository or ZIP containing:

```text
Source code
go.mod
go.sum
README.md
Automated tests
Required fixtures/test data
Sample JSON output
Build/run instructions
```

The sample JSON must represent a **real live run**, not manually fabricated availability data.

---

## 12. Functional Review

During the live review, the reviewer will provide **two Argos product pages/products and postcode/town inputs**.

The developer must be prepared to:

1. Build the application.
2. Run the application from the README instructions.
3. Query the reviewer-supplied products.
4. Query both Collection and Delivery.
5. Demonstrate live results.
6. Explain the request/session implementation.
7. Explain error handling.
8. Explain architecture and tradeoffs.
9. Answer questions about dependencies and external services.
10. Demonstrate that the solution is reproducible.

---

## 13. Acceptance Criteria

The project is considered complete only when:

* The reviewer can build the project from the README.
* The application runs successfully.
* Live Collection requests succeed.
* Live Delivery requests succeed.
* Reviewer-supplied products can be queried.
* Reviewer-supplied postcodes/towns can be queried.
* Cookies/session behavior is handled reliably.
* Required tokens are handled reliably.
* Required headers are handled reliably.
* Redirect behavior is handled correctly.
* TLS behavior is appropriate.
* Results are live rather than hard-coded or stale.
* Results are structured and truthful.
* Failures produce specific, structured errors.
* The runtime does not depend on browser automation.
* The implementation and any external services are documented and reproducible.
* The developer can explain and defend the implementation during code review.

**A documented 403 response by itself does not satisfy the acceptance criteria.**

---

## 14. Evaluation / Scoring

| Area | Weight |
| ---- | -----: |
| Functional correctness | **40%** |
| Resilience & error handling | **25%** |
| Idiomatic Go / design | **20%** |
| Tests & documentation | **10%** |
| Explanation & tradeoffs | **5%** |
| **Total** | **100%** |

### Functional correctness — 40%

Live Collection and Delivery requests work correctly with reviewer-supplied inputs.

### Resilience & error handling — 25%

The application handles timeouts, unavailable products, invalid inputs, unexpected responses, session problems, and blocking without crashing or producing false availability.

### Go/design — 20%

Clean, idiomatic, maintainable Go implementation with sensible separation of concerns.

### Tests/documentation — 10%

Useful automated tests and clear reproduction/documentation.

### Explanation/tradeoffs — 5%

Ability to explain the request flow, architecture, dependencies, limitations, and technical decisions.

---

## 15. Final Goal

The expected final result is a **reliable Go-based Argos availability client** that can take:

```text
Product
+
UK postcode/town
+
Collection / Delivery / Both
```

and perform live request-based checks:

```text
                  Argos
                    │
             Live HTTP requests
                    │
          ┌─────────┴─────────┐
          ▼                   ▼
     Collection            Delivery
          │                   │
          └─────────┬─────────┘
                    ▼
             Normalized result
                    │
             ┌──────┴──────┐
             ▼             ▼
           JSON         Console
```

The final solution should be **working, truthful, reproducible, resilient, and defensible during live code review**.
