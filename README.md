# XMC NBI VlanLister (Go)

VlanLister uses the GraphQL-based API provided by the Northbound Interface (NBI) of [Extreme Management Center](https://www.extremenetworks.com/product/extreme-management-center/) (XMC; formerly known as NetSight) to generate a CSV file that lists all VLANs that are configured on all up switches along with port associations. The tool is intended to be used during audits and for general network documentation.

## Branches

This project uses two defined branches:

* `master` is the primary development branch. Code within `master` may be broken at any time.
* `stable` is reserved for code that compiles without errors and is tested. Track `stable` if you just want to use the software.

Other branches, for example for developing specific features, may be created and deleted at any time.

## Dependencies

VlanLister uses the [module xmcnbiclient](https://gitlab.com/rbrt-weiler/go-module-xmcnbiclient). This module has to be installed with `go get gitlab.com/rbrt-weiler/go-module-xmcnbiclient` or updated with `go get -u gitlab.com/rbrt-weiler/go-module-xmcnbiclient` before running or compiling VlanLister. All other dependencies are included in a standard Go installation.

## Compiling

Use `go run VlanLister.go` to run the tool directly or `go build VlanLister.go` to compile a binary.

Tested with [go1.13](https://golang.org/doc/go1.13).

## Usage

`VlanLister -h`:

<pre>
Available options:
  -clientid string
        Client ID for OAuth
  -clientsecret string
        Client Secret for OAuth
  -host string
        XMC Hostname / IP (default "localhost")
  -includedown
        Include inactive devices in result
  -insecurehttps
        Do not validate HTTPS certificates
  -nohttps
        Use HTTP instead of HTTPS
  -operationwait uint
        Minutes to wait after refreshing devices (default 15)
  -outfile string
        File to write CSV data to
  -password string
        Password for HTTP Basic Auth
  -refreshdevices
        Refresh (rediscover) devices (default true)
  -refreshwait uint
        Seconds to wait between triggering each refresh (default 5)
  -timeout uint
        Timeout for HTTP(S) connections (default 5)
  -username string
        Username for HTTP Basic Auth (default "admin")
  -version
        Print version information and exit

OAuth will be preferred over username/password.
</pre>

## Source

The original project is [hosted at GitLab](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go), with a [copy over at GitHub](https://github.com/rbrt-weiler/xmc-nbi-vlanlister-go) for the folks over there. Additionally, there is a project at GitLab which [collects all available clients](https://gitlab.com/rbrt-weiler/xmc-nbi-clients).
