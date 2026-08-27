package argos

import (
	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/http2"
	"github.com/bogdanfinn/tls-client/profiles"
	tls "github.com/bogdanfinn/utls"
)

// ML-DSA signature schemes advertised by Chrome 151+/152.
const (
	sigMLDSA44 = tls.SignatureScheme(0x0904)
	sigMLDSA65 = tls.SignatureScheme(0x0905)
	sigMLDSA87 = tls.SignatureScheme(0x0906)
)

const chrome152UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"
const chrome152SecCHUA = `"Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"`

// chrome152Profile matches real Chrome 152 JA4 (t13d1517h2_8daaf6152771_cb7bf5808d99),
// including ML-DSA sigs and trust_anchors that stock tls-client Chrome_146 lacks.
func chrome152Profile() profiles.ClientProfile {
	helloID := tls.ClientHelloID{
		Client:               "Chrome",
		RandomExtensionOrder: true,
		Version:              "152",
		SpecFactory: func() (tls.ClientHelloSpec, error) {
			return tls.ClientHelloSpec{
				CipherSuites: []uint16{
					tls.GREASE_PLACEHOLDER,
					tls.TLS_AES_128_GCM_SHA256,
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
				CompressionMethods: []byte{tls.CompressionNone},
				Extensions: []tls.TLSExtension{
					&tls.UtlsGREASEExtension{},
					&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
						{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
						{Group: tls.X25519MLKEM768},
						{Group: tls.X25519},
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
					&tls.SupportedVersionsExtension{Versions: []uint16{
						tls.GREASE_PLACEHOLDER, tls.VersionTLS13, tls.VersionTLS12,
					}},
					&tls.SupportedCurvesExtension{Curves: []tls.CurveID{
						tls.GREASE_PLACEHOLDER, tls.X25519MLKEM768, tls.X25519, tls.CurveP256, tls.CurveP384,
					}},
					&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
					&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
					&tls.UtlsCompressCertExtension{Algorithms: []tls.CertCompressionAlgo{tls.CertCompressionBrotli}},
					&tls.GenericExtension{Id: 0xca34, Data: []byte{0x00, 0x00}}, // trust_anchors
					&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
						tls.SignatureScheme(tls.GREASE_PLACEHOLDER),
						sigMLDSA44,
						sigMLDSA65,
						sigMLDSA87,
						tls.ECDSAWithP256AndSHA256,
						tls.PSSWithSHA256,
						tls.PKCS1WithSHA256,
						tls.ECDSAWithP384AndSHA384,
						tls.PSSWithSHA384,
						tls.PKCS1WithSHA384,
						tls.PSSWithSHA512,
						tls.PKCS1WithSHA512,
					}},
					&tls.UtlsGREASEExtension{},
				},
			}, nil
		},
	}

	settings := map[http2.SettingID]uint32{
		http2.SettingHeaderTableSize:   65536,
		http2.SettingEnablePush:        0,
		http2.SettingInitialWindowSize: 6291456,
		http2.SettingMaxHeaderListSize: 262144,
	}
	order := []http2.SettingID{
		http2.SettingHeaderTableSize,
		http2.SettingEnablePush,
		http2.SettingInitialWindowSize,
		http2.SettingMaxHeaderListSize,
	}
	pseudo := []string{":method", ":authority", ":scheme", ":path"}
	return profiles.NewClientProfile(helloID, settings, order, pseudo, 15663105, nil, nil, 0, false, nil, nil, 0, nil, false)
}

func navigateHeaders() http.Header {
	return http.Header{
		"sec-ch-ua":                 {chrome152SecCHUA},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {chrome152UA},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-US,en;q=0.9"},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"upgrade-insecure-requests", "user-agent", "accept",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
			"accept-encoding", "accept-language", "priority",
		},
		http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	}
}

func apiHeaders(referer string) http.Header {
	return http.Header{
		"accept":             {"application/json,*/*"},
		"accept-language":    {"en-US,en;q=0.9"},
		"content-type":       {"application/json"},
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
			"accept", "accept-language", "content-type", "referer",
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site",
			"user-agent", "x-newrelic-id", "accept-encoding", "cookie",
		},
		http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	}
}
