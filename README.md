# Gohome

An implementation of `go/links` for your local machine.

## Quick Start

### Container

Using a [container image](https://github.com/EBNull/gohome/pkgs/container/gohome):

```shell
docker run ghcr.io/ebnull/gohome:latest
```

### Homebrew

Using homebrew (see [tap](https://github.com/EBNull/homebrew-gohome)):

```shell
brew install ebnull/gohome/gohome
```

## Creating Links

Creating new links is not supported in the web interface.

To add links manually edit the `--cache` path, by default
`"~/.cache/golink_cache.json`.

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

## Chained Links

If a `--chain` url is specified and a link is not known, `golinks`
will redirect to this URL (substituting the the link target for `%s`).
This is designed for the case of new links being added that have not
yet been cached.

## Automatic loopback & Binding

By default (controlled by `--auto`), on a linux or mac machine, `gohome`
will add a local loopback IP address to the loopback interface. This
IP address is given as the IP to `--bind`, which by default is set
to `127.0.0.53:80`. It will also edit `/etc/hosts` to enable resolution
of the domain name given by `--hostname` (default `gohome`).

In effect, this allows your local machine to immediately resolve 
`http://gohome` and URLs without conflicting with other services
running locally.

## Binding

To change the bind address use `--bind`. You'lll probably also
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
intersital page to preview the link destination.

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

The standard [MIT License](LICENSE.txt). It would still be nice to give acknowledgements and a link to this repository, but you are not required to do so.
