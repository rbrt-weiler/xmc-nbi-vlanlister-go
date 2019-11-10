# XMC API Clients - Go - VlanLister

Generates a CSV file that lists all VLANs that are configured on all up switches along with port associations.

## Compiling

`go build VlanLister.go`

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
