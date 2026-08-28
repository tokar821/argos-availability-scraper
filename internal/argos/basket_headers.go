package argos

import http "github.com/bogdanfinn/fhttp"

const basketPageURL = BaseURL + "/basket"

func basketAPIHeaders(referer string, extra map[string]string) http.Header {
	h := http.Header{
		"accept":             {"*/*"},
		"accept-language":    {"en-US,en;q=0.9"},
		"content-type":       {"application/json; charset=UTF-8"},
		"origin":             {BaseURL},
		"referer":            {referer},
		"sec-ch-ua":          {chrome152SecCHUA},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {`"Windows"`},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-origin"},
		"user-agent":         {chrome152UA},
		"x-newrelic-id":      {"VQEPU15SARAGV1hVDgMBUVY="},
		"accept-encoding":    {"gzip, deflate, br, zstd"},
		http.HeaderOrderKey: {
			"content-length", "sec-ch-ua", "accept", "sec-ch-ua-mobile", "content-type",
			"sec-ch-ua-platform", "user-agent", "origin", "sec-fetch-site", "sec-fetch-mode",
			"sec-fetch-dest", "referer", "accept-encoding", "accept-language", "x-newrelic-id", "cookie",
		},
		http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	}
	for k, v := range extra {
		h.Set(k, v)
	}
	return h
}
