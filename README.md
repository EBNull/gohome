# Gohome

A `go/links` daemon for your local machine.

Unlike other golink implementations `gohome` is designed to run
on a single user's mac or linux machine. By default DNS and IP
settings are configured automatically and deconfigured on exit.
Root privileges are required for this functionality.

`gohome` can continually download and refresh golinks from a central
server. This allows integration with an existing link service
that may not always be accessible. In this configuration
`gohome` works as a local cache to ensure no workflow disruptions
in case of an outage.

By default `gohome` configures the URL `http://gohome` to avoid
interfering with an existing `go` domain. Pass `--hostname go`
to allow `http://go` to resolve to `gohome` instead.

There is no web interface to add or edit golinks. Instead you can
add them [manually](#creating-links) or periodically [pull links](#chained-links)
from another golinks source.

## Quick Start

### Local

```shell
go install github.com/ebnull/gohome@latest
```
```
gohome
open http://gohome/
```

### Homebrew

Via a homebrew [tap](https://github.com/EBNull/homebrew-gohome):

```shell
brew install ebnull/gohome/gohome
```

```
gohome --hostname go
open http://gohome/
```

Homebrew can also install `gohome` as a service running as root.
In that case, the default paths are `/opt/homebrew/etc/gohome.flags`
and `/opt/homebrew/var/cache/golink_cache.json`. You can find
log output at `/opt/homebrew/var/gohome-stderr.txt`.

To write a default configuration to that path, run

```
gohome --config /opt/homebrew/etc/gohome.flags --write-config
```

### Container

Using a [container image](https://github.com/EBNull/gohome/pkgs/container/gohome)
for network deployment:

```shell
docker run ghcr.io/ebnull/gohome:latest
```

## Creating Links

Creating new links is not supported in the web interface.

To add links manually edit the `--cache` path, by default
`~/.cache/golink_cache.json`.

Note that as of `v0.0.12` the only required fields are `display` and `destination`.
`source` is the [*canonicalized* link](https://github.com/search?q=repo%3AEBNull%2Fgohome+path%3Alink.go+canonicalize&type=code)
while `display` is the *visible* link.

There is no ownership management; the field `owner` may be set arbitrarily.

```json
[
  {
    "display": "Foo-Bar",
    "source": "foobar",
    "destination": "http://example.org",
    "owner": "Me"
  }
]
```

## Pulling Links

Links can be pulled from a remote source given by `--remote`. This is the
full URL to a remote JSON file whose contents are a list of links.
This file will be updated and merged with already known links every `--interval` (default: `15m`).

Updated links will be written to the cache file specified by `--cache`.

If a web URL exists to add a new link on the upstream server you can
enable `golinks` to add UI links to it by setting `--add-link-url`.

## Chained Links

If a `--chain` url is specified and a link is not known, `golinks`
will redirect to this URL (substituting the the link target for `%s`).
This is designed for the case of new links being added that have not
yet been cached.

## Automatic loopback configuration

By default (controlled by `--auto`), on a linux or mac machine, `gohome`
will add a local loopback IP address to the loopback interface. This
IP address is given as the IP to `--bind`, which by default is set
to `127.0.0.53:80`. It will also edit `/etc/hosts` to enable resolution
of the domain name given by `--hostname` (default `gohome`).
These changes are reverted when `golinks` exits.

In effect, this allows your local machine to immediately resolve
`http://gohome` and URLs without conflicting with other services
running locally.

## Binding

To change the bind address use `--bind`. You'll probably also
want to disable autoconfiguration when using this flag the bind
address is also used as the loopback address.

By default `gohome` expects to be able to bind to port 80, which means
it needs at least the [`CAP_NET_BIND_SERVICE`](https://man7.org/linux/man-pages/man7/capabilities.7.html)
capability on linux or root permissions on Mac.

To run without elevated permissions choose a non-default port and
disable automatic loopback configuration:

```shell
gohome --auto=false --bind 127.0.0.1:8080
```

## Network-wide golinks

You can also use `gohome` as a server for your network by
specifying a bind port without an IP. See golang's
[`net.Listen`](https://pkg.go.dev/net#Listen) documentation for
more on how `--bind` is parsed.


```shell
gohome --auto=false --bind :8080
```

## Per-user options

`golinks` supports storing options for each user in cookies.

You can choose to enable `no-redirect`, which will show an
interstitial page to preview the link destination.

Alternately, you can enable `no-chain`, which will show an
error page locally instead of redirecting to the configured
remote golink provider.

There is a plain web-ui hosted at the root of the server
for configuring these options.

## Known Issues

On mac, if you use the default configuration, you'll get a firewall
prompt asking you to allow `gohome` to accept connections even though
the `127.0.0.53` local bind is not accesbile from outside the machine.
Even if you click deny here golinks will still be accessible locally.

## Run-time configuration

`gohome` accepts flags from the command line, environment variables,
and a configuration file (in that order). `--config` sets the path
to the configuration file. The format of this file is defined by
[`ff.PlainParser`](https://pkg.go.dev/github.com/peterbourgon/ff/v3#PlainParser)
and by default is located at `~/.config/gohome.flags`.

A default configuration file can be written with `--write-config`.

### Example configuration file

```
# Configuration file for gohome
#
# https://github.com/EBNull/gohome
#
# Syntax:
#   key value
#
# For syntax details, see https://pkg.go.dev/github.com/peterbourgon/ff/v3#PlainParser
#

# The url to add a new golink. If set a link will be displayed when a golink is not found.
#add-link-url

# Automatically alias the bind IP address to the loopback interface
auto true

# The IP and port to bind to
bind 127.0.0.53:80

# The filename to load cached golinks from
cache ~/.cache/golink_cache.json

# The remote URL to chain redirect to (if link not found in local cache)
#chain

# Specifies the location of the hostfile to edit for --auto mode
hostfile /etc/hosts

# The hostname to add to /etc/hosts for --auto mode (resolvable to the bind address)
hostname gohome

# The periodic interval for downloading new golinks
interval 15m0s

# Specifies the loopback adapter interface for --auto mode
loopback-interface lo

# The remote URL to update golinks from
#remote
```

## Build-time configuration

You can set the defaults for some options by passing flags to the
go linker. See [build.go](build/build.go) for the list of overrideable
flags or the [linker documentation](https://pkg.go.dev/cmd/link) for details.


For example:

```shell
go build -ldflags="-X 'github.com/ebnull/gohome/build.DefaultRemote=http://example.org/links.json' -X 'github.com/ebnull/gohome/build.DefaultChain=http://example.org/goto/%s' -X 'github.com/ebnull/gohome/build.DefaultAddLinkUrl=http://example.org/new-link'"
```

You could also use [`goreleaser`](https://goreleaser.com) to make a snapshot build with a [custom configuration](.goreleaser.yaml#L18-L27):

```
goreleaser build --snapshot --clean
```

## License

The standard [MIT License](LICENSE.txt).
