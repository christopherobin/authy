// Implements several middlewares for using Authy with Martini
package authy

import (
	"github.com/christopherobin/authy"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/sessions"
	"net/http"
	"net/url"
	"regexp"
)

type Config authy.Config
type Token authy.Token

func (t Token) Client() *http.Client {
	return authy.Token(t).Client()
}

// Takes an Authy config and returns a middleware to use with martini
// See examples below
// TODO: add error handler in config that allows the use to retrieve the context+error
func Authy(config Config) martini.Handler {
	baseRoute := "/authy"
	if config.BasePath != "" {
		baseRoute = config.BasePath
	}

	// should be moved in the authy package
	if config.PathLogin == "" {
		config.PathLogin = "/login"
	}

	authRoute := regexp.MustCompile("^" + baseRoute + "/([^/#?]+)")
	callbackRoute := regexp.MustCompile("^" + baseRoute + "/([^/]+)/callback")
	authy, err := authy.NewAuthy(authy.Config(config))

	// due to the way middleware are used, it's the cleanest? way to deal with this?
	if err != nil {
		panic(err)
	}

	return func(s sessions.Session, c martini.Context, w http.ResponseWriter, r *http.Request) {
		c.Map(config)

		// if we are already logged, ignore login route matching
		if serializedToken := s.Get("authy.token"); serializedToken != nil {
			token, err := authy.TokenFromSerialized(serializedToken.([]byte))
			// TODO: implement refresh here
			if err != nil {
				panic(err)
			}
			c.Map(Token(*token))
			return
		}

		// match authorization URL
		matches := authRoute.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 && matches[0] == r.URL.Path {
			redirectUrl, err := authy.Authorize(matches[1], s, r)
			if err != nil {
				panic(err)
			}

			// redirect user to oauth website
			http.Redirect(w, r, redirectUrl, http.StatusFound)
			return
		}

		// match access URL
		matches = callbackRoute.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 && matches[0] == r.URL.Path {
			token, redirectUrl, err := authy.Access(matches[1], s, r)
			if err != nil {
				panic(err)
			}

			// save token in session
			serializedToken, err := token.Serialize()
			if err != nil {
				panic(err)
			}
			s.Set("authy.token", serializedToken)

			http.Redirect(w, r, redirectUrl, http.StatusFound)
			return
		}
	}
}

// Use this middleware on the routes where you need the user to be logged in
func LoginRequired() martini.Handler {
	return func(config Config, s sessions.Session, w http.ResponseWriter, r *http.Request) {
		if tokenValue := s.Get("authy.token"); tokenValue == nil {
			next := url.QueryEscape(r.URL.RequestURI())
			http.Redirect(w, r, config.PathLogin+"?next="+next, http.StatusFound)
		}
	}
}
