# XMC NBI VlanLister (Go)

VlanLister uses the GraphQL-based API provided by the Northbound Interface (NBI) of [Extreme Management Center (XMC)](https://www.extremenetworks.com/product/extreme-management-center/) to generate a CSV file that lists all VLANs that are configured on all up switches along with port associations. The tool is intended to be used during audits and for general network documentation.

## Branches

This project uses two defined branches:

  * `master` is the primary development branch. Code within `master` may be broken at any time.
  * `stable` is reserved for code that compiles without errors and is tested. Track `stable` if you just want to use the software.

Other branches, for example for developing specific features, may be created and deleted at any time.

## Compiling

Use `go run VlanLister.go` to run the tool directly or `go build VlanLister.go` to compile a binary.

Tested with go1.11 and go1.13.

## Usage

`VlanLister -h`:

<pre>
  -host string
    	XMC Hostname / IP (default "localhost")
  -httptimeout uint
    	Timeout for HTTP(S) connections (default 5)
  -insecurehttps
    	Do not validate HTTPS certificates
  -mutdevices
    	Mutate (rediscover) devices (default true)
  -mutpause uint
    	Minutes to wait after mutating devices (default 15)
  -mutwait uint
    	Seconds to wait between mutations (default 5)
  -outfile string
    	File to write CSV data to
  -password string
    	Password for HTTP auth
  -username string
    	Username for HTTP auth (default "admin")
</pre>

## Source

The original project is [hosted at GitLab](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go), with a [copy over at GitHub](https://github.com/rbrt-weiler/xmc-nbi-vlanlister-go) for the folks over there. Additionally, there is a project at GitLab which [collects all available clients](https://gitlab.com/rbrt-weiler/xmc-nbi-clients).
