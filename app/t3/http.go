package t3

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const userAgent = "tern/0.1"

// Error is a failed T3, relay or Clerk call, with the decoded body.
type Error struct {
	What   string
	Status int
	Reason string
	Body   map[string]any
}

func (e *Error) Error() string {
	if e.Status == 0 {
		return e.What
	}
	return strings.TrimSpace(fmt.Sprintf("%s failed (%d) %s", e.What, e.Status, e.Reason))
}

// readJSON decodes a response into out, or turns a non-2xx into an *Error
// with the most useful reason the body has.
func readJSON(resp *http.Response, what string, out any) error {
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body := map[string]any{}
		_ = json.Unmarshal(raw, &body)
		return &Error{What: what, Status: resp.StatusCode, Reason: reasonOf(body, raw), Body: body}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}

func reasonOf(body map[string]any, raw []byte) string {
	if errs, ok := body["errors"].([]any); ok && len(errs) > 0 {
		if e, ok := errs[0].(map[string]any); ok {
			for _, k := range []string{"long_message", "message"} {
				if s, ok := e[k].(string); ok && s != "" {
					return s
				}
			}
		}
	}
	var parts []string
	for _, k := range []string{"_tag", "reason", "dpopFailureReason", "message"} {
		if s, ok := body[k].(string); ok && s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 && len(raw) < 200 {
		return strings.TrimSpace(string(raw))
	}
	return strings.Join(parts, " ")
}

func postForm(rawURL string, fields url.Values, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(fields.Encode()))
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	return httpClient.Do(req)
}
