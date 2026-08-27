package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokar821/argos-availability-scraper/internal/argos"
	"github.com/tokar821/argos-availability-scraper/internal/model"
	"github.com/tokar821/argos-availability-scraper/internal/normalize"
)

func main() {
	os.Exit(run())
}

func run() int {
	loadDotEnv(".env")

	var (
		product  = flag.String("product", "", "Argos product ID or product URL")
		location = flag.String("location", "", "UK postcode or town")
		mode     = flag.String("mode", "both", "collection | delivery | both")
		jsonOut  = flag.Bool("json", true, "print normalized JSON (default true)")
		quiet    = flag.Bool("quiet", false, "suppress human-readable console summary")
		timeout  = flag.Duration("timeout", 75*time.Second, "per-request timeout")
		proxy    = flag.String("proxy", "", "optional HTTP(S) proxy URL (else HTTP_PROXY / HTTPS_PROXY from env/.env)")
	)
	flag.Parse()

	args := flag.Args()
	if *product == "" && len(args) > 0 {
		*product = args[0]
	}
	if *location == "" && len(args) > 1 {
		*location = args[1]
	}
	if len(args) > 2 && flag.Lookup("mode").Value.String() == "both" {
		*mode = args[2]
	}

	*mode = strings.ToLower(strings.TrimSpace(*mode))
	*product = strings.TrimSpace(*product)
	*location = strings.TrimSpace(*location)

	if *product == "" || *location == "" {
		fmt.Fprintln(os.Stderr, "usage: argos --product <id|url> --location <postcode|town> [--mode collection|delivery|both]")
		fmt.Fprintln(os.Stderr, "   or: argos <product> <location> [mode]")
		return 2
	}
	switch *mode {
	case "collection", "delivery", "both":
	default:
		fmt.Fprintf(os.Stderr, "invalid mode %q (expected collection, delivery, or both)\n", *mode)
		return 2
	}

	productID, err := argos.ResolveProductID(*product)
	if err != nil {
		writeFailure(*product, *location, *mode, err)
		return 2
	}

	client := argos.NewClient()
	client.Timeout = *timeout
	client.ProxyURL = strings.TrimSpace(*proxy)
	defer client.Close()

	runBudget := *timeout
	if *mode == "both" {
		runBudget = *timeout * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), runBudget)
	defer cancel()

	checkedAt := time.Now().UTC()

	productInfo, err := client.FetchProduct(ctx, productID)
	if err != nil {
		res := model.Result{
			ProductID:  productID,
			Location:   *location,
			Mode:       *mode,
			CheckedAt:  checkedAt,
			ProductURL: argos.ProductURL(productID),
			Error:      &model.ErrorInfo{Code: errorCode(err), Message: err.Error()},
		}
		emit(res, *jsonOut, !*quiet)
		return exitCode(err)
	}

	var (
		collectionResp, deliveryResp *argos.AvailabilityResponse
		collectionErr, deliveryErr   error
	)

	if *mode == "collection" || *mode == "both" {
		collectionResp, collectionErr = client.FetchCollection(ctx, productID, *location)
	}
	if *mode == "delivery" || *mode == "both" {
		deliveryResp, deliveryErr = client.FetchDelivery(ctx, productID, *location)
	}

	res := normalize.BuildResult(productInfo, *location, *mode, checkedAt, collectionResp, deliveryResp, collectionErr, deliveryErr)
	emit(res, *jsonOut, !*quiet)

	if res.Error != nil {
		return exitCode(fmt.Errorf("%s", res.Error.Message))
	}
	if (*mode == "collection" || *mode == "both") && collectionErr != nil {
		return exitCode(collectionErr)
	}
	if (*mode == "delivery" || *mode == "both") && deliveryErr != nil {
		return exitCode(deliveryErr)
	}
	return 0
}

func emit(res model.Result, asJSON, summary bool) {
	if summary {
		printSummary(res)
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	}
}

func printSummary(res model.Result) {
	fmt.Fprintln(os.Stderr, "────────────────────────────────────────")
	fmt.Fprintf(os.Stderr, "Product : %s (%s)\n", res.Title, res.ProductID)
	if res.Price != nil {
		fmt.Fprintf(os.Stderr, "Price   : %s\n", res.Price.Display)
	}
	fmt.Fprintf(os.Stderr, "Location: %s\n", res.Location)
	fmt.Fprintf(os.Stderr, "Mode    : %s\n", res.Mode)
	fmt.Fprintf(os.Stderr, "Checked : %s\n", res.CheckedAt.Format(time.RFC3339))
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "ERROR   : [%s] %s\n", res.Error.Code, res.Error.Message)
	}
	if res.Collection != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "COLLECTION: %s\n", strings.ToUpper(res.Collection.Status))
		if res.Collection.Message != "" {
			fmt.Fprintf(os.Stderr, "  %s\n", res.Collection.Message)
		}
		if res.Collection.EarliestDate != "" {
			fmt.Fprintf(os.Stderr, "  Earliest: %s\n", res.Collection.EarliestDate)
		}
		for i, s := range res.Collection.Stores {
			if i >= 5 {
				fmt.Fprintf(os.Stderr, "  … %d more stores\n", len(res.Collection.Stores)-5)
				break
			}
			dist := ""
			if s.DistanceMiles != nil {
				dist = fmt.Sprintf(" (%.2f miles)", *s.DistanceMiles)
			}
			fmt.Fprintf(os.Stderr, "  - %s%s — %s\n", s.Name, dist, s.Message)
		}
		if res.Collection.Error != nil {
			fmt.Fprintf(os.Stderr, "  error: [%s] %s\n", res.Collection.Error.Code, res.Collection.Error.Message)
		}
	}
	if res.Delivery != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "DELIVERY  : %s\n", strings.ToUpper(res.Delivery.Status))
		if res.Delivery.Message != "" {
			fmt.Fprintf(os.Stderr, "  %s\n", res.Delivery.Message)
		}
		if res.Delivery.EarliestDate != "" {
			fmt.Fprintf(os.Stderr, "  Earliest: %s\n", res.Delivery.EarliestDate)
		}
		if res.Delivery.Fee != nil {
			fmt.Fprintf(os.Stderr, "  Fee     : %s\n", res.Delivery.Fee.Display)
		}
		if res.Delivery.Error != nil {
			fmt.Fprintf(os.Stderr, "  error: [%s] %s\n", res.Delivery.Error.Code, res.Delivery.Error.Message)
		}
	}
	fmt.Fprintln(os.Stderr, "────────────────────────────────────────")
}

func writeFailure(product, location, mode string, err error) {
	res := model.Result{
		ProductID: product,
		Location:  location,
		Mode:      mode,
		CheckedAt: time.Now().UTC(),
		Error:     &model.ErrorInfo{Code: errorCode(err), Message: err.Error()},
	}
	emit(res, true, true)
}

func errorCode(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "blocked"), strings.Contains(msg, "access denied"):
		return "blocked"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
		return "timeout"
	case strings.Contains(msg, "not found"):
		return "not_found"
	case strings.Contains(msg, "could not parse"), strings.Contains(msg, "invalid"):
		return "invalid_input"
	default:
		return "request_error"
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	switch errorCode(err) {
	case "invalid_input":
		return 2
	case "not_found":
		return 3
	case "blocked":
		return 4
	case "timeout":
		return 5
	default:
		return 1
	}
}
