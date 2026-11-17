package t3

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	clerkAPI      = "https://clerk.t3.codes/v1"
	clerkTemplate = "t3-relay"
)

// Clerk is T3 Connect's sign-in, through Clerk's native Frontend API. The
// client JWT it rotates is the long-lived credential, so every new value is
// saved before anything else happens.
type Clerk struct {
	store *Store
	mu    sync.Mutex
	jwt   string // client JWT
	token string // relay-audience session JWT, about a minute
}

// SignIn is a sign-in in progress.
type SignIn struct {
	ID             string
	EmailAddressID string
	HasPassword    bool
}

func NewClerk(s *Store) *Clerk {
	c := &Clerk{store: s}
	_, _ = s.Get("clerk.client", &c.jwt)
	return c
}

func (c *Clerk) HasClient() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.jwt != ""
}

type clerkResp struct {
	Response json.RawMessage `json:"response"`
	JWT      string          `json:"jwt"`
}

func (c *Clerk) fapi(path string, fields url.Values) (*clerkResp, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	u := clerkAPI + path + sep + "_is_native=1"
	method := http.MethodGet
	var body *strings.Reader
	if fields != nil {
		method = http.MethodPost
		body = strings.NewReader(fields.Encode())
	} else {
		body = strings.NewReader("")
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.jwt != "" {
		req.Header.Set("Authorization", c.jwt)
	}
	c.mu.Unlock()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if rotated := resp.Header.Get("Authorization"); rotated != "" {
		c.mu.Lock()
		c.jwt = rotated
		c.mu.Unlock()
		if err := c.store.Put("clerk.client", rotated); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}
	var out clerkResp
	if err := readJSON(resp, "Clerk "+strings.SplitN(path, "?", 2)[0], &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Clerk) StartSignIn(email string) (*SignIn, error) {
	r, err := c.fapi("/client/sign_ins", url.Values{"identifier": {email}})
	if err != nil {
		return nil, err
	}
	var si struct {
		ID      string `json:"id"`
		Factors []struct {
			Strategy       string `json:"strategy"`
			EmailAddressID string `json:"email_address_id"`
		} `json:"supported_first_factors"`
	}
	if err := json.Unmarshal(r.Response, &si); err != nil {
		return nil, err
	}
	out := &SignIn{ID: si.ID}
	for _, f := range si.Factors {
		switch f.Strategy {
		case "email_code":
			out.EmailAddressID = f.EmailAddressID
		case "password":
			out.HasPassword = true
		}
	}
	return out, nil
}

func (c *Clerk) SendEmailCode(si *SignIn) error {
	if si.EmailAddressID == "" {
		return errors.New("this account has no email-code sign-in")
	}
	_, err := c.fapi("/client/sign_ins/"+si.ID+"/prepare_first_factor", url.Values{
		"strategy":         {"email_code"},
		"email_address_id": {si.EmailAddressID},
	})
	return err
}

// FinishWithCode completes the sign-in with the emailed code.
func (c *Clerk) FinishWithCode(si *SignIn, code string) error {
	return c.finish(si, url.Values{"strategy": {"email_code"}, "code": {code}})
}

func (c *Clerk) FinishWithPassword(si *SignIn, password string) error {
	return c.finish(si, url.Values{"strategy": {"password"}, "password": {password}})
}

func (c *Clerk) finish(si *SignIn, fields url.Values) error {
	r, err := c.fapi("/client/sign_ins/"+si.ID+"/attempt_first_factor", fields)
	if err != nil {
		return err
	}
	var st struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(r.Response, &st)
	if st.Status != "complete" {
		return errors.New("sign-in needs another step (" + st.Status + "), which tern doesn't support yet")
	}
	return nil
}

func (c *Clerk) sessionID() (string, error) {
	r, err := c.fapi("/client", nil)
	if err != nil {
		return "", err
	}
	var cl struct {
		LastActive string `json:"last_active_session_id"`
		Sessions   []struct {
			ID string `json:"id"`
		} `json:"sessions"`
	}
	_ = json.Unmarshal(r.Response, &cl)
	if cl.LastActive != "" {
		return cl.LastActive, nil
	}
	if len(cl.Sessions) > 0 {
		return cl.Sessions[0].ID, nil
	}
	return "", ErrSignedOut
}

// ErrSignedOut means there's no Clerk session: show the sign-in screen.
var ErrSignedOut = errors.New("not signed in to T3 Connect")

// RelayToken is a Clerk session JWT with the relay audience.
func (c *Clerk) RelayToken() (string, error) {
	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	if tok != "" && time.Until(jwtExp(tok)) > 10*time.Second {
		return tok, nil
	}
	sid, err := c.sessionID()
	if err != nil {
		return "", err
	}
	r, err := c.fapi("/client/sessions/"+sid+"/tokens/"+clerkTemplate, url.Values{})
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.token = r.JWT
	c.mu.Unlock()
	return r.JWT, nil
}

func (c *Clerk) SignOut() {
	c.mu.Lock()
	c.jwt, c.token = "", ""
	c.mu.Unlock()
	_ = c.store.Delete("clerk.client")
}

func jwtExp(jwt string) time.Time {
	parts := strings.Split(jwt, ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	raw, err := b64.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	_ = json.Unmarshal(raw, &claims)
	return time.Unix(claims.Exp, 0)
}
