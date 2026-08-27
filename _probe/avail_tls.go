package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/http2"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	tls "github.com/bogdanfinn/utls"
)

const (
	sigMLDSA44 = tls.SignatureScheme(0x0904)
	sigMLDSA65 = tls.SignatureScheme(0x0905)
	sigMLDSA87 = tls.SignatureScheme(0x0906)
)

func chrome152() profiles.ClientProfile {
	helloID := tls.ClientHelloID{
		Client: "Chrome", RandomExtensionOrder: true, Version: "152",
		SpecFactory: func() (tls.ClientHelloSpec, error) {
			return tls.ClientHelloSpec{
				CipherSuites: []uint16{
					tls.GREASE_PLACEHOLDER, tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384, tls.TLS_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA, tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
				CompressionMethods: []byte{tls.CompressionNone},
				Extensions: []tls.TLSExtension{
					&tls.UtlsGREASEExtension{},
					&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
						{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
						{Group: tls.X25519MLKEM768}, {Group: tls.X25519},
					}},
					&tls.PSKKeyExchangeModesExtension{Modes: []uint8{tls.PskModeDHE}},
					&tls.SCTExtension{},
					&tls.ApplicationSettingsExtensionNew{SupportedProtocols: []string{"h2"}},
					&tls.ExtendedMasterSecretExtension{},
					tls.BoringGREASEECH(),
					&tls.SupportedPointsExtension{SupportedPoints: []byte{tls.PointFormatUncompressed}},
					&tls.SessionTicketExtension{},
					&tls.StatusRequestExtension{},
					&tls.SNIExtension{},
					&tls.SupportedVersionsExtension{Versions: []uint16{tls.GREASE_PLACEHOLDER, tls.VersionTLS13, tls.VersionTLS12}},
					&tls.SupportedCurvesExtension{Curves: []tls.CurveID{tls.GREASE_PLACEHOLDER, tls.X25519MLKEM768, tls.X25519, tls.CurveP256, tls.CurveP384}},
					&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
					&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
					&tls.UtlsCompressCertExtension{Algorithms: []tls.CertCompressionAlgo{tls.CertCompressionBrotli}},
					&tls.GenericExtension{Id: 0xca34, Data: []byte{0x00, 0x00}},
					&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
						tls.SignatureScheme(tls.GREASE_PLACEHOLDER), sigMLDSA44, sigMLDSA65, sigMLDSA87,
						tls.ECDSAWithP256AndSHA256, tls.PSSWithSHA256, tls.PKCS1WithSHA256,
						tls.ECDSAWithP384AndSHA384, tls.PSSWithSHA384, tls.PKCS1WithSHA384,
						tls.PSSWithSHA512, tls.PKCS1WithSHA512,
					}},
					&tls.UtlsGREASEExtension{},
				},
			}, nil
		},
	}
	settings := map[http2.SettingID]uint32{
		http2.SettingHeaderTableSize: 65536, http2.SettingEnablePush: 0,
		http2.SettingInitialWindowSize: 6291456, http2.SettingMaxHeaderListSize: 262144,
	}
	order := []http2.SettingID{http2.SettingHeaderTableSize, http2.SettingEnablePush, http2.SettingInitialWindowSize, http2.SettingMaxHeaderListSize}
	return profiles.NewClientProfile(helloID, settings, order, []string{":method", ":authority", ":scheme", ":path"}, 15663105, nil, nil, 0, false, nil, nil, 0, nil, false)
}

func main() {
	opts := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(chrome152()),
		tls_client.WithTimeoutSeconds(45),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithDisableHttp3(),
		tls_client.WithRandomTLSExtensionOrder(),
	}
	if p := os.Getenv("HTTP_PROXY"); p != "" {
		opts = append(opts, tls_client.WithProxyUrl(p))
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		panic(err)
	}
	productID := "7885338"
	pageURL := "https://www.argos.co.uk/product/" + productID
	req, _ := http.NewRequest("GET", pageURL, nil)
	nav(req)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println("product", resp.StatusCode, len(body), "denied", strings.Contains(string(body), "Access Denied"))

	q := url.Values{}
	q.Set("origin", "London")
	q.Set("skuQty", productID+"_1")
	q.Set("maxResults", "10")
	q.Set("maxDistance", "50")
	q.Set("ssm", "true")
	apiURL := "https://www.argos.co.uk/stores/api/orchestrator/v0/locator/availability?" + q.Encode()
	req2, _ := http.NewRequest("GET", apiURL, nil)
	api(req2, pageURL)
	resp2, err := client.Do(req2)
	if err != nil {
		panic(err)
	}
	b2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Println("collection", resp2.StatusCode, len(b2))
	fmt.Println(snip(string(b2), 250))

	q2 := url.Values{}
	q2.Set("postcode", "M1 1AE")
	q2.Set("skuQty", productID+"_1")
	q2.Set("maxResults", "10")
	q2.Set("maxDistance", "50")
	apiURL2 := "https://www.argos.co.uk/stores/api/orchestrator/v0/locator/availability?" + q2.Encode()
	req3, _ := http.NewRequest("GET", apiURL2, nil)
	api(req3, pageURL)
	resp3, err := client.Do(req3)
	if err != nil {
		panic(err)
	}
	b3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	fmt.Println("delivery", resp3.StatusCode, len(b3))
	fmt.Println(snip(string(b3), 250))

	var probe map[string]any
	_ = json.Unmarshal(b2, &probe)
	if _, ok := probe["stores"]; ok {
		fmt.Println("collection JSON has stores")
	}
}

func snip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func nav(req *http.Request) {
	req.Header = http.Header{
		"sec-ch-ua": {`"Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"`},
		"sec-ch-ua-mobile": {"?0"}, "sec-ch-ua-platform": {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"},
		"accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site": {"none"}, "sec-fetch-mode": {"navigate"}, "sec-fetch-user": {"?1"}, "sec-fetch-dest": {"document"},
		"accept-encoding": {"gzip, deflate, br, zstd"}, "accept-language": {"en-US,en;q=0.9"}, "priority": {"u=0, i"},
		http.HeaderOrderKey: {"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "upgrade-insecure-requests", "user-agent", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest", "accept-encoding", "accept-language", "priority"},
		http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	}
}

func api(req *http.Request, referer string) {
	req.Header = http.Header{
		"accept": {"application/json,*/*"}, "accept-language": {"en-US,en;q=0.9"}, "content-type": {"application/json"},
		"referer": {referer},
		"sec-ch-ua": {`"Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"`},
		"sec-ch-ua-mobile": {"?0"}, "sec-ch-ua-platform": {`"Windows"`},
		"sec-fetch-dest": {"empty"}, "sec-fetch-mode": {"cors"}, "sec-fetch-site": {"same-origin"},
		"user-agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"},
		"x-newrelic-id": {"VQEPU15SARAGV1hVDgMBUVY="}, "accept-encoding": {"gzip, deflate, br, zstd"},
		http.HeaderOrderKey: {"accept", "accept-language", "content-type", "referer", "sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site", "user-agent", "x-newrelic-id", "accept-encoding", "cookie"},
		http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	}
}
