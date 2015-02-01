Authy
=====

![Authy temporary logo](https://raw.githubusercontent.com/gophergala/authy/master/logo.png)

Authy is a go library that acts as an oauth authentication middleware for [net/http](http://golang.org/pkg/net/http),
it aims to provide drop-in support for most OAuth 1 (not implemented yet) and 2 providers. It is inspired from node.js
libraries such as [grant](https://github.com/simov/grant) or [everyauth](https://github.com/bnoguchi/everyauth).

The current OAuth implementation is kinda rough and basic but should do the trick.

Providers
---------

The current list of providers is a verbatim adaptation of the one provided by [grant](https://github.com/simov/grant).

[`500px`](https://developers.500px.com/)
[`amazon`](http://login.amazon.com/documentation)
[`angellist`](https://angel.co/api)
[`appnet`](https://developers.app.net/reference/resources/)
[`asana`](http://developer.asana.com/documentation/)
[`assembla`](http://api-doc.assembla.com/)
[`basecamp`](https://github.com/basecamp/bcx-api/)
[`bitbucket`](https://confluence.atlassian.com/display/BITBUCKET)
[`bitly`](http://dev.bitly.com)
[`box`](https://developers.box.com/)
[`buffer`](http://dev.buffer.com)
[`cheddar`](https://cheddarapp.com/developer/)
[`coinbase`](https://www.coinbase.com/docs/api/overview)
[`dailymile`](http://www.dailymile.com/api/documentation)
[`dailymotion`](https://developer.dailymotion.com/documentation#graph-api)
[`deezer`](http://developers.deezer.com/)
[`deviantart`](https://www.deviantart.com/developers/)
[`digitalocean`](https://developers.digitalocean.com/)
[`disqus`](https://disqus.com/api/docs/)
[`dropbox`](https://www.dropbox.com/developers)
[`edmodo`](https://developers.edmodo.com/)
[`elance`](https://www.elance.com/q/api2)
[`eventbrite`](http://developer.eventbrite.com/)
[`evernote`](https://dev.evernote.com/doc/)
[`everyplay`](https://developers.everyplay.com/)
[`eyeem`](https://www.eyeem.com/developers)
[`facebook`](https://developers.facebook.com)
[`feedly`](https://developer.feedly.com/)
[`fitbit`](http://dev.fitbit.com/)
[`flattr`](http://developers.flattr.net/)
[`flickr`](https://www.flickr.com/services/api/)
[`flowdock`](https://www.flowdock.com/api)
[`foursquare`](https://developer.foursquare.com/)
[`geeklist`](http://hackers.geekli.st/)
[`getpocket`](http://getpocket.com/developer/)
[`github`](http://developer.github.com)
[`gitter`](https://developer.gitter.im/docs/welcome)
[`goodreads`](https://www.goodreads.com/api)
[`google`](https://developers.google.com/)
[`harvest`](https://github.com/harvesthq/api)
[`heroku`](https://devcenter.heroku.com/categories/platform-api)
[`imgur`](https://api.imgur.com/)
[`instagram`](http://instagram.com/developer)
[`jawbone`](https://jawbone.com/up/developer/)
[`linkedin`](http://developer.linkedin.com)
[`live`](http://msdn.microsoft.com/en-us/library/dn783283.aspx)
[`mailchimp`](http://apidocs.mailchimp.com/)
[`meetup`](http://www.meetup.com/meetup_api/)
[`mixcloud`](http://www.mixcloud.com/developers/)
[`odesk`](https://developers.odesk.com)
[`openstreetmap`](http://wiki.openstreetmap.org/wiki/API_v0.6)
[`paypal`](https://developer.paypal.com/docs/)
[`podio`](https://developers.podio.com/)
[`rdio`](http://www.rdio.com/developers/)
[`redbooth`](https://redbooth.com/api/)
[`reddit`](http://www.reddit.com/dev/api)
[`runkeeper`](http://developer.runkeeper.com/healthgraph/overview)
[`salesforce`](https://www.salesforce.com/us/developer/docs/api_rest)
[`shopify`](http://docs.shopify.com/api)
[`skyrock`](http://www.skyrock.com/developer/documentation/)
[`slack`](https://api.slack.com/)
[`slice`](https://developer.slice.com/)
[`soundcloud`](http://developers.soundcloud.com)
[`spotify`](https://developer.spotify.com)
[`stackexchange`](https://api.stackexchange.com)
[`stocktwits`](http://stocktwits.com/developers)
[`strava`](http://strava.github.io/api/)
[`stripe`](https://stripe.com/docs)
[`traxo`](https://developer.traxo.com/)
[`trello`](https://trello.com/docs/)
[`tripit`](https://www.tripit.com/developer)
[`tumblr`](http://www.tumblr.com/docs/en/api/v2)
[`twitch`](https://github.com/justintv/twitch-api)
[`twitter`](https://dev.twitter.com)
[`uber`](https://developer.uber.com/v1/api-reference/)
[`vimeo`](https://developer.vimeo.com/)
[`vk`](http://vk.com/dev)
[`withings`](http://oauth.withings.com/api)
[`wordpress`](https://developer.wordpress.com/docs/api/)
[`xing`](https://dev.xing.com/docs)
[`yahoo`](https://developer.yahoo.com/)
[`yammer`](https://developer.yammer.com/)
[`yandex`](http://api.yandex.com/)
[`zendesk`](https://developer.zendesk.com/rest_api/docs/core/introduction)

Usage
-----

With [martini](https://github.com/go-martini/martini):

`server.go`
```go
package main

import (
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/gophergala/authy/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"os"
)

type Config struct {
	Secret string       `json:"secret"`
	Authy  authy.Config `json:"authy"`
}

func readConfig() (Config, error) {
	f, err := os.Open("config.json")
	if err != nil {
		return Config{}, err
	}

	decoder := json.NewDecoder(f)

	var config Config
	decoder.Decode(&config)

	return config, nil
}

func main() {
	// read app config (and authy config)
	config, err := readConfig()
	if err != nil {
		panic(err)
	}

	// setup Martini
	m := martini.Classic()
	m.Use(sessions.Sessions("authy", sessions.NewCookieStore([]byte(config.Secret))))
	// register our middleware
	m.Use(authy.Authy(config.Authy))
	m.Use(render.Renderer())

	// see the LoginRequired middleware, automatically redirect to the login page if necessary
	m.Get("/generic_callback", authy.LoginRequired(), func(token authy.Token, r render.Render) {
		r.HTML(200, "callback", token)
	})

	m.Run()
}
```

`templates/callback.tmpl`
```html
<html>
<body>
	<h2>{{.Value}} <small>({{.Scope}})</small></h2>
</body>
</html>
```

`config.json`
```json
{
	"authy": {
		"login_page": "/login",
		"callback": "/generic_callback",
		"providers": {
			"github": {
				"key": "my-app-key",
				"secret": "my-app-secret",
				"scope": ["repo", "user:email"]
			}
		}
	}
}
```
