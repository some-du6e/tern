package t3

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"sync/atomic"
	"time"
)

var b64 = base64.RawURLEncoding

// Key is the DPoP key that binds relay and environment tokens to this
// install. It's generated once and kept in the store.
type Key struct {
	priv       *ecdsa.PrivateKey
	jwk        map[string]string
	Thumbprint string
	skew       atomic.Int64 // server clock minus ours, in seconds
}

func LoadKey(s *Store) (*Key, error) {
	var der []byte
	ok, err := s.Get("dpop.key", &der)
	if err != nil {
		return nil, err
	}
	var priv *ecdsa.PrivateKey
	if ok {
		k, err := x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			return nil, err
		}
		priv = k.(*ecdsa.PrivateKey)
	} else {
		if priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
			return nil, err
		}
		if der, err = x509.MarshalPKCS8PrivateKey(priv); err != nil {
			return nil, err
		}
		if err := s.Put("dpop.key", der); err != nil {
			return nil, err
		}
	}
	return newKey(priv)
}

func newKey(priv *ecdsa.PrivateKey) (*Key, error) {
	pub, err := priv.PublicKey.ECDH()
	if err != nil {
		return nil, err
	}
	b := pub.Bytes() // 0x04 || X || Y
	x, y := b64.EncodeToString(b[1:33]), b64.EncodeToString(b[33:65])
	// RFC 7638: required members in lexicographic order, no whitespace.
	sum := sha256.Sum256([]byte(`{"crv":"P-256","kty":"EC","x":"` + x + `","y":"` + y + `"}`))
	return &Key{
		priv:       priv,
		jwk:        map[string]string{"kty": "EC", "crv": "P-256", "x": x, "y": y},
		Thumbprint: b64.EncodeToString(sum[:]),
	}, nil
}

// Proof is a single-use DPoP proof for one request. boundToken, when set,
// fills ath with the token's hash.
func (k *Key) Proof(method, rawURL, boundToken string) string {
	htu := rawURL
	if u, err := url.Parse(rawURL); err == nil {
		u.RawQuery, u.Fragment = "", ""
		htu = u.String()
	}
	payload := map[string]any{
		"htm": method,
		"htu": htu,
		"jti": newUUID(),
		"iat": time.Now().Unix() + k.skew.Load(),
	}
	if boundToken != "" {
		sum := sha256.Sum256([]byte(boundToken))
		payload["ath"] = b64.EncodeToString(sum[:])
	}
	head, _ := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": k.jwk})
	body, _ := json.Marshal(payload)
	input := b64.EncodeToString(head) + "." + b64.EncodeToString(body)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, k.priv, digest[:])
	if err != nil {
		panic(err) // only fails if the system RNG does
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return input + "." + b64.EncodeToString(sig)
}

// noteClock records the server's clock from a Date header, so the next
// proof's iat lands inside the ±300 s window on a drifting machine.
func (k *Key) noteClock(date string) bool {
	t, err := time.Parse(time.RFC1123, date)
	if err != nil {
		return false
	}
	k.skew.Store(int64(time.Until(t).Seconds()))
	return true
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	const hex = "0123456789abcdef"
	out := make([]byte, 0, 36)
	for i, c := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out = append(out, '-')
		}
		out = append(out, hex[c>>4], hex[c&15])
	}
	return string(out)
}

// NewID is a random v4 UUID, for command and message ids.
func NewID() string { return newUUID() }
