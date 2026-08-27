package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	proxy := os.Getenv("HTTP_PROXY")
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("proxy-server", strings.TrimPrefix(strings.TrimPrefix(proxy, "http://"), "https://")),
		chromedp.Flag("ignore-certificate-errors", true),
	)
	// proxy with auth needs extension or chrome doesn't support user:pass in --proxy-server well
	fmt.Println("NOTE: Chrome --proxy-server may not pass user:pass; testing proxy host only may fail auth")
	fmt.Println("proxy", proxy)

	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel2 := chromedp.NewContext(alloc)
	defer cancel2()
	ctx, cancel3 := context.WithTimeout(ctx, 45*time.Second)
	defer cancel3()

	var title, html string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.argos.co.uk/product/7885338"),
		chromedp.Sleep(3*time.Second),
		chromedp.Title(&title),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		fmt.Println("chrome err", err)
		return
	}
	fmt.Println("title", title, "len", len(html), "denied", strings.Contains(html, "Access Denied"))
}
