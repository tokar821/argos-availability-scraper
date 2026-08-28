package argos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	hyper "github.com/Hyper-Solutions/hyper-sdk-go/v2"
	"github.com/Hyper-Solutions/hyper-sdk-go/v2/akamai"
	http "github.com/bogdanfinn/fhttp"
)

type adaptiveChallenge struct {
	Provider        string `json:"provider"`
	ChlgDuration    int    `json:"chlg_duration"`
	BrandingCustURL string `json:"branding_cust_url"`
	Token           string `json:"token"`
	Timestamp       int    `json:"timestamp"`
	Nonce           string `json:"nonce"`
	Difficulty      int    `json:"difficulty"`
	Count           int    `json:"count"`
	Timeout         int    `json:"timeout"`
}

func (c *Client) solveAdaptiveChallenge(ctx context.Context, hyperKey, challengeJSON string) error {
	var ch adaptiveChallenge
	if err := json.Unmarshal([]byte(challengeJSON), &ch); err != nil {
		return fmt.Errorf("parse 428 payload: %w", err)
	}
	if ch.Count == 0 {
		ch.Count = 1
	}
	dur := ch.ChlgDuration
	if dur <= 0 {
		dur = 30
	}

	brandingURL := ch.BrandingCustURL
	if brandingURL == "" {
		brandingURL = BaseURL + "/challenge/adaptive.html"
	} else if strings.HasPrefix(brandingURL, "/") {
		brandingURL = BaseURL + brandingURL
	}

	if _, _, err := c.fetchGET(ctx, brandingURL, navigateHeaders()); err != nil {
		return fmt.Errorf("fetch branding page: %w", err)
	}
	secCpt := c.getCookie("sec_cpt")
	if secCpt == "" {
		return fmt.Errorf("sec_cpt cookie missing after branding fetch")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(dur) * time.Second):
	}

	powPayload, err := buildAdaptivePow(secCpt, &ch)
	if err != nil {
		return err
	}
	powHdr := basketAPIHeaders(brandingURL, nil)
	powHdr.Set("content-type", "application/json")
	if _, status, err := c.fetchPOST(ctx, BaseURL+"/_sec/verify?provider=adaptive", powPayload, powHdr); err != nil {
		return fmt.Errorf("adaptive pow post: %w", err)
	} else if status >= 400 {
		return fmt.Errorf("adaptive pow HTTP %d", status)
	}

	api := hyper.NewSession(hyperKey)
	ip, err := hyperOutboundIP(ctx, c, hyperKey)
	if err != nil {
		return fmt.Errorf("hyper ip lookup: %w", err)
	}

	brHTML, _, err := c.fetchGET(ctx, brandingURL, navigateHeaders())
	if err != nil {
		return fmt.Errorf("re-fetch branding: %w", err)
	}
	scriptPath, err := akamai.ParseScriptPath(strings.NewReader(brHTML))
	if err != nil {
		return fmt.Errorf("parse sensor script: %w", err)
	}
	scriptURL := BaseURL + scriptPath
	scriptBody, _, err := c.fetchGET(ctx, scriptURL, scriptHeaders(basketPageURL))
	if err != nil {
		return fmt.Errorf("fetch sensor script: %w", err)
	}

	var sensorCtx string
	for i := 0; i < 3; i++ {
		in := &hyper.SensorInput{
			Abck:           c.getCookie("_abck"),
			Bmsz:           c.getCookie("bm_sz"),
			Version:        "3",
			PageUrl:        basketPageURL,
			UserAgent:      chrome152UA,
			ScriptUrl:      scriptURL,
			AcceptLanguage: "en-US,en;q=0.9",
			IP:             ip,
			Context:        sensorCtx,
		}
		if i == 0 {
			in.Script = scriptBody
		}
		sensorData, next, err := api.GenerateSensorData(ctx, in)
		if err != nil {
			return fmt.Errorf("hyper sensor: %w", err)
		}
		sensorCtx = next
		body, _ := json.Marshal(map[string]string{"sensor_data": sensorData})
		postHdr := sensorPostHeaders(basketPageURL)
		if _, _, err := c.fetchPOST(ctx, scriptURL, body, postHdr); err != nil {
			return fmt.Errorf("sensor post: %w", err)
		}
		if strings.Contains(c.getCookie("sec_cpt"), "~3~") {
			break
		}
	}

	if _, _, err := c.fetchGET(ctx, BaseURL+"/_sec/cp_challenge/verify", navigateHeaders()); err != nil {
		return fmt.Errorf("challenge verify: %w", err)
	}
	if !strings.Contains(c.getCookie("sec_cpt"), "~3~") {
		return fmt.Errorf("sec_cpt not validated after adaptive solve")
	}
	return nil
}

func buildAdaptivePow(secCpt string, ch *adaptiveChallenge) ([]byte, error) {
	count := ch.Count
	if count == 0 {
		count = 1
	}
	data, err := json.Marshal(map[string]any{
		"token":      ch.Token,
		"timestamp":  ch.Timestamp,
		"nonce":      ch.Nonce,
		"difficulty": ch.Difficulty,
		"count":      count,
		"timeout":    ch.Timeout,
		"cpu":        false,
	})
	if err != nil {
		return nil, err
	}
	path := "/challenge/adaptive.html"
	if ch.BrandingCustURL != "" {
		if strings.HasPrefix(ch.BrandingCustURL, "http") {
			path = strings.TrimPrefix(ch.BrandingCustURL, BaseURL)
		} else {
			path = ch.BrandingCustURL
		}
	}
	dur := ch.ChlgDuration
	if dur <= 0 {
		dur = 30
	}
	html := fmt.Sprintf(`<iframe challenge="%s" data-duration=%d src="%s"></iframe>`,
		base64.StdEncoding.EncodeToString(data), dur, path)
	parsed, err := akamai.ParseSecCptChallenge(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	return parsed.GenerateSecCptPayload(secCpt)
}

func hyperOutboundIP(ctx context.Context, c *Client, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://ip.hypersolutions.co/ip", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("user-agent", chrome152UA)
	resp, err := c.do(ctx, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if out.IP == "" {
		return "", fmt.Errorf("empty ip in hyper response")
	}
	return out.IP, nil
}

func scriptHeaders(referer string) http.Header {
	h := navigateHeaders()
	h.Set("referer", referer)
	h.Set("sec-fetch-site", "same-origin")
	h.Set("sec-fetch-mode", "no-cors")
	h.Set("sec-fetch-dest", "script")
	return h
}

func sensorPostHeaders(referer string) http.Header {
	h := basketAPIHeaders(referer, nil)
	h.Set("content-type", "text/plain;charset=UTF-8")
	return h
}
