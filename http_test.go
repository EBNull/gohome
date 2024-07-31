package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPref(t *testing.T) {
	tests := []struct {
		desc   string
		params map[string]string
		cookie *http.Cookie
		want   string
	}{
		{
			desc:   "no cookie or param, return default",
			params: nil,
			cookie: &http.Cookie{},
			want:   "default",
		},
		{
			desc:   "param set, no cookie",
			params: map[string]string{"test": "rofl"},
			cookie: &http.Cookie{},
			want:   "rofl",
		},
		{
			desc:   "param and cookie set, param wins",
			params: map[string]string{"test": "yeehaw"},
			cookie: &http.Cookie{Name: "test", Value: "cowboy"},
			want:   "yeehaw",
		},
		{
			desc:   "cookie set, no params, cookie wins",
			params: nil,
			cookie: &http.Cookie{Name: "pref-test", Value: "cowboy"},
			want:   "cowboy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			req, err := http.NewRequest("GET", "", nil)
			if err != nil {
				t.Fatal(err)
			}

			p := req.URL.Query()
			for k, v := range tc.params {
				p.Add(k, v)
			}

			req.URL.RawQuery = p.Encode()
			req.AddCookie(tc.cookie)

			rr := httptest.NewRecorder()
			g := goHttp{
				W: rr,
				R: req,
			}

			if got := g.getPref("test", "default"); tc.want != got {
				t.Errorf("getPref('test', 'default') = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSetPref(t *testing.T) {
	req, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	g := goHttp{
		W: rr,
		R: req,
	}

	g.setPref("test", "cowboy")

	if len(rr.Result().Cookies()) != 1 {
		t.Fatalf("g.SetPref('test', 'cowboy') expected 1 cookie in result, found %v cookies", len(rr.Result().Cookies()))
	}

	cookie := rr.Result().Cookies()[0]
	if cookie.Name != "pref-test" {
		t.Errorf("g.setPref('test', 'cowboy') got cookie name %v, want %v", cookie.Name, "pref-test")
	}

	if cookie.Value != "cowboy" {
		t.Errorf("g.setPref('test', 'cowboy') = %v, want %v", cookie.Value, "cowboy")
	}
}
