24-HOUR DEVELOPMENT TRIAL
Argos Collection & Delivery Scraper
Prepared for Tokar  •  Adventure AIO
WEBSITE
argos.co.uk
LANGUAGE
Go
TRIAL WINDOW
Aug 26, 1:00 PM → Aug 27, 1:00 PM CST


Objective  Build a working Go program that retrieves live Argos product availability for both Collection and Delivery using a UK postcode or town.

REQUIRED BEHAVIOR
Accept a product URL or Argos product ID, a UK postcode/town, and mode: collection, delivery, or both.
Fetch live results at runtime—no hard-coded availability and no manually prepared dataset.
Return product title, product ID, price, requested location, timestamp, and availability status for each mode.
When shown by Argos, include the earliest delivery/collection date or window, delivery fee, and nearby store name/distance.
Produce structured JSON and a readable console summary; handle invalid products, unavailable stock, timeouts, and blocking cleanly.
SUBMISSION
A Git repository or ZIP containing all Go source code, go.mod/go.sum, and clear build/run instructions.
A README describing the approach, assumptions, dependencies, known limitations, and any external services used.
Automated tests for core parsing/normalization logic, plus sample JSON output from a real run.
A live code review and functional review will take place tomorrow. Be prepared to explain the architecture and tradeoffs, answer code questions, and run two reviewer-supplied Argos product pages with a postcode/town.
ACCEPTANCE & REVIEW
Pass condition: the reviewer can build and run the project from the README, query both modes against live Argos pages, and see truthful results—or a specific, structured failure—instead of stale or fabricated data.
Scoring: functional correctness 40%  •  resilience and error handling 25%  •  idiomatic Go/design 20%  •  tests and documentation 10%  •  explanation and tradeoffs 5%.
Constraints: open-source Go libraries and browser automation are allowed. Disclose all external services; do not rely on paid extraction APIs, bypass CAPTCHA/authentication, or circumvent access controls. The supplied screenshots are visual references only.

