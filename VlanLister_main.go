package main

/*
#### ##     ## ########   #######  ########  ########  ######
 ##  ###   ### ##     ## ##     ## ##     ##    ##    ##    ##
 ##  #### #### ##     ## ##     ## ##     ##    ##    ##
 ##  ## ### ## ########  ##     ## ########     ##     ######
 ##  ##     ## ##        ##     ## ##   ##      ##          ##
 ##  ##     ## ##        ##     ## ##    ##     ##    ##    ##
#### ##     ## ##         #######  ##     ##    ##     ######
*/

import (
	"fmt"
	"log"
	"os"
	"path"
	"sort"

	godotenv "github.com/joho/godotenv"
	pflag "github.com/spf13/pflag"
	envordef "gitlab.com/rbrt-weiler/go-module-envordef"
	xmcnbiclient "gitlab.com/rbrt-weiler/go-module-xmcnbiclient"
)

/*
 ######   #######  ##    ##  ######  ########    ###    ##    ## ########  ######
##    ## ##     ## ###   ## ##    ##    ##      ## ##   ###   ##    ##    ##    ##
##       ##     ## ####  ## ##          ##     ##   ##  ####  ##    ##    ##
##       ##     ## ## ## ##  ######     ##    ##     ## ## ## ##    ##     ######
##       ##     ## ##  ####       ##    ##    ######### ##  ####    ##          ##
##    ## ##     ## ##   ### ##    ##    ##    ##     ## ##   ###    ##    ##    ##
 ######   #######  ##    ##  ######     ##    ##     ## ##    ##    ##     ######
*/

const (
	toolName    string = "VlanLister.go"
	toolVersion string = "2.2.0-dev"
	toolID      string = toolName + "/" + toolVersion
	toolURL     string = "https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go"
	envFileName string = ".xmcenv"
)

/*
##     ##    ###    ########   ######
##     ##   ## ##   ##     ## ##    ##
##     ##  ##   ##  ##     ## ##
##     ## ##     ## ########   ######
 ##   ##  ######### ##   ##         ##
  ## ##   ##     ## ##    ##  ##    ##
   ###    ##     ## ##     ##  ######
*/

var (
	// The actual client that connects to XMC
	xmcClient xmcnbiclient.NBIClient
	// The usable instance of app configuration
	config appConfig
	// Logging-formatted stderr
	stdErr = log.New(os.Stderr, "", log.LstdFlags)
)

/*
######## ##     ## ##    ##  ######   ######
##       ##     ## ###   ## ##    ## ##    ##
##       ##     ## ####  ## ##       ##
######   ##     ## ## ## ## ##        ######
##       ##     ## ##  #### ##             ##
##       ##     ## ##   ### ##    ## ##    ##
##        #######  ##    ##  ######   ######
*/

// Parses the CLI options and arguments into app config
func parseCLIOptions() {
	pflag.StringVarP(&config.XMCHost, "host", "h", envordef.StringVal("XMCHOST", ""), "XMC Hostname / IP")
	pflag.UintVar(&config.XMCPort, "port", envordef.UintVal("XMCPORT", 8443), "HTTP port where XMC is listening")
	pflag.StringVar(&config.XMCPath, "path", envordef.StringVal("XMCPATH", ""), "Path where XMC is reachable")
	pflag.UintVar(&config.HTTPTimeout, "timeout", envordef.UintVal("XMCTIMEOUT", 5), "Timeout for HTTP(S) connections")
	pflag.BoolVar(&config.NoHTTPS, "nohttps", envordef.BoolVal("XMCNOHTTPS", false), "Use HTTP instead of HTTPS")
	pflag.BoolVar(&config.InsecureHTTPS, "insecurehttps", envordef.BoolVal("XMCINSECUREHTTPS", false), "Do not validate HTTPS certificates")
	pflag.StringVarP(&config.XMCUserID, "userid", "u", envordef.StringVal("XMCUSERID", ""), "Client ID (OAuth) or username (Basic Auth) for authentication")
	pflag.StringVarP(&config.XMCSecret, "secret", "s", envordef.StringVal("XMCSECRET", ""), "Client Secret (OAuth) or password (Basic Auth) for authentication")
	pflag.BoolVar(&config.BasicAuth, "basicauth", envordef.BoolVal("XMCBASICAUTH", false), "Use HTTP Basic Auth instead of OAuth")
	pflag.BoolVar(&config.NoRefresh, "norefresh", envordef.BoolVal("XMCNOREFRESH", false), "Do not refresh (rediscover) devices")
	pflag.UintVar(&config.RefreshInterval, "refreshinterval", envordef.UintVal("XMCREFRESHINTERVAL", 5), "Seconds to wait between triggering each refresh")
	pflag.UintVar(&config.RefreshWait, "refreshwait", envordef.UintVal("XMCREFRESHWAIT", 15), "Minutes to wait after refreshing devices")
	pflag.BoolVar(&config.IncludeDown, "includedown", envordef.BoolVal("XMCINCLUDEDOWN", false), "Include inactive devices in result")
	pflag.BoolVar(&config.NoColor, "nocolor", envordef.BoolVal("XMCNOCOLOR", false), "Do not colorize output (Excel)")
	pflag.BoolVar(&config.CompressOutput, "compress-output", envordef.BoolVal("XMCCOMPRESSOUTPUT", false), "Compress output using gzip")
	pflag.Var(&config.Outfile, "outfile", "File to write data to")
	pflag.BoolVar(&config.PrintVersion, "version", false, "Print version information and exit")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", toolID)
		fmt.Fprintf(os.Stderr, "%s\n", toolURL)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "This tool fetches a list of active devices (state = up) from XMC. It then\n")
		fmt.Fprintf(os.Stderr, "retrieves a list of all VLANs and VLAN to port associations, which is\n")
		fmt.Fprintf(os.Stderr, "written to outfile.\n")
		fmt.Fprintf(os.Stderr, "Optionally (but recommended), all found devices are refreshed before\n")
		fmt.Fprintf(os.Stderr, "retrieving the VLAN data.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", path.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Available options:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "It is required to provide at least one outfile. File types are determined\n")
		fmt.Fprintf(os.Stderr, "by the prefix FILETYPE: or the suffix .FILETYPE. Prefixes take priority\n")
		fmt.Fprintf(os.Stderr, "over suffixes. Valid FILETYPEs are:\n")
		fmt.Fprintf(os.Stderr, "  csv     -->  writes CSV data to the given file\n")
		fmt.Fprintf(os.Stderr, "  json    -->  writes JSON data to the given file\n")
		fmt.Fprintf(os.Stderr, "  stdout  -->  prints CSV data to stdout\n")
		fmt.Fprintf(os.Stderr, "  xlsx    -->  writes XLSX data to the given file\n")
		fmt.Fprintf(os.Stderr, "When using stdout, you should remove all stderr output (2>/dev/null).\n")
		fmt.Fprintf(os.Stderr, "The additional suffix .gz can be used to trigger compression.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Nearly all options that take a value can be set via environment variables:\n")
		fmt.Fprintf(os.Stderr, "  XMCHOST             -->  --host\n")
		fmt.Fprintf(os.Stderr, "  XMCPORT             -->  --port\n")
		fmt.Fprintf(os.Stderr, "  XMCPATH             -->  --path\n")
		fmt.Fprintf(os.Stderr, "  XMCTIMEOUT          -->  --timeout\n")
		fmt.Fprintf(os.Stderr, "  XMCNOHTTPS          -->  --nohttps\n")
		fmt.Fprintf(os.Stderr, "  XMCINSECUREHTTPS    -->  --insecurehttps\n")
		fmt.Fprintf(os.Stderr, "  XMCUSERID           -->  --userid\n")
		fmt.Fprintf(os.Stderr, "  XMCSECRET           -->  --secret\n")
		fmt.Fprintf(os.Stderr, "  XMCBASICAUTH        -->  --basicauth\n")
		fmt.Fprintf(os.Stderr, "  XMCNOREFRESH        -->  --norefresh\n")
		fmt.Fprintf(os.Stderr, "  XMCREFRESHINTERVAL  -->  --refreshinterval\n")
		fmt.Fprintf(os.Stderr, "  XMCREFRESHWAIT      -->  --refreshwait\n")
		fmt.Fprintf(os.Stderr, "  XMCINCLUDEDOWN      -->  --includedown\n")
		fmt.Fprintf(os.Stderr, "  XMCNOCOLOR          -->  --nocolor\n")
		fmt.Fprintf(os.Stderr, "  XMCCOMPRESSOUTPUT   -->  --compress-output\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Environment variables can also be configured via a file called %s,\n", envFileName)
		fmt.Fprintf(os.Stderr, "located in the current directory or in the home directory of the current\n")
		fmt.Fprintf(os.Stderr, "user.\n")
	}
	pflag.Parse()
}

/*
##     ##    ###    #### ##    ##
###   ###   ## ##    ##  ###   ##
#### ####  ##   ##   ##  ####  ##
## ### ## ##     ##  ##  ## ## ##
##     ## #########  ##  ##  ####
##     ## ##     ##  ##  ##   ###
##     ## ##     ## #### ##    ##
*/

// init loads environment files if available.
func init() {
	// if envFileName exists in the current directory, load it
	localEnvFile := fmt.Sprintf("./%s", envFileName)
	if _, localEnvErr := os.Stat(localEnvFile); localEnvErr == nil {
		if loadErr := godotenv.Load(localEnvFile); loadErr != nil {
			stdErr.Printf("Could not load env file <%s>: %s", localEnvFile, loadErr)
		}
	}

	// if envFileName exists in the user's home directory, load it
	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		homeEnvFile := fmt.Sprintf("%s/%s", homeDir, ".xmcenv")
		if _, homeEnvErr := os.Stat(homeEnvFile); homeEnvErr == nil {
			if loadErr := godotenv.Load(homeEnvFile); loadErr != nil {
				stdErr.Printf("Could not load env file <%s>: %s", homeEnvFile, loadErr)
			}
		}
	}
}

func main() {
	parseCLIOptions()

	if config.PrintVersion {
		fmt.Println(toolID)
		os.Exit(0)
	}
	if config.XMCHost == "" {
		stdErr.Fatal("host is required.")
	}
	if len(config.Outfile) <= 0 {
		stdErr.Fatal("outfile is required.")
	}

	initializeClient(&xmcClient)

	upDevices, downDevices := discoverManagedDevices(&xmcClient)

	var rediscoveredDevices []string
	if config.NoRefresh {
		rediscoveredDevices = upDevices
	} else {
		rediscoveredDevices = rediscoverDevices(&xmcClient, upDevices)
	}
	if config.IncludeDown {
		rediscoveredDevices = append(rediscoveredDevices, downDevices...)
	}
	sort.Strings(rediscoveredDevices)

	queryResults := []singleDevice{}
	for _, deviceIP := range rediscoveredDevices {
		deviceResult, deviceErr := queryDevice(&xmcClient, deviceIP)
		if deviceErr != nil {
			stdErr.Println(deviceErr)
			continue
		}
		queryResults = append(queryResults, deviceResult)
	}
	sort.Slice(queryResults, func(i, j int) bool { return queryResults[i].ID < queryResults[j].ID })

	var writeRows uint
	var writeErr error
	for _, outfile := range config.Outfile {
		writeRows, writeErr = writeResults(outfile, devicesWrapper{queryResults})
		if writeErr != nil {
			stdErr.Println(writeErr)
		} else {
			stdErr.Printf("%d rows written to <%s>.\n", writeRows, outfile)
		}
	}
}
