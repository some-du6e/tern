package t3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const protocol = "2"

// Session is a login on one environment. Connect sessions are DPoP-bound
// and last an hour; paired ones are bearer tokens that last 30 days.
type Session struct {
	EnvironmentID string    `json:"environmentId"`
	Label         string    `json:"label"`
	HTTPBaseURL   string    `json:"httpBaseUrl"`
	WSBaseURL     string    `json:"wsBaseUrl"`
	AccessToken   string    `json:"accessToken"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Bearer        bool      `json:"bearer"` // paired: no DPoP
}

func sessionName(envID string) string { return "env." + envID + ".json" }

// CachedSession is a stored session with at least two minutes left.
func CachedSession(s *Store, envID string) *Session {
	var sess Session
	if ok, err := s.Get(sessionName(envID), &sess); !ok || err != nil {
		return nil
	}
	if time.Until(sess.ExpiresAt) < 2*time.Minute {
		return nil
	}
	return &sess
}

func ForgetSession(s *Store, envID string) { _ = s.Delete(sessionName(envID)) }

// OpenSession asks the relay to connect, then trades the bootstrap
// credential with the environment for a DPoP session.
func OpenSession(r *Relay, s *Store, env Environment) (*Session, error) {
	c, err := r.connect(env.EnvironmentID)
	if err != nil {
		return nil, err
	}
	u := strings.TrimRight(c.Endpoint.HTTPBaseURL, "/") + "/oauth/token"
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	err = r.withClockRetry(func() (*http.Response, error) {
		return postForm(u, url.Values{
			"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":        {c.Credential},
			"subject_token_type":   {"urn:t3:params:oauth:token-type:environment-bootstrap"},
			"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
			"client_label":         {"tern"},
			"client_device_type":   {"desktop"},
		}, http.Header{"Dpop": {r.key.Proof("POST", u, "")}})
	}, "Environment token exchange", &body)
	if err != nil {
		return nil, err
	}
	sess := &Session{
		EnvironmentID: env.EnvironmentID,
		Label:         env.Label,
		HTTPBaseURL:   c.Endpoint.HTTPBaseURL,
		WSBaseURL:     c.Endpoint.WSBaseURL,
		AccessToken:   body.AccessToken,
		ExpiresAt:     time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
	}
	return sess, s.Put(sessionName(env.EnvironmentID), sess)
}

// ErrPairingLink is a pairing URL without a host or a #token=.
var ErrPairingLink = errors.New("the pairing link is missing its backend host or token")

// Pair trades a pairing URL from `t3 pair` (https://host/pair#token=…) for
// a 30-day bearer session. No Clerk, relay or DPoP involved. Like T3's
// registerPairing, it reads the server's descriptor and checks that the
// two speak the same protocol first, so a wrong or incompatible host never
// spends the one-time token.
func Pair(s *Store, pairURL string) (*Session, error) {
	u, err := url.Parse(strings.TrimSpace(pairURL))
	frag := url.Values{}
	if err == nil {
		frag, _ = url.ParseQuery(u.Fragment)
	}
	token := frag.Get("token")
	if err != nil || u.Host == "" || token == "" {
		return nil, ErrPairingLink
	}
	base := u.Scheme + "://" + u.Host
	var desc struct {
		EnvironmentID string `json:"environmentId"`
		Label         string `json:"label"`
		Protocol      *int   `json:"orchestrationProtocolVersion"`
	}
	endpoint := base + "/.well-known/t3/environment"
	resp, err := httpClient.Get(endpoint)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("failed to fetch remote environment endpoint %s (%v)", endpoint, err)
	}
	if err := readJSON(resp, "Environment descriptor", &desc); err != nil {
		return nil, err
	}
	// Servers from before the negotiation speak the first protocol
	// (orchestrationProtocolCompatibilityError).
	server := 1
	if desc.Protocol != nil {
		server = *desc.Protocol
	}
	switch client, _ := strconv.Atoi(protocol); {
	case server > client:
		return nil, fmt.Errorf("this client is not supported by this server. Update your app or use a compatible release to connect to %s", desc.Label)
	case server < client:
		return nil, fmt.Errorf("this client requires a newer server. Update T3 Code on %s to connect", desc.Label)
	}
	resp, err = postForm(base+"/oauth/token", url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":        {token},
		"subject_token_type":   {"urn:t3:params:oauth:token-type:environment-bootstrap"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"client_label":         {"tern"},
		"client_device_type":   {"desktop"},
	}, nil)
	if err != nil {
		return nil, err
	}
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := readJSON(resp, "Pairing", &body); err != nil {
		return nil, err
	}
	ws := "ws" + strings.TrimPrefix(base, "http")
	sess := &Session{
		EnvironmentID: desc.EnvironmentID,
		Label:         desc.Label,
		HTTPBaseURL:   base,
		WSBaseURL:     ws,
		AccessToken:   body.AccessToken,
		ExpiresAt:     time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
		Bearer:        true,
	}
	if err := s.Put(sessionName(desc.EnvironmentID), sess); err != nil {
		return nil, err
	}
	return sess, AddPaired(s, Environment{EnvironmentID: desc.EnvironmentID, Label: desc.Label})
}

// Paired lists environments added by pairing URL rather than Connect.
func Paired(s *Store) []Environment {
	var envs []Environment
	_, _ = s.Get("paired.json", &envs)
	return envs
}

func AddPaired(s *Store, env Environment) error {
	envs := Paired(s)
	for _, e := range envs {
		if e.EnvironmentID == env.EnvironmentID {
			return nil
		}
	}
	return s.Put("paired.json", append(envs, env))
}

// authorize signs a request to the environment with the session.
func (s *Session) authorize(req *http.Request, k *Key) {
	if s.Bearer {
		req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	} else {
		req.Header.Set("Authorization", "DPoP "+s.AccessToken)
		req.Header.Set("Dpop", k.Proof(req.Method, req.URL.String(), s.AccessToken))
	}
	req.Header.Set("X-T3-Orchestration-Protocol", protocol)
	req.Header.Set("User-Agent", userAgent)
}

func (s *Session) do(k *Key, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	u, err := url.JoinPath(s.HTTPBaseURL, path)
	if err != nil {
		return nil, err
	}
	if i := strings.Index(path, "?"); i >= 0 {
		u, _ = url.JoinPath(s.HTTPBaseURL, path[:i])
		u += path[i:]
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	s.authorize(req, k)
	return httpClient.Do(req)
}

// socketURL mints a WebSocket ticket and builds the URL to open.
func (s *Session) socketURL(k *Key) (string, error) {
	resp, err := s.do(k, http.MethodPost, "/api/auth/websocket-ticket/rpc", nil, "")
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusNotFound {
		// Servers older than ticket audiences only have the bare route.
		resp.Body.Close()
		if resp, err = s.do(k, http.MethodPost, "/api/auth/websocket-ticket", nil, ""); err != nil {
			return "", err
		}
	}
	var t struct {
		Ticket string `json:"ticket"`
	}
	if err := readJSON(resp, "WebSocket ticket", &t); err != nil {
		return "", err
	}
	ws, err := url.Parse(s.WSBaseURL)
	if err != nil {
		return "", err
	}
	if ws.Path == "" || ws.Path == "/" {
		ws.Path = "/ws"
	}
	q := ws.Query()
	q.Set("orchestrationProtocol", protocol)
	q.Set("wsTicket", t.Ticket)
	if !s.Bearer {
		// The upgrade proof names the socket as an HTTP URL and binds the ticket.
		scheme := "http"
		if ws.Scheme == "wss" {
			scheme = "https"
		}
		q.Set("dpop", k.Proof("GET", fmt.Sprintf("%s://%s%s", scheme, ws.Host, ws.Path), t.Ticket))
		q.Set("connectionMethod", "relay")
	}
	q.Set("clientSurface", "desktop")
	ws.RawQuery = q.Encode()
	return ws.String(), nil
}

// HistoryPage is one page of older turn items from a bounded snapshot's cursor.
func (c *Conn) HistoryPage(threadID, cursor string) (*HistoryPage, error) {
	resp, err := c.sess.do(c.key, http.MethodGet, "/api/orchestration/threads/"+url.PathEscape(threadID)+"/history?cursor="+url.QueryEscape(cursor), nil, "")
	if err != nil {
		return nil, err
	}
	var page HistoryPage
	return &page, readJSON(resp, "Thread history", &page)
}

// UploadImage puts an image on the environment for a message attachment.
func (c *Conn) UploadImage(name, mime string, data []byte) (Attachment, error) {
	meta := Attachment{Type: "image", Name: name, MimeType: mime, SizeBytes: len(data)}
	var up struct {
		AttachmentID string `json:"attachmentId"`
		RelativeURL  string `json:"relativeUrl"`
	}
	if err := c.Call("attachments.createUploadUrl", meta, &up); err != nil {
		return meta, err
	}
	base, _ := url.Parse(c.sess.HTTPBaseURL)
	ref, err := url.Parse(up.RelativeURL)
	if err != nil {
		return meta, err
	}
	// The URL is signed, so the upload itself carries no auth.
	resp, err := httpClient.Post(base.ResolveReference(ref).String(), mime, bytes.NewReader(data))
	if err != nil {
		return meta, err
	}
	if err := readJSON(resp, "Image upload", nil); err != nil {
		return meta, err
	}
	meta.ID = up.AttachmentID
	return meta, nil
}

// FetchAttachment downloads a sent attachment through the signed URL
// assets.createUrl makes for it.
func (c *Conn) FetchAttachment(id string) ([]byte, error) {
	var res struct {
		RelativeURL string `json:"relativeUrl"`
	}
	resource := map[string]any{"_tag": "attachment", "attachmentId": id, "disposition": "inline"}
	if err := c.Call("assets.createUrl", map[string]any{"resource": resource}, &res); err != nil {
		return nil, err
	}
	base, _ := url.Parse(c.sess.HTTPBaseURL)
	ref, err := url.Parse(res.RelativeURL)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Get(base.ResolveReference(ref).String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &Error{What: "Attachment", Status: resp.StatusCode}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// MediaSocketURL is the URL for one of the environment's media sockets (a
// preview tab's frames, a device screen) at path, with query. It mints a
// media ticket the way socketURL mints the RPC one.
func MediaSocketURL(s *Session, k *Key, path string, query map[string]string) (string, error) {
	resp, err := s.do(k, http.MethodPost, "/api/auth/websocket-ticket/media", nil, "")
	if err != nil {
		return "", err
	}
	var t struct {
		Ticket string `json:"ticket"`
	}
	if err := readJSON(resp, "media ticket", &t); err != nil {
		return "", err
	}
	u, err := url.Parse(s.HTTPBaseURL)
	if err != nil {
		return "", err
	}
	u = u.JoinPath(path)
	scheme := u.Scheme
	u.Scheme = map[string]string{"https": "wss", "http": "ws"}[scheme]
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	q.Set("wsTicket", t.Ticket)
	if !s.Bearer {
		q.Set("dpop", k.Proof("GET", fmt.Sprintf("%s://%s%s", scheme, u.Host, u.Path), t.Ticket))
		q.Set("connectionMethod", "relay")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
