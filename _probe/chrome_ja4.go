package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("enable-automation", false),
	)
	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel2 := chromedp.NewContext(alloc)
	defer cancel2()
	ctx, cancel3 := context.WithTimeout(ctx, 40*time.Second)
	defer cancel3()

	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://tls.peet.ws/api/all"),
		chromedp.Sleep(2*time.Second),
		chromedp.InnerHTML("body", &body),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(body)
	if i := strings.Index(body, "ja4"); i >= 0 {
		fmt.Println("around ja4:", body[i:min(i+120, len(body))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
