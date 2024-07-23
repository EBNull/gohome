package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"syscall"
	"time"

	"github.com/ebnull/gohome/network"
)

func main() {
	err := mainImpl(os.Args)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func onCleanupSignalOrDone(doneChan <-chan struct{}, f ...func() error) func() {
	return func() {
		sigchan := make(chan os.Signal)
		signal.Notify(sigchan, os.Interrupt)
		signal.Notify(sigchan, syscall.SIGTERM)
		var s os.Signal
		select {
		case s = <-sigchan:
			ss, ok := s.(syscall.Signal)
			if !ok {
				log.Printf("Caught signal: %s", s.String())
			} else {
				log.Printf("Caught signal %d:  %s", int(ss), ss.String())
			}
		case <-doneChan:
		}
		signal.Stop(sigchan)
		for _, f := range f {
			err := f()
			if err != nil {
				log.Print(err)
			}
		}
		if s != nil {
			ss, ok := s.(syscall.Signal)
			log.Printf("Reraising signal %s\n", s)
			if !ok {
				os.Exit(1)
			}
			syscall.Kill(syscall.Getpid(), ss)
		}
	}
}

func runBackgroundUpdate(ctx context.Context, db *LinkDB) {
	if db.Len() == 0 {
		err := updateLinksFromRemote(db, *flagRemote, *flagCache)
		if err != nil {
			log.Printf("Error fetching inital golinks: %s\n", err)
		}
	}
	go func(ctx context.Context) {
		for {
			select {
			case <-time.After(*flagUpdateInterval):
				err := updateLinksFromRemote(db, *flagRemote, *flagCache)
				if err != nil {
					log.Printf("Error fetching golinks: %s\n", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
}

func setupAutoconfig(ctx context.Context) ([]string, error) {
	if !slices.Contains([]string{"darwin", "linux"}, runtime.GOOS) {
		log.Printf("GOOS is %s; skipping loopback alias and editing of /etc/hosts", runtime.GOOS)
		return nil, nil
	}

	host, _, err := net.SplitHostPort(*flagBind)
	if err != nil {
		return nil, err
	}
	am, err := network.NewAliasManager(*flagHostname, host)
	if err != nil {
		return nil, fmt.Errorf("%w\n\nHint: are you root? Try again with sudo.", err)
	}
	err, stop := am.Start()
	if err != nil {
		stop()
		return nil, err
	}
	go onCleanupSignalOrDone(ctx.Done(), func() error {
		stop()
		return nil
	})()
	if ok, _ := am.Exists(); ok {
		return []string{am.Host()}, nil
	}
	return nil, fmt.Errorf("Could not set up name resolution for %s to %s", *flagHostname, host)
}

func mainImpl(argv []string) error {
	err := handleFlags(argv)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := &LinkDB{}
	if err := db.LoadJson(*flagCache); err != nil {
		return err
	}

	if *flagRemote == "" {
		log.Printf("There is no remote configured; no golinks will be downloaded")
	} else {
		runBackgroundUpdate(ctx, db)
	}

	if *flagChain == "" {
		log.Printf("There is no chain configured; no redirection will occur on missing links")
	}

	hostResolve := []string{}
	if *flagAuto {
		hostResolve, err = setupAutoconfig(ctx)
		if err != nil {
			return err
		}
	}

	return serveHttp(ctx, db, hostResolve)
}
