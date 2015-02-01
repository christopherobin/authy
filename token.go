package authy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/christopherobin/authy/oauth2"
	"net/http"
	"time"
)

// Token is returned on a successful auth
type Token struct {
	// Authy instance this token was generated with
	authy Authy `json:"-"`
	// The OAuth version of the token
	Version int `json:"version"`
	// The provider on which that token can be used
	Provider string `json:"provider"`
	// The actual value of the token
	Value string `json:"value"`
	// The scopes returned by the provider, some providers may allow the user to change the scope of an auth request
	// Make sure to check the available scopes before doing queries on their webservices
	Scope []string `json:"scope"`
	// The type of token
	Type string `json:"type"`
	// Expiry date
	Expires *time.Time `json:"time"`
	// The refresh token if one
	RefreshToken string `json:"refresh_token"`
}

func tokenFromOAuth2(a Authy, provider string, t oauth2.Token) *Token {
	return &Token{
		authy:        a,
		Version:      2,
		Provider:     provider,
		Value:        t.AccessToken,
		Scope:        t.Scope,
		Type:         t.Type,
		Expires:      t.Expires,
		RefreshToken: t.RefreshToken,
	}
}

// convert to oauth2 token
func (t *Token) oauth2() oauth2.Token {
	return oauth2.Token{
		AccessToken:  t.Value,
		Scope:        t.Scope,
		Type:         t.Type,
		Expires:      t.Expires,
		RefreshToken: t.RefreshToken,
	}
}

// Deserialize a token back from its serialized form
func (a Authy) TokenFromSerialized(data []byte) (*Token, error) {
	var t Token
	err := json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}

	// assign authy
	t.authy = a
	return &t, nil
}

// Returns true if token is expired
func (t *Token) Expired() bool {
	if t.Expires == nil {
		return false
	}
	return time.Now().After(*t.Expires)
}

// Whether or not the token can be refreshed via the provider's api
func (t *Token) IsRefreshable() bool {
	return t.Version == 2 && t.RefreshToken != ""
}

// Try to refresh token
func (t *Token) Refresh() error {
	if !t.IsRefreshable() {
		return errors.New("Token cannot be refreshed")
	}

	providerConfig, ok := t.authy.providers[t.Provider]
	if ok != true {
		return errors.New(fmt.Sprintf("unknown provider %s", t.Provider))
	}

	if t.Version == 2 {
		newToken, err := oauth2.Refresh(providerConfig, t.oauth2())
		if err != nil {
			return err
		}

		t.RefreshToken = newToken.RefreshToken
		t.Value = newToken.AccessToken
		t.Expires = newToken.Expires
		t.Type = newToken.Type
	}

	return nil
}

// Serialize a token in a format that Authy can decode later, useful for session storage
// For now use JSON, maybe switch to encoding/gob or capnproto later if we need more perfs
func (t *Token) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Quick transport implementation for an oauth client
type TokenTransport struct {
	token     Token
	transport http.RoundTripper
}

func NewTokenTranport(t Token) *TokenTransport {
	return &TokenTransport{
		token:     t,
		transport: http.DefaultTransport,
	}
}

func (tt *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// make a copy of the request object (requested by RoundTripper interface)
	newReq := *req

	// deepcopy headers
	newReq.Header = make(http.Header)
	for name, val := range req.Header {
		valCopy := make([]string, len(val))
		copy(valCopy, val)
		newReq.Header[name] = valCopy
	}

	if !tt.token.Expired() {
		newReq.Header["Authorization"] = []string{"Bearer " + tt.token.Value}
	}

	return tt.transport.RoundTrip(&newReq)
}

// Return a http.Client to be used to query distant APIs
func (t Token) Client() *http.Client {
	return &http.Client{
		Transport: NewTokenTranport(t),
	}
}
