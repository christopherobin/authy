// This package partially implements OAuth2 for Authy
package oauth2

// see http://tools.ietf.org/html/rfc6749

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/christopherobin/authy/provider"
	"github.com/google/go-querystring/query"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// used to generate requests to the distant server
type authorizationRequest struct {
	ClientId     string `url:"client_id"`
	ResponseType string `url:"response_type"`
	RedirectURI  string `url:"redirect_uri,omitempty"`
	Scope        string `url:"scope,omitempty"`
	State        string `url:"state,omitempty"`
}

type accessTokenRequest struct {
	ClientId     string `url:"client_id"`
	ClientSecret string `url:"client_secret"`
	GrantType    string `url:"grant_type"`
	Code         string `url:"code"`
	RedirectURI  string `url:"redirect_uri,omitempty"`
}

type refreshTokenRequest struct {
	GrantType    string `url:"grant_type"`
	RefreshToken string `url:"refresh_token"`
}

type Token struct {
	AccessToken  string
	Scope        []string
	Type         string
	Expires      *time.Time
	RefreshToken string
}

// standard oauth2 error (http://tools.ietf.org/html/rfc6749#section-5.2)
type Error struct {
	Code        string
	Description string
	URI         string
	// We also pass the raw error in case the server does something funky with it's error output
	Raw map[string][]string
}

// utility function to retrieve the value of a specific entry in a decoded query string

var errorTextRe = regexp.MustCompile("[[:^print:]]|[\\\\]")
var errorURIRe = regexp.MustCompile("[[:^print:]]|[ \\\\]")

func NewError(response url.Values) (err Error) {
	err.Raw = response
	err.Code = errorTextRe.ReplaceAllString(response.Get("error"), "")
	if err.Code == "" {
		err.Code = "invalid_response"
		err.Description = "The response generated by the server could not be parsed by Authy"
		return
	}
	err.Description = errorTextRe.ReplaceAllString(response.Get("error_description"), "")
	err.URI = errorURIRe.ReplaceAllString(response.Get("error_uri"), "")

	return
}

func (err Error) Error() string {
	msg := err.Code
	if err.Description != "" {
		msg += ": " + err.Description
	}
	if err.URI != "" {
		msg += " (see " + err.URI + ")"
	}
	return msg
}

func genCallbackURL(config provider.ProviderConfig, r *http.Request) string {
	var redirectURI = url.URL{
		Host: r.Host,
		Path: r.URL.Path + "/callback",
	}

	if _, ok := r.Header["X-HTTPS"]; r.TLS != nil || ok == true {
		redirectURI.Scheme = "https"
	} else {
		redirectURI.Scheme = "http"
	}

	return redirectURI.String()
}

// create a new random token for the CSRF check
func NewState() (string, error) {
	rawState := make([]byte, 16)
	_, err := rand.Read(rawState)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(rawState), nil
}

// Generates the proper authorization URL for the given service
func AuthorizeURL(config provider.ProviderConfig, r *http.Request) (dest string, err error) {
	// subdomain support
	baseUrl := config.Provider.AuthorizeURL
	if config.Provider.Subdomain == true {
		if config.Subdomain == "" {
			err = errors.New(fmt.Sprintf("provider %s expects the config to contain your subdomain", config.Provider.Name))
			return
		}
		baseUrl = strings.Replace(baseUrl, "[subdomain]", config.Subdomain, -1)
	}

	authUrl, err := url.Parse(baseUrl)
	if err != nil {
		return
	}

	values, err := query.Values(authorizationRequest{
		ClientId:     config.Key,
		ResponseType: "code",
		RedirectURI:  genCallbackURL(config, r),
		Scope:        strings.Join(config.Scope, config.Provider.ScopeDelimiter),
		State:        config.State,
	})

	// custom parameters
	if len(config.CustomParameters) > 0 {
		for _, name := range config.Provider.CustomParameters {
			if value, ok := config.CustomParameters[name]; ok == true {
				values.Set(name, value)
			}
		}
	}

	if err != nil {
		return
	}

	authUrl.RawQuery = values.Encode()
	dest = authUrl.String()
	return
}

func parseTokenResponse(config provider.ProviderConfig, values url.Values) (token Token, err error) {
	token.AccessToken = values.Get("access_token")
	token.Type = values.Get("token_type")
	token.RefreshToken = values.Get("refresh_token")

	if token.AccessToken == "" || token.Type == "" {
		err = Error{
			Code:        "invalid_response",
			Description: "The response returned by the server couldn't be parsed by Authy",
			Raw:         values,
		}
		return
	}

	// optional stuff
	if scope := values.Get("scope"); scope != "" {
		token.Scope = strings.Split(scope, config.Provider.ScopeDelimiter)
	}

	if expires_in := values.Get("expires_in"); expires_in != "" {
		// silently ignore errors in this case, later we might add a log
		if to_add, err := strconv.ParseInt(expires_in, 10, 32); err != nil {
			expires := time.Now().Add(time.Duration(to_add) * time.Second)
			token.Expires = &expires
		}
	}

	return
}

// Query the remote service for an access token
func GetAccessToken(config provider.ProviderConfig, r *http.Request) (token Token, err error) {
	queryValues, err := query.Values(accessTokenRequest{
		ClientId:     config.Key,
		ClientSecret: config.Secret,
		Code:         r.URL.Query().Get("code"),
		GrantType:    "authorization_code",
		RedirectURI:  genCallbackURL(config, r),
	})

	if err != nil {
		return
	}

	resp, err := http.PostForm(config.Provider.AccessURL, queryValues)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return
	}

	if _, ok := values["error"]; ok == true {
		err = NewError(values)
		return
	}

	// everything went A-OK!
	token, err = parseTokenResponse(config, values)

	return
}

// Refresh an access token
func Refresh(config provider.ProviderConfig, originalToken Token) (token Token, err error) {
	queryValues, err := query.Values(refreshTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: token.RefreshToken,
	})

	if err != nil {
		return
	}

	resp, err := http.PostForm(config.Provider.AccessURL, queryValues)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return
	}

	if _, ok := values["error"]; ok == true {
		err = NewError(values)
		return
	}

	// everything went A-OK!
	token, err = parseTokenResponse(config, values)

	return
}
