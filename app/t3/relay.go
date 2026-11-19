package t3

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const RelayURL = "https://relay.t3.codes"

// Relay is T3 Connect's control plane: it lists your environments and asks
// one to mint a bootstrap credential for this key.
type Relay struct {
	clerk *Clerk
	key   *Key

	mu      sync.Mutex
	access  string
	expires time.Time
}

func NewRelay(c *Clerk, k *Key) *Relay { return &Relay{clerk: c, key: k} }

// Environment is one machine linked to the account.
type Environment struct {
	EnvironmentID string `json:"environmentId"`
	Label         string `json:"label"`
}

func (r *Relay) accessToken() (string, error) {
	r.mu.Lock()
	if r.access != "" && time.Until(r.expires) > time.Minute {
		defer r.mu.Unlock()
		return r.access, nil
	}
	r.mu.Unlock()
	subject, err := r.clerk.RelayToken()
	if err != nil {
		return "", err
	}
	u := RelayURL + "/v1/client/dpop-token"
	fields := url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":        {subject},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:jwt"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"resource":             {RelayURL},
		"scope":                {"environment:connect environment:status"},
		"client_id":            {"t3-web"},
	}
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	err = r.withClockRetry(func() (*http.Response, error) {
		return postForm(u, fields, http.Header{"Dpop": {r.key.Proof("POST", u, "")}})
	}, "Relay token exchange", &body)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	r.access, r.expires = body.AccessToken, time.Now().Add(time.Duration(body.ExpiresIn)*time.Second)
	r.mu.Unlock()
	return body.AccessToken, nil
}

// withClockRetry runs a DPoP request, and if the proof failed on time it
// adopts the server's clock and tries once more with a fresh proof.
func (r *Relay) withClockRetry(do func() (*http.Response, error), what string, out any) error {
	for attempt := 0; ; attempt++ {
		resp, err := do()
		if err != nil {
			return err
		}
		date := resp.Header.Get("Date")
		err = readJSON(resp, what, out)
		if e, ok := err.(*Error); ok && attempt == 0 && e.Body["dpopFailureReason"] == "time_window" && r.key.noteClock(date) {
			continue
		}
		return err
	}
}

func (r *Relay) dpopPost(path string, payload any, out any) error {
	u := RelayURL + path
	tok, err := r.accessToken()
	if err != nil {
		return err
	}
	return r.withClockRetry(func() (*http.Response, error) {
		var body []byte
		if payload != nil {
			body, _ = json.Marshal(payload)
		}
		req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "DPoP "+tok)
		req.Header.Set("Dpop", r.key.Proof("POST", u, tok))
		req.Header.Set("User-Agent", userAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return httpClient.Do(req)
	}, "Relay "+path, out)
}

func (r *Relay) ListEnvironments() ([]Environment, error) {
	tok, err := r.clerk.RelayToken()
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodGet, RelayURL+"/v1/environments", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	var body struct {
		Environments []Environment `json:"environments"`
	}
	return body.Environments, readJSON(resp, "List environments", &body)
}

// RelayStatus is how the relay sees an environment: "online" or "offline",
// and when offline, what it last saw fail (RelayEnvironmentStatusResponse).
type RelayStatus struct {
	Status          string `json:"status"`
	EndpointFailure string `json:"endpointFailure"` // "tunnel_down", "origin_unreachable", …
	OfflineReason   string `json:"offlineReason"`   // "tunnel_released"
}

func (r *Relay) Status(envID string) (RelayStatus, error) {
	var body RelayStatus
	err := r.dpopPost("/v1/environments/"+url.PathEscape(envID)+"/status", nil, &body)
	return body, err
}

type connectResult struct {
	Endpoint struct {
		HTTPBaseURL string `json:"httpBaseUrl"`
		WSBaseURL   string `json:"wsBaseUrl"`
	} `json:"endpoint"`
	Credential string `json:"credential"`
}

func (r *Relay) connect(envID string) (*connectResult, error) {
	var out connectResult
	err := r.dpopPost("/v1/environments/"+url.PathEscape(envID)+"/connect", map[string]string{"clientProofKeyThumbprint": r.key.Thumbprint}, &out)
	return &out, err
}
