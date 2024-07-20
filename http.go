package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ebnull/gohome/build"
)

var flagAddLinkUrl = flag.String("add-link-url", build.DefaultAddLinkUrl, "The url to add a new golink. If set a link will be displayed when a golink is not found.")

func htmlTemplate(w http.ResponseWriter, status int, tpl string, data any) error {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	t, err := template.New("foo").Parse(tpl)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

type goHttp struct {
	W http.ResponseWriter
	R *http.Request
}

func (g *goHttp) getPref(key string, def string) string {
	if g.R.URL.Query().Has(key) {
		return g.R.URL.Query().Get(key)
	}
	v, ok := getCookie(g.R, fmt.Sprintf("pref-%s", key), def)
	if !ok {
		return def
	}
	return v
}

func (g *goHttp) setPref(key string, v string) error {
	cookie := &http.Cookie{
		Name:     fmt.Sprintf("pref-%s", key),
		Value:    v,
		Path:     "/",
		MaxAge:   int(10 * 365 * 24 * time.Hour / time.Second), // Ten years
		HttpOnly: true,
		Secure:   false,                // We serve HTTP, not HTTPS. Also, prefs are not sensitive.
		SameSite: http.SameSiteLaxMode, // SameSite=None requires Secure=true
	}
	http.SetCookie(g.W, cookie)
	return nil
}

func (g *goHttp) handleRoot() error {
	return htmlTemplate(g.W,
		http.StatusOK,
		`<h1>gohome</h1>
		<p>The local go link redirector</p>
		{{if .AddLinkUrl}}<p><a href="{{.AddLinkUrl}}">Add a new link</a></p>{{end}}
		<table><tr><th>Pref</th><th>Value</th><th>Description</th><th></th></tr>
		<tr>
		  <th>no-redirect</th><td>{{.NoRedir}}</td>
		  <td>If nonzero, render a html page to preview the link destination instead of automatically redirecting.</td>
		  <td><a href="/_/pref?k=no-redirect&v=1">Enable</a> | <a href="/_/pref?k=no-redirect&v=0">Disable</a></td>
		</tr>
		<tr>
		  <th>no-chain</th><td>{{.NoChain}}</td>
		  <td>If nonzero and a golink is not found render a html page instead of automatically redirecting to upstream.</td>
		  <td><a href="/_/pref?k=no-chain&v=1">Enable</a> | <a href="/_/pref?k=no-chain&v=0">Disable</a></td>
		</tr>
		</table>
		</p>
		<style>
			tr th {
			  font-family: monospace;
   			  white-space: pre;
			}
			tr td:first {
			  font-family: monospace;
   			  white-space: pre;
			}
			a {
			  color: LinkText !important
		        }
		</style>
		`, struct {
			NoRedir    string
			NoChain    string
			AddLinkUrl string
		}{g.getPref("no-redirect", "0"), g.getPref("no-chain", "0"), *flagAddLinkUrl},
	)
}

func (g *goHttp) handleLink(name string, l *Link, chainUrl string) error {
	if l == nil {
		return g.linkMissing(name, chainUrl)
	}
	return g.linkFound(l)
}

func (g *goHttp) linkFound(link *Link) error {
	log.Printf("Found link go/%s -> %s\n", link.Display, link.Destination)
	if g.getPref("no-redirect", "0") == "0" {
		http.Redirect(g.W, g.R, link.Destination, http.StatusTemporaryRedirect)
		return nil
	}
	return htmlTemplate(g.W,
		http.StatusOK,
		`<h1>go/{{.Display}}</h1><a href="/{{.Display}}">go/{{.Display}}</a> redirects to <a href="{{.Destination}}">{{.Destination}}</a>`,
		link,
	)
}

func (g *goHttp) linkMissing(name string, chainUrl string) error {
	redir := chainUrl
	if strings.Contains(chainUrl, "%") {
		redir = fmt.Sprintf(chainUrl, name)
	}
	if chainUrl != "" {
		if g.getPref("no-redirect", "0") == "0" && g.getPref("no-chain", "0") == "0" {
			log.Printf("Missing link go/%s; chaining to %s\n", name, redir)
			http.Redirect(g.W, g.R, redir, http.StatusTemporaryRedirect)
			return nil
		}
		log.Printf("Missing link go/%s; would chain to %s\n", name, redir)
	} else {
		log.Printf("Missing link go/%s; chaining not configured", name)

		if name == "links" {
			return htmlTemplate(g.W,
				http.StatusNotFound,
				`<h1>Not Found</h1><pre style="display: inline">go/{{.Name}}</pre> does not redirect anywhere.<p>This UI does not support adding links.`,
				struct {
					Name    string
					ChainTo string
				}{name, redir},
			)

		}
	}

	return htmlTemplate(g.W,
		http.StatusNotFound,
		`<h1>Not Found</h1><pre style="display: inline">go/{{.Name}}</pre> does not redirect anywhere. Maybe you'd like to <a href="/links">add it</a>?{{if .ChainTo}} Or try <a href="{{.ChainTo}}">upstream</a>?{{end}}`,
		struct {
			Name    string
			ChainTo string
		}{name, redir},
	)
}

func (g *goHttp) handlePref() error {
	q := g.R.URL.Query()
	k := q.Get("k")
	set := q.Has("v")
	v := q.Get("v")
	if !slices.Contains([]string{"no-redirect", "no-chain"}, k) {
		return htmlTemplate(g.W,
			http.StatusNotFound,
			`<h1>Unknown Preference</h1>The pref <pre style="display: inline">{{.Name}}</pre> does not exist.<p><a href="/">Home</a>`,
			struct{ Name string }{k},
		)
	}
	if set {
		g.setPref(k, v)
		return htmlTemplate(g.W,
			http.StatusOK,
			`<meta http-equiv="refresh" content="1;URL='/'"><h1>Set Preference</h1>The pref <pre style="display: inline">{{.Name}}</pre> was set to <pre style="display: inline">{{.Value}}</pre>.<p><a href="/">Home</a>`,
			struct {
				Name  string
				Value string
			}{k, v},
		)
	}
	v = g.getPref(k, "")
	return htmlTemplate(g.W,
		http.StatusOK,
		`<h1>Get Preference</h1>The pref <pre style="display: inline">{{.Name}}</pre> is set to <pre style="display: inline">{{.Value}}</pre>.<p><a href="/">Home</a>`,
		struct {
			Name  string
			Value string
		}{k, v},
	)
}
