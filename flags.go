package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3"

	"github.com/ebnull/gohome/build"
)

var (
	version = ""
	commit  = ""
	date    = ""
)

var (
	flagVersion          = flag.Bool("version", false, "Show version and exit")
	flagCache            = flag.String("cache", build.DefaultCache, "The filename to load cached golinks from")
	flagConfig           = flag.String("config", build.DefaultConfig, "The filename to load configuration from.\n\nArguments from the command line and environment variables override entries set here.\n\nThe file format is 'flagname value\\n' as specified by\nhttps://pkg.go.dev/github.com/peterbourgon/ff/v4#PlainParser")
	flagWriteConfig      = flag.Bool("write-config", false, "Write a default config to --config and exit.")
	flagWriteConfigForce = flag.Bool("write-config-force", false, "Same as --write-config, but overwrite the file if it exists.")

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
	flagHostname = flag.String("hostname", build.DefaultHostname, "The hostname to add to /etc/hosts for --auto mode (resolvable to the bind address)")

	flagAddLinkUrl = flag.String("add-link-url", build.DefaultAddLinkUrl, "The url to add a new golink. If set a link will be displayed when a golink is not found.")
)

func init() {
	if runtime.GOOS == "linux" {
		li := flag.Lookup("loopback-interface")
		li.DefValue = "lo"
		li.Value.Set("lo")
	}
}

func expandPath(pth string) (string, error) {
	pth = os.ExpandEnv(pth)
	if strings.HasPrefix(pth, "~/") || pth == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		pth = path.Join(home, pth[1:])
	}
	return pth, nil
}

func handleFlags(argv []string) error {
	fs := flag.CommandLine
	for i := range 2 {
		// We need to do this twice because the config file may need path expansion
		err := ff.Parse(fs, argv[1:],
			ff.WithEnvVarPrefix("GOHOME"),
			ff.WithAllowMissingConfigFile(true),
			ff.WithConfigFileFlag("config"),
			ff.WithConfigFileParser(ff.PlainParser),
		)
		if err != nil {
			return err
		}
		if i == 1 {
			break
		}
		{ // Config filename expansion and resetting
			ep, err := expandPath(*flagConfig)
			if err != nil {
				return err
			}
			if ep == *flagConfig {
				// No need to change the value
				break
			}
			// This is really ugly, but ff doesn't give us a hook to interpolate the config path
			expanded := false
			for i, s := range argv {
				// Update argv, so when ff parses this, it pulls out the expanded value
				if s == "config" && len(argv) > i && argv[i+1] == *flagConfig {
					argv[i+1] = ep
					expanded = true
					break
				}
			}
			if !expanded {
				// Not in argv, so expand the default
				// Note: We need .Value.Set() here, since .Value is set when the flag is initialized.
				fs.Lookup("config").Value.Set(ep)
				fs.Lookup("config").DefValue = ep
			}
		}
	}

	if *flagVersion {
		fmt.Printf("%s\n%s\n%s\n%s\n", "gohome", version, commit, date)
		os.Exit(0)
	}

	_, configMissingErr := os.Stat(*flagConfig)
	skip := []string{"config", "write-config", "write-config-force", "version"}

	if *flagWriteConfig || *flagWriteConfigForce {
		if configMissingErr == nil && !*flagWriteConfigForce {
			return fmt.Errorf("Config file %s already exists. To overwrite it, use --write-config-force.", *flagConfig)
		}
		out := []string{
			"# Configuration file for gohome",
			"#",
			"# https://github.com/EBNull/gohome",
			"#",
			"# Syntax:",
			"#   key value",
			"#",
			"# For syntax details, see https://pkg.go.dev/github.com/peterbourgon/ff/v3#PlainParser",
			"#",
			"",
		}
		flag.CommandLine.VisitAll(func(f *flag.Flag) {
			if slices.Contains(skip, f.Name) {
				return
			}
			initialUsage, _, _ := strings.Cut(f.Usage, "\n")
			name := f.Name
			if f.DefValue == "" {
				name = "#" + name
			}
			out = append(out, fmt.Sprintf("# %s\n%s %s\n", initialUsage, name, f.DefValue))
		})
		err := os.WriteFile(*flagConfig, []byte(strings.Join(out, "\n")), 0644)
		if err != nil {
			return err
		}
		fmt.Printf("Wrote default config to %s\n", *flagConfig)
		os.Exit(0)
	}

	if configMissingErr == nil {
		log.Printf("Read config from %s\n", *flagConfig)
	} else {
		log.Printf("Config at %s does not exist\n", *flagConfig)
	}

	cf := flag.Lookup("cache")
	ep, err := expandPath(cf.Value.String())
	if err != nil {
		return err
	}
	cf.Value.Set(ep)

	log.Printf("Effective configuration:\n")
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if slices.Contains(skip, f.Name) && f.Name != "config" {
			return
		}
		log.Printf("\t--%s=%#v\n", f.Name, f.Value.String())
	})
	return nil
}
