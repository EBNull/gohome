package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"
)

func serveHttp(ctx context.Context, db *LinkDB, hostnames []string) error {
	http.HandleFunc("/", httpErrorWrap(
		func(w *bufWriter, r *http.Request, err error) {
			serr := "nil"
			if err != nil {
				serr = fmt.Sprintf("%#v", err.Error())
			}
			log.Printf("Handled request from %s: %s %s (HTTP %d, %d bytes, error=%s)\n", r.RemoteAddr, r.Method, r.URL.Path, w.Code, w.Body.Len(), serr)
		},
		func(w http.ResponseWriter, r *http.Request) error {
			g := goHttp{w, r}
			p := strings.TrimPrefix(r.URL.Path, "/")

			switch {
			case p == "":
				return g.handleRoot()
			case p == "_/pref":
				return g.handlePref()
			case p == "_/view":
				return g.handleView(db)
			case p == "favicon.ico":
				fallthrough
			case strings.HasPrefix(p, ".well-known"):
				fallthrough
			case strings.HasPrefix(p, "."):
				fallthrough
			case strings.HasPrefix(p, "_"):
				w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", 2*time.Hour/time.Second))
				http.NotFound(w, r)
				return nil
			default:
				return g.handleLink(p, db.Lookup(p), *flagChain)
			}
		}))

	return listen(ctx, *flagBind, hostnames)
}

type goHttp struct {
	W http.ResponseWriter
	R *http.Request
}

func executeTmpl(w http.ResponseWriter, status int, titleSuffix string, tplName string, data any) error {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "header.tmpl", struct{ TitleSuffix string }{titleSuffix}); err != nil {
		return err
	}
	if err := tmpl.ExecuteTemplate(w, tplName, data); err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, "footer.tmpl", struct {
		Version   string
		BuildDate string
	}{version, date})
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
	data := struct {
		NoRedir    string
		NoChain    string
		AddLinkUrl string
		CanChain   bool
	}{g.getPref("no-redirect", "0"), g.getPref("no-chain", "0"), *flagAddLinkUrl, *flagChain != ""}
	return executeTmpl(g.W, http.StatusOK, "", "index.tmpl", data)
}

func (g *goHttp) handleView(db *LinkDB) error {
	data := struct {
		Links  map[string]Link
		Prefix string
	}{db.links, g.R.Host}
	return executeTmpl(g.W, http.StatusOK, " - View", "view.tmpl", data)
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
	return executeTmpl(g.W, http.StatusOK, fmt.Sprintf(" - go/%s", link), "linkinfo.tmpl", nil)
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
	}

	return executeTmpl(g.W, http.StatusNotFound, " - 404", "not_found.tmpl", struct {
		Name       string
		ChainTo    string
		AddLinkUrl string
	}{name, redir, *flagAddLinkUrl})
}

func (g *goHttp) handlePref() error {
	q := g.R.URL.Query()
	k := q.Get("k")
	set := q.Has("v")
	v := q.Get("v")
	if !slices.Contains([]string{"no-redirect", "no-chain"}, k) {
		return executeTmpl(g.W, http.StatusNotFound, " - Preference not found", "bad_preference.tmpl", struct{ Name string }{k})
	}
	if set {
		g.setPref(k, v)
		return executeTmpl(g.W, http.StatusOK, " - Changed preference", "change_preference.tmpl",
			struct {
				Name  string
				Value string
			}{k, v},
		)
	}
	v = g.getPref(k, "")
	return executeTmpl(g.W, http.StatusOK, " - Get Preference", "get_preference.tmpl", struct {
		Name  string
		Value string
	}{k, v})
}
