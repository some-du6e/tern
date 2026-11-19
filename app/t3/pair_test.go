package t3

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Pairing reads the server's descriptor first and refuses a server that
// speaks another protocol before the one-time token is sent, as T3's
// registerPairing does; a link without a host or a token sends nothing.
func TestPairChecksTheServerFirst(t *testing.T) {
	for _, c := range []struct {
		descriptor string
		want       string // in the error; "" pairs
	}{
		{`{"environmentId":"e1","label":"studio","orchestrationProtocolVersion":3}`, "this client is not supported by this server"},
		{`{"environmentId":"e1","label":"studio"}`, "this client requires a newer server. Update T3 Code on studio to connect"},
		{`{"environmentId":"e1","label":"studio","orchestrationProtocolVersion":2}`, ""},
	} {
		var redeemed atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/t3/environment":
				_, _ = w.Write([]byte(c.descriptor))
			case "/oauth/token":
				redeemed.Add(1)
				_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
			}
		}))
		store, err := OpenStoreAt(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		_, err = Pair(store, srv.URL+"/pair#token=one-time")
		srv.Close()
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s: %v", c.descriptor, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s: %v, want %q", c.descriptor, err, c.want)
		case c.want != "" && redeemed.Load() != 0:
			t.Errorf("%s: the token was sent", c.descriptor)
		}
	}
	store, _ := OpenStoreAt(t.TempDir())
	for _, link := range []string{"https://studio.example.com/pair", "not a link", "#token=abc"} {
		if _, err := Pair(store, link); !errors.Is(err, ErrPairingLink) {
			t.Errorf("%q: %v", link, err)
		}
	}
}
