package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

func getOutboundIPs(dest string) []net.IP {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:53", dest))
	if err != nil {
		return []net.IP{}
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return []net.IP{localAddr.IP}
}

func localNamesFromBind(ctx context.Context, addr net.Addr) ([]string, string) {
	lhs, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		log.Printf("Could not split hostport: %s", err)
		return nil, ""
	}
	lh := net.ParseIP(lhs)
	lookups := []net.IP{}
	isEmpty := false
	if lh.Equal(net.IPv4zero) {
		isEmpty = true
		lookups = append(lookups, getOutboundIPs("127.0.0.1")...)
		lookups = append(lookups, getOutboundIPs("8.8.8.8")...)
	}
	if lh.Equal(net.IPv6zero) {
		isEmpty = true
		lookups = append(lookups, getOutboundIPs("127.0.0.1")...)
		lookups = append(lookups, getOutboundIPs("8.8.8.8")...)
		lookups = append(lookups, getOutboundIPs("[::1]")...)
		lookups = append(lookups, getOutboundIPs("[2001:4860:4860::8888]")...)
	}
	if !isEmpty {
		lookups = append(lookups, lh)
	}
	names := map[string]struct{}{}
	for _, l := range lookups {
		hns, err := new(net.Resolver).LookupAddr(ctx, l.String())
		if err != nil {
			log.Printf("Could not look up '%s': %s", l, err)
		}
		for _, n := range hns {
			names[n] = struct{}{}
		}
	}
	n := make([]string, len(names))

	i := 0
	for k := range names {
		n[i] = k
		i++
	}
	return n, port
}

func listen(ctx context.Context, addr string, hostnames []string) error {
	log.Printf("Binding to %s\n", addr)
	l, err := (&(net.ListenConfig{})).Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on http://%s\n", l.Addr())
	bindNames, port := localNamesFromBind(ctx, l.Addr())
	hostnames = append(hostnames, bindNames...)
	printed := map[string]struct{}{}
	for _, hn := range hostnames {
		_, ok := printed[hn]
		if ok {
			continue
		}
		printed[hn] = struct{}{}
		lps := fmt.Sprintf(":%s", port)
		if lps == ":80" {
			lps = ""
		}
		log.Printf("Resolvable at http://%s%s\n", hn, lps)
	}
	s := &http.Server{Addr: l.Addr().String()}
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		s.Shutdown(ctx)
	}()
	return s.Serve(l)
}

type bufWriter struct {
	Code    int
	headers http.Header
	Body    bytes.Buffer
	once    sync.Once
}

func (b *bufWriter) Header() http.Header {
	b.once.Do(func() {
		b.headers = make(http.Header)
	})
	return b.headers
}

func (b *bufWriter) WriteHeader(status int) {
	b.Code = status
}

func (b *bufWriter) Write(d []byte) (int, error) {
	return b.Body.Write(d)
}

func (b *bufWriter) WriteTo(d http.ResponseWriter, skipHeaders bool) (int, error) {
	h := d.Header()
	for key, val := range b.headers {
		h[key] = val
	}
	if !skipHeaders {
		d.WriteHeader(b.Code)
	}
	return d.Write(b.Body.Bytes())
}

var _ http.ResponseWriter = &bufWriter{}

func httpErrorWrap(reportf func(w *bufWriter, r *http.Request, err error), f func(w http.ResponseWriter, r *http.Request) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bw := &bufWriter{}
		err := f(bw, r)
		reportf(bw, r, err)
		if err != nil {
			http.Error(w, fmt.Sprintf("500 Internal Server Error\n\n%s", err.Error()), http.StatusInternalServerError)
			if bw.Body.Len() > 0 {
				w.Write([]byte("\n\n\nOutput buffer prior to error:\n\n"))
				bw.WriteTo(w, true)
			}
			return
		}
		bw.WriteTo(w, false)
	}
}

func getCookie(r *http.Request, name string, def string) (string, bool) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return def, false
	}
	return cookie.Value, true
}
