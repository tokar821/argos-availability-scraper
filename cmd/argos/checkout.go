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
)

func runCheckout(args []string) int {
	loadDotEnv(".env")

	fs := flag.NewFlagSet("checkout", flag.ExitOnError)
	product := fs.String("product", "", "Argos product ID or URL")
	postcode := fs.String("postcode", "", "UK postcode for delivery checkout")
	quiet := fs.Bool("quiet", false, "suppress stderr summary")
	timeout := fs.Duration("timeout", 120*time.Second, "overall timeout (includes 30s adaptive wait if challenged)")
	proxy := fs.String("proxy", "", "optional HTTP(S) proxy URL")
	_ = fs.Parse(args)

	if *product == "" && fs.NArg() > 0 {
		*product = fs.Arg(0)
	}
	if *postcode == "" && fs.NArg() > 1 {
		*postcode = fs.Arg(1)
	}

	*product = strings.TrimSpace(*product)
	*postcode = strings.TrimSpace(*postcode)
	if *product == "" || *postcode == "" {
		fmt.Fprintln(os.Stderr, "usage: argos checkout --product <id|url> --postcode <UK postcode>")
		fmt.Fprintln(os.Stderr, "   or: argos checkout <product> <postcode>")
		return 2
	}

	productID, err := argos.ResolveProductID(*product)
	if err != nil {
		emitCheckout(model.CheckoutResult{
			ProductID: *product, Postcode: *postcode, Fulfilment: "delivery",
			CheckedAt: time.Now().UTC(),
			Error:     &model.ErrorInfo{Code: "invalid_input", Message: err.Error()},
		}, !*quiet)
		return 2
	}

	client := argos.NewClient()
	client.Timeout = *timeout
	client.ProxyURL = strings.TrimSpace(*proxy)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	checkedAt := time.Now().UTC()
	resp, err := client.CheckoutDelivery(ctx, productID, *postcode)
	res := model.CheckoutResult{
		ProductID:  productID,
		Postcode:   *postcode,
		Fulfilment: "delivery",
		CheckedAt:  checkedAt,
	}
	if err != nil {
		res.Error = &model.ErrorInfo{Code: errorCode(err), Message: err.Error()}
		emitCheckout(res, !*quiet)
		return exitCode(err)
	}
	res.SnapshotID = resp.SnapshotID
	res.RedirectTo = resp.RedirectTo
	emitCheckout(res, !*quiet)
	return 0
}

func emitCheckout(res model.CheckoutResult, summary bool) {
	if summary {
		fmt.Fprintln(os.Stderr, "────────────────────────────────────────")
		fmt.Fprintf(os.Stderr, "Checkout: product %s → %s (%s)\n", res.ProductID, res.Postcode, res.Fulfilment)
		if res.Error != nil {
			fmt.Fprintf(os.Stderr, "ERROR: [%s] %s\n", res.Error.Code, res.Error.Message)
		} else {
			fmt.Fprintf(os.Stderr, "Snapshot: %s\n", res.SnapshotID)
			fmt.Fprintf(os.Stderr, "Redirect: %s\n", res.RedirectTo)
		}
		fmt.Fprintln(os.Stderr, "────────────────────────────────────────")
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res)
}
