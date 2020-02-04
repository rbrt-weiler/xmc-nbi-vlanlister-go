/*
Copyright (c) 2019,2020 Robert Weiler <https://robert.weiler.one/>
Copyright (c) 2019 BELL Computer-Netzwerke GmbH <https://www.bell.de/>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

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
	toolName           string = "VlanLister.go"
	toolVersion        string = "2.0.0-dev"
	httpUserAgent      string = toolName + "/" + toolVersion
	gqlDeviceListQuery string = `
		query {
			network {
				devices {
					up
					ip
				}
			}
		}
	`
	gqlMutationQuery string = `
		mutation {
			network {
				rediscoverDevices(input: {devices: [{ipAddress: "%s"}]}) {
					status
					message
				}
			}
		}
  	`
	gqlDeviceDataQuery string = `
		query {
			network {
				device(ip: "%s") {
					id
					up
					sysName
					sysLocation
					nickName
					baseMac
					ip
					entityData {
						allPorts {
							ifIndex
							ifName
							ifOperStatus
							vlanList
						}
					}
				}
				deviceVlans(ip: "%s") {
					vid
				}
			}
		}
	`
)

/*
######## ##    ## ########  ########  ######
   ##     ##  ##  ##     ## ##       ##    ##
   ##      ####   ##     ## ##       ##
   ##       ##    ########  ######    ######
   ##       ##    ##        ##             ##
   ##       ##    ##        ##       ##    ##
   ##       ##    ##        ########  ######
*/

// Stores configuration used throughout the app
type appConfig struct {
	httpHost          string
	httpTimeoutSecs   uint
	noHTTPS           bool
	insecureHTTPS     bool
	username          string
	password          string
	clientID          string
	clientSecret      string
	refreshDevices    bool
	refreshWaitSecs   uint
	operationWaitMins uint
	includeDown       bool
	outfile           string
	printVersion      bool
}

// Used for parsing the list of devices returned by XMC
type deviceList struct {
	Data struct {
		Network struct {
			Devices []struct {
				Up bool   `json:"up"`
				IP string `json:"ip"`
			} `json:"devices"`
		} `json:"network"`
	} `json:"data"`
}

// Used to parse the result of each single mutation (device refresh)
type mutationMessage struct {
	Data struct {
		Network struct {
			RediscoverDevices struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"rediscoverDevices"`
		} `json:"network"`
	} `json:"data"`
}

// Used to store the result of ports and VLANs for each single device
type deviceData struct {
	Data struct {
		Network struct {
			Device struct {
				ID          int    `json:"id"`
				Up          bool   `json:"up"`
				SysName     string `json:"sysName"`
				SysLocation string `json:"sysLocation"`
				NickName    string `json:"nickName"`
				BaseMac     string `json:"baseMac"`
				IP          string `json:"ip"`
				EntityData  struct {
					AllPorts []struct {
						IfIndex      int      `json:"ifIndex"`
						IfName       string   `json:"ifName"`
						IfOperStatus string   `json:"ifOperStatus"`
						VlanList     []string `json:"vlanList"`
					} `json:"allPorts"`
				} `json:"entityData"`
			} `json:"device"`
			DeviceVlans []struct {
				Vid int `json:"vid"`
			} `json:"deviceVlans"`
		} `json:"network"`
	} `json:"data"`
}

// Used to store the values for each row that is written to outfile
type resultSet struct {
	ID          int
	BaseMac     string
	IP          string
	SysUpDown   string
	SysName     string
	SysLocation string
	IfName      string
	IfStatus    string
	Untagged    []string
	Tagged      []string
}

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
	client xmcnbiclient.NBIClient
	// The usable instance of app configuration
	config appConfig
	// Logging-formatted stdout
	stdOut = log.New(os.Stdout, "", log.LstdFlags)
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
	flag.StringVar(&config.httpHost, "host", "localhost", "XMC Hostname / IP")
	flag.UintVar(&config.httpTimeoutSecs, "timeout", 5, "Timeout for HTTP(S) connections")
	flag.BoolVar(&config.noHTTPS, "nohttps", false, "Use HTTP instead of HTTPS")
	flag.BoolVar(&config.insecureHTTPS, "insecurehttps", false, "Do not validate HTTPS certificates")
	flag.StringVar(&config.username, "username", "admin", "Username for HTTP Basic Auth")
	flag.StringVar(&config.password, "password", "", "Password for HTTP Basic Auth")
	flag.StringVar(&config.clientID, "clientid", "", "Client ID for OAuth")
	flag.StringVar(&config.clientSecret, "clientsecret", "", "Client Secret for OAuth")
	flag.BoolVar(&config.refreshDevices, "refreshdevices", true, "Refresh (rediscover) devices")
	flag.UintVar(&config.refreshWaitSecs, "refreshwait", 5, "Seconds to wait between triggering each refresh")
	flag.UintVar(&config.operationWaitMins, "operationwait", 15, "Minutes to wait after refreshing devices")
	flag.BoolVar(&config.includeDown, "includedown", false, "Include inactive devices in result")
	flag.StringVar(&config.outfile, "outfile", "", "File to write CSV data to")
	flag.BoolVar(&config.printVersion, "version", false, "Print version information and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "This tool fetches a list of active devices (state = up) from XMC. It then\n")
		fmt.Fprintf(os.Stderr, "retrieves a list of all VLANs and VLAN to port associations, which is\n")
		fmt.Fprintf(os.Stderr, "written to outfile.\n")
		fmt.Fprintf(os.Stderr, "Optionally (but recommended), all found devices are refreshed before\n")
		fmt.Fprintf(os.Stderr, "retrieving the VLAN data.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", path.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Available options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "OAuth will be preferred over username/password.\n")
	}
	flag.Parse()
}

// Initializes the actual XMC client
func initializeClient() {
	client = xmcnbiclient.New(config.httpHost)
	client.SetUserAgent(httpUserAgent)
	client.UseHTTPS()
	if config.noHTTPS {
		client.UseHTTP()
	}
	client.UseBasicAuth(config.username, config.password)
	if config.clientID != "" && config.clientSecret != "" {
		client.UseOAuth(config.clientID, config.clientSecret)
	}
	client.UseSecureHTTPS()
	if config.insecureHTTPS {
		client.UseInsecureHTTPS()
	}
	timeoutErr := client.SetTimeout(config.httpTimeoutSecs)
	if timeoutErr != nil {
		stdErr.Fatalf("Could not set HTTP timeout: %s\n", timeoutErr)
	}
}

// Refreshes the OAuth token if it is to expire soon
func proactiveTokenRefresh() {
	if client.Authentication.Type == xmcnbiclient.AuthTypeOAuth {
		if client.AccessToken.ExpiresSoon(config.httpTimeoutSecs + 1) {
			go client.RetrieveOAuthToken()
		}
	}
}

// Fetches the complete list of managed devices from XMC
func discoverManagedDevices() ([]string, []string) {
	stdOut.Println("Discovering managed devices...")

	body, bodyErr := client.QueryAPI(gqlDeviceListQuery)
	if bodyErr != nil {
		stdErr.Fatalf("Could not fetch device list: %s\n", bodyErr)
	}
	proactiveTokenRefresh()

	devices := deviceList{}
	jsonErr := json.Unmarshal(body, &devices)
	if jsonErr != nil {
		stdOut.Fatalf("Could not decode JSON: %s\n", jsonErr)
	}

	var upDevices []string
	var downDevices []string
	for _, d := range devices.Data.Network.Devices {
		if d.Up {
			upDevices = append(upDevices, d.IP)
		} else {
			downDevices = append(downDevices, d.IP)
		}
	}
	sort.Strings(upDevices)
	stdOut.Println("Finished discovering managed devices.")

	return upDevices, downDevices
}

// Triggers a rediscover for a list of devices
func rediscoverDevices(ipList []string) []string {
	var rediscoveredDevices []string
	for _, deviceIP := range ipList {
		body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlMutationQuery, deviceIP))
		if bodyErr != nil {
			stdErr.Printf("Could not mutate device %s: %s\n", deviceIP, bodyErr)
			continue
		}
		proactiveTokenRefresh()

		mutation := mutationMessage{}
		jsonErr := json.Unmarshal(body, &mutation)
		if jsonErr != nil {
			stdErr.Printf("Could not decode JSON: %s\n", jsonErr)
			continue
		}

		if mutation.Data.Network.RediscoverDevices.Status == "SUCCESS" {
			stdOut.Printf("Successfully triggered rediscover for %s.\n", deviceIP)
			rediscoveredDevices = append(rediscoveredDevices, deviceIP)
		} else {
			stdErr.Printf("Rediscover for %s failed: %s\n", deviceIP, mutation.Data.Network.RediscoverDevices.Message)
		}

		stdOut.Printf("Waiting for %d second(s)...\n", config.refreshWaitSecs)
		time.Sleep(time.Second * time.Duration(config.refreshWaitSecs))
	}
	for i := config.operationWaitMins; i > 0; i-- {
		proactiveTokenRefresh()
		stdOut.Printf("Waiting for %d minute(s) to finish rediscover...\n", i)
		time.Sleep(time.Minute * time.Duration(1))
	}
	return rediscoveredDevices
}

// Fetches the detailed data for a single device from XMC
func queryDevice(deviceIP string) ([]resultSet, error) {
	var deviceResult []resultSet

	body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlDeviceDataQuery, deviceIP, deviceIP))
	if bodyErr != nil {
		return deviceResult, fmt.Errorf("Could not query device %s: %s", deviceIP, bodyErr)
	}
	proactiveTokenRefresh()

	jsonData := deviceData{}
	jsonErr := json.Unmarshal(body, &jsonData)
	if jsonErr != nil {
		return deviceResult, fmt.Errorf("Could not decode JSON: %s", jsonErr)
	}

	device := jsonData.Data.Network.Device
	vlans := jsonData.Data.Network.DeviceVlans
	ports := jsonData.Data.Network.Device.EntityData.AllPorts

	stdOut.Printf("Fetched data for %s: Got %d VLANs and %d ports.\n", device.IP, len(vlans), len(ports))

	systemResult := resultSet{}
	systemResult.ID = device.ID
	systemResult.BaseMac = device.BaseMac
	systemResult.IP = device.IP
	systemResult.SysUpDown = "down"
	if device.Up {
		systemResult.SysUpDown = "up"
	}
	systemResult.SysName = device.SysName
	systemResult.SysLocation = device.SysLocation
	systemResult.IfName = "SYSTEM"
	systemResult.IfStatus = "N/A"
	for _, vlan := range vlans {
		systemResult.Tagged = append(systemResult.Tagged, strconv.Itoa(vlan.Vid))
	}
	deviceResult = append(deviceResult, systemResult)

	for _, port := range ports {
		portResult := resultSet{}
		portResult.ID = device.ID
		portResult.BaseMac = device.BaseMac
		portResult.IP = device.IP
		portResult.SysUpDown = "down"
		if device.Up {
			portResult.SysUpDown = "up"
		}
		portResult.SysName = device.SysName
		portResult.SysLocation = device.SysLocation
		portResult.IfName = port.IfName
		portResult.IfStatus = port.IfOperStatus
		for _, vlan := range port.VlanList {
			if strings.Contains(vlan, "Untagged") {
				portResult.Untagged = append(portResult.Untagged, strings.Split(vlan, "[")[0])
			} else if strings.Contains(vlan, "Tagged") {
				portResult.Tagged = append(portResult.Tagged, strings.Split(vlan, "[")[0])
			}
		}
		deviceResult = append(deviceResult, portResult)
	}

	return deviceResult, nil
}

// Writes the results to outfile
func writeResults(results []resultSet) (uint, error) {
	var rowsWritten uint = 0

	fileHandle, fileErr := os.Create(config.outfile)
	if fileErr != nil {
		return rowsWritten, fmt.Errorf("Could not write outfile: %s", fileErr)
	}
	fileWriter := bufio.NewWriter(fileHandle)
	_, writeErr := fileWriter.WriteString("ID,BaseMac,IP,SysUpDown,SysName,SysLocation,IfName,IfStatus,Untagged,Tagged\n")
	if writeErr != nil {
		return rowsWritten, fmt.Errorf("Could not write outfile: %s", writeErr)
	}
	for _, row := range results {
		_, writeErr := fileWriter.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", row.ID, row.BaseMac, row.IP, row.SysUpDown, row.SysName, row.SysLocation, row.IfName, row.IfStatus, strings.Join(row.Untagged, ","), strings.Join(row.Tagged, ",")))
		if writeErr != nil {
			return rowsWritten, fmt.Errorf("Could not write outfile: %s", writeErr)
		}
		flushErr := fileWriter.Flush()
		if flushErr != nil {
			stdErr.Printf("Could not flush file buffer: %s\n", flushErr)
		}
		rowsWritten++
	}
	syncErr := fileHandle.Sync()
	if syncErr != nil {
		stdErr.Printf("Could not sync file handle: %s\n", syncErr)
	}
	fhErr := fileHandle.Close()
	if fhErr != nil {
		stdErr.Printf("Could not close file handle: %s\n", fhErr)
	}

	return rowsWritten, nil
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

func main() {
	parseCLIOptions()

	if config.printVersion {
		fmt.Println(httpUserAgent)
		os.Exit(0)
	}
	if config.outfile == "" {
		stdErr.Fatal("outfile is required.")
	}

	initializeClient()

	upDevices, downDevices := discoverManagedDevices()

	var rediscoveredDevices []string
	if config.refreshDevices {
		rediscoveredDevices = rediscoverDevices(upDevices)
	} else {
		rediscoveredDevices = upDevices
	}
	if config.includeDown {
		rediscoveredDevices = append(rediscoveredDevices, downDevices...)
	}
	sort.Strings(rediscoveredDevices)

	queryResults := []resultSet{}
	for _, deviceIP := range rediscoveredDevices {
		deviceResult, deviceErr := queryDevice(deviceIP)
		if deviceErr != nil {
			fmt.Println(deviceErr)
			continue
		}
		queryResults = append(queryResults, deviceResult...)
	}

	writeRows, writeErr := writeResults(queryResults)
	if writeErr != nil {
		stdOut.Println(writeErr)
	}

	stdOut.Printf("%d rows written to <%s>.", writeRows, config.outfile)
}
