# XMC NBI VlanLister (Go)

VlanLister uses the GraphQL-based API provided by the Northbound Interface (NBI) of [Extreme Management Center](https://www.extremenetworks.com/product/extreme-management-center/) (XMC; formerly known as NetSight) to generate a CSV file that lists all VLANs that are configured on all up switches along with port associations. The tool is intended to be used during audits and for general network documentation.

## Branches

This project uses two defined branches:

* `master` is the primary development branch. Code within `master` may be broken at any time.
* `stable` is reserved for code that compiles without errors and is tested. Track `stable` if you just want to use the software.

Other branches, for example for developing specific features, may be created and deleted at any time.

## Dependencies

VlanLister uses the modules [godotenv](https://github.com/joho/godotenv), [envordef](https://gitlab.com/rbrt-weiler/go-module-envordef) and [xmcnbiclient](https://gitlab.com/rbrt-weiler/go-module-xmcnbiclient). Execute...

1. `go get -u github.com/joho/godotenv`
1. `go get -u gitlab.com/rbrt-weiler/go-module-envordef`
1. `go get -u gitlab.com/rbrt-weiler/go-module-xmcnbiclient`

...before running or compiling VlanLister. All other dependencies are included in a standard Go installation.

## Compiling

Use `go run VlanLister.go` to run the tool directly or `go build VlanLister.go` to compile a binary.

Tested with [go1.13](https://golang.org/doc/go1.13).

## Usage

`VlanLister -h`:

<pre>
Available options:
  -basicauth
    	Use HTTP Basic Auth instead of OAuth
  -finishwait uint
    	Minutes to wait after refreshing devices (default 15)
  -host string
    	XMC Hostname / IP
  -includedown
    	Include inactive devices in result
  -insecurehttps
    	Do not validate HTTPS certificates
  -nohttps
    	Use HTTP instead of HTTPS
  -outfile string
    	File to write CSV data to
  -path string
    	Path where XMC is reachable
  -port uint
    	HTTP port where XMC is listening (default 8443)
  -refreshdevices
    	Refresh (rediscover) devices (default true)
  -refreshwait uint
    	Seconds to wait between triggering each refresh (default 5)
  -secret string
    	Client Secret (OAuth) or password (Basic Auth) for authentication
  -timeout uint
    	Timeout for HTTP(S) connections (default 5)
  -userid string
    	Client ID (OAuth) or username (Basic Auth) for authentication
  -version
    	Print version information and exit

All options that take a value can be set via environment variables:
  XMCHOST            -->  -host
  XMCPORT            -->  -port
  XMCPATH            -->  -path
  XMCTIMEOUT         -->  -timeout
  XMCNOHTTPS         -->  -nohttps
  XMCINSECUREHTTPS   -->  -insecurehttps
  XMCUSERID          -->  -userid
  XMCSECRET          -->  -secret
  XMCBASICAUTH       -->  -basicauth
  XMCREFRESHDEVICES  -->  -refreshdevices
  XMCREFRESHWAIT     -->  -refreshwait
  XMCFINISHWAIT      -->  -finishwait
  XMCINCLUDEDOWN     -->  -includedown
  XMCOUTFILE         -->  -outfile
</pre>

## Authentication

VlanLister supports two methods of authentication: OAuth2 and HTTP Basic Auth.

* OAuth2: To use OAuth2, provide the parameters `userid` and `secret`. VlanLister will attempt to obtain a OAuth2 token from XMC with the supplied credentials and, if successful, submit only that token with each API request as part of the HTTP header.
* HTTP Basic Auth: To use HTTP Basic Auth, provide the parameters `userid` and `secret` as well as `basicauth`. VlanLister will transmit the supplied credentials with each API request as part of the HTTP request header.

As all interactions between VlanLister and XMC are secured with HTTPS by default both methods should be safe for transmission over networks. It is strongly recommended to use OAuth2 though. Should the credentials ever be compromised, for example when using them on the CLI on a shared workstation, remediation will be much easier with OAuth2. When using unencrypted HTTP transfer (`nohttps`), Basic Auth should never be used.

In order to use OAuth2 you will need to create a Client API Access client. To create such a client, visit the _Administration_ -> _Client API Access_ tab within XMC and click on _Add_. Make sure to note the returned credentials, as they will never be shown again.

## Authorization

Any user or API client who wants to access the Northbound Interface needs the appropriate access rights. In general, checking the full _Northbound API_ section within rights management will suffice. Depending on the use case, it may be feasible to go into detail and restrict the rights to the bare minimum required.

For API clients (OAuth2) the rights are defined when creating an API client and can later be adjusted in the same tab. For regular users (HTTP Basic Auth) the rights are managed via _Authorization Groups_ found in the _Administration_ -> _Users_ tab within XMC.

## Source

The original project is [hosted at GitLab](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go), with a [copy over at GitHub](https://github.com/rbrt-weiler/xmc-nbi-vlanlister-go) for the folks over there. Additionally, there is a project at GitLab which [collects all available clients](https://gitlab.com/rbrt-weiler/xmc-nbi-clients).
