package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ebnull/gohome/build"
	"github.com/ebnull/gohome/network"
)

var (
	flagCache = flag.String("cache", build.DefaultCache, "The filename to load cached golinks from")

	flagChain          = flag.String("chain", build.DefaultChain, "The remote URL to chain redirect to (if link not found in local cache)")
	flagRemote         = flag.String("remote", build.DefaultRemote, "The remote URL to update golinks from")
	flagUpdateInterval = flag.Duration("interval", func() time.Duration {
		d, err := time.ParseDuration(build.DefaultInterval)
		if err != nil {
			panic(err)
		}
		return d
	}(), "The periodic interval for downloading new golinks")

	flagBind = flag.String("bind", build.DefaultBind, "The IP and port to bind to")

	flagAuto = flag.Bool("auto", func() bool {
		b, err := strconv.ParseBool(build.DefaultAuto)
		if err != nil {
			panic(err)
		}
		return b
	}(), "Automatically alias the bind IP address to the loopback interface")
	flagHostname = flag.String("hostname", build.DefaultHostname, "The hostname to add to /etc/hosts (resolvable to the bind address)")
)

func init() {
	if runtime.GOOS == "linux" {
		li := flag.Lookup("loopback-interface")
		li.DefValue = "lo"
		li.Value.Set("lo")
	}
}

func main() {
	err := mainImpl()
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func onSigintOrDone(doneChan <-chan struct{}, f ...func() error) func() {
	return func() {
		sigchan := make(chan os.Signal)
		signal.Notify(sigchan, os.Interrupt)
		sendInt := false
		select {
		case <-sigchan:
			log.Printf("SIGINT")
			sendInt = true
		case <-doneChan:
		}
		signal.Stop(sigchan)
		for _, f := range f {
			err := f()
			if err != nil {
				log.Print(err)
			}
		}
		if sendInt {
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}
	}
}

func mainImpl() error {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := &LinkDB{}
	if err := db.LoadJson(*flagCache); err != nil {
		return err
	}
	if *flagRemote == "" {
		log.Printf("There is no remote configured; no golinks will be downloaded.")
	} else {
		if db.Len() == 0 {
			err := updateLinksFromRemote(db, *flagRemote, *flagCache)
			if err != nil {
				return err
			}
		}
		go func(ctx context.Context) {
			for {
				select {
				case <-time.After(*flagUpdateInterval):
					updateLinksFromRemote(db, *flagRemote, *flagCache)
				case <-ctx.Done():
					return
				}
			}
		}(ctx)
	}

	if *flagChain == "" {
		log.Printf("There is no chain configured, no redirection will occur on missing link")
	}

	hostResolve := []string{}
	if *flagAuto {
		if !slices.Contains([]string{"darwin", "linux"}, runtime.GOOS) {
			log.Printf("GOOS is %s; skipping loopback alias and editing of /etc/hosts", runtime.GOOS)
		} else {
			host, _, err := net.SplitHostPort(*flagBind)
			if err != nil {
				return err
			}
			am, err := network.NewAliasManager(*flagHostname, host)
			if err != nil {
				return fmt.Errorf("%w\n\nHint: are you root? Try again with sudo.", err)
			}
			err, stop := am.Start()
			if err != nil {
				stop()
				return err
			}
			go onSigintOrDone(ctx.Done(), func() error {
				stop()
				return nil
			})()
			if ok, _ := am.Exists(); ok {
				hostResolve = append(hostResolve, am.Host())
			}
		}
	}

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

	return listen(ctx, *flagBind, hostResolve)
}
