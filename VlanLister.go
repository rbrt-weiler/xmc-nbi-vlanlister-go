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

	excelize "github.com/360EntSecGroup-Skylar/excelize"
	godotenv "github.com/joho/godotenv"
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
	toolName           string = "VlanLister.go"
	toolVersion        string = "2.0.0-dev"
	toolID             string = toolName + "/" + toolVersion
	envFileName        string = ".xmcenv"
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
	XMCHost        string
	XMCPort        uint
	XMCPath        string
	HTTPTimeout    uint
	NoHTTPS        bool
	InsecureHTTPS  bool
	BasicAuth      bool
	XMCUserID      string
	XMCSecret      string
	XMCQuery       string
	RefreshDevices bool
	RefreshWait    uint
	FinishWait     uint
	IncludeDown    bool
	Outfile        string
	OutXLSX        bool
	PrintVersion   bool
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

// Convert a single resultSet to an array
func (rs *resultSet) ToArray() []string {
	retVal := []string{strconv.Itoa(rs.ID), rs.BaseMac, rs.IP, rs.SysUpDown, rs.SysName, rs.SysLocation, rs.IfName, rs.IfStatus, strings.Join(rs.Untagged, ","), strings.Join(rs.Tagged, ",")}
	return retVal
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
	// Columns used in outfiles
	tableColumns = [...]string{"ID", "BaseMac", "IP", "SysUpDown", "SysName", "SysLocation", "IfName", "IfStatus", "Untagged", "Tagged"}
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
	flag.StringVar(&config.XMCHost, "host", envordef.StringVal("XMCHOST", ""), "XMC Hostname / IP")
	flag.UintVar(&config.XMCPort, "port", envordef.UintVal("XMCPORT", 8443), "HTTP port where XMC is listening")
	flag.StringVar(&config.XMCPath, "path", envordef.StringVal("XMCPATH", ""), "Path where XMC is reachable")
	flag.UintVar(&config.HTTPTimeout, "timeout", envordef.UintVal("XMCTIMEOUT", 5), "Timeout for HTTP(S) connections")
	flag.BoolVar(&config.NoHTTPS, "nohttps", envordef.BoolVal("XMCNOHTTPS", false), "Use HTTP instead of HTTPS")
	flag.BoolVar(&config.InsecureHTTPS, "insecurehttps", envordef.BoolVal("XMCINSECUREHTTPS", false), "Do not validate HTTPS certificates")
	flag.StringVar(&config.XMCUserID, "userid", envordef.StringVal("XMCUSERID", ""), "Client ID (OAuth) or username (Basic Auth) for authentication")
	flag.StringVar(&config.XMCSecret, "secret", envordef.StringVal("XMCSECRET", ""), "Client Secret (OAuth) or password (Basic Auth) for authentication")
	flag.BoolVar(&config.BasicAuth, "basicauth", envordef.BoolVal("XMCBASICAUTH", false), "Use HTTP Basic Auth instead of OAuth")
	flag.BoolVar(&config.RefreshDevices, "refreshdevices", envordef.BoolVal("XMCREFRESHDEVICES", true), "Refresh (rediscover) devices")
	flag.UintVar(&config.RefreshWait, "refreshwait", envordef.UintVal("XMCREFRESHWAIT", 5), "Seconds to wait between triggering each refresh")
	flag.UintVar(&config.FinishWait, "finishwait", envordef.UintVal("XMCFINISHWAIT", 15), "Minutes to wait after refreshing devices")
	flag.BoolVar(&config.IncludeDown, "includedown", envordef.BoolVal("XMCINCLUDEDOWN", false), "Include inactive devices in result")
	flag.StringVar(&config.Outfile, "outfile", "", "File to write CSV/XLSX data to")
	flag.BoolVar(&config.OutXLSX, "outxlsx", false, "Write XLSX instead of CSV")
	flag.BoolVar(&config.PrintVersion, "version", false, "Print version information and exit")
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
		fmt.Fprintf(os.Stderr, "Nearly all options that take a value can be set via environment variables:\n")
		fmt.Fprintf(os.Stderr, "  XMCHOST            -->  -host\n")
		fmt.Fprintf(os.Stderr, "  XMCPORT            -->  -port\n")
		fmt.Fprintf(os.Stderr, "  XMCPATH            -->  -path\n")
		fmt.Fprintf(os.Stderr, "  XMCTIMEOUT         -->  -timeout\n")
		fmt.Fprintf(os.Stderr, "  XMCNOHTTPS         -->  -nohttps\n")
		fmt.Fprintf(os.Stderr, "  XMCINSECUREHTTPS   -->  -insecurehttps\n")
		fmt.Fprintf(os.Stderr, "  XMCUSERID          -->  -userid\n")
		fmt.Fprintf(os.Stderr, "  XMCSECRET          -->  -secret\n")
		fmt.Fprintf(os.Stderr, "  XMCBASICAUTH       -->  -basicauth\n")
		fmt.Fprintf(os.Stderr, "  XMCREFRESHDEVICES  -->  -refreshdevices\n")
		fmt.Fprintf(os.Stderr, "  XMCREFRESHWAIT     -->  -refreshwait\n")
		fmt.Fprintf(os.Stderr, "  XMCFINISHWAIT      -->  -finishwait\n")
		fmt.Fprintf(os.Stderr, "  XMCINCLUDEDOWN     -->  -includedown\n")
	}
	flag.Parse()
}

// Initializes the actual XMC client
func initializeClient() {
	client = xmcnbiclient.New(config.XMCHost)
	client.SetUserAgent(toolID)
	client.UseHTTPS()
	if config.NoHTTPS {
		client.UseHTTP()
	}
	client.UseOAuth(config.XMCUserID, config.XMCSecret)
	if config.BasicAuth {
		client.UseBasicAuth(config.XMCUserID, config.XMCSecret)
	}
	client.UseSecureHTTPS()
	if config.InsecureHTTPS {
		client.UseInsecureHTTPS()
	}
	timeoutErr := client.SetTimeout(config.HTTPTimeout)
	if timeoutErr != nil {
		stdErr.Fatalf("Could not set HTTP timeout: %s\n", timeoutErr)
	}
}

// Refreshes the OAuth token if it is to expire soon
func proactiveTokenRefresh() {
	if client.Authentication.Type == xmcnbiclient.AuthTypeOAuth {
		if client.AccessToken.ExpiresSoon(config.HTTPTimeout + 1) {
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

		stdOut.Printf("Waiting for %d second(s)...\n", config.RefreshWait)
		time.Sleep(time.Second * time.Duration(config.RefreshWait))
	}
	for i := config.FinishWait; i > 0; i-- {
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

// Writes the results to outfile in CSV format
func writeResultsCSV(results []resultSet) (uint, error) {
	var rowsWritten uint = 0

	fileHandle, fileErr := os.Create(config.Outfile)
	if fileErr != nil {
		return rowsWritten, fmt.Errorf("Could not write outfile: %s", fileErr)
	}
	fileWriter := bufio.NewWriter(fileHandle)
	_, writeErr := fileWriter.WriteString(fmt.Sprintf("%s\n", strings.Join(tableColumns[0:10], ",")))
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

// Writes the results to outfile in XLSX format
func writeResultsXLSX(results []resultSet) (uint, error) {
	var rowsWritten uint = 0
	var colIndex int = 1
	var rowIndex int = 1

	xlsx := excelize.NewFile()

	for _, columnName := range tableColumns {
		position, positionErr := excelize.CoordinatesToCellName(colIndex, rowIndex)
		if positionErr != nil {
			return rowsWritten, positionErr
		}
		xlsx.SetCellValue("Sheet1", position, columnName)
		colIndex++
	}

	for _, row := range results {
		colIndex = 1
		rowIndex++
		fmt.Println(row.ToArray())
		for _, element := range row.ToArray() {
			position, positionErr := excelize.CoordinatesToCellName(colIndex, rowIndex)
			if positionErr != nil {
				return rowsWritten, positionErr
			}
			xlsx.SetCellValue("Sheet1", position, element)
			colIndex++
		}
		rowsWritten++
	}

	if saveErr := xlsx.SaveAs(config.Outfile); saveErr != nil {
		return rowsWritten, saveErr
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

// init loads environment files if available.
func init() {
	// if envFileName exists in the current directory, load it
	localEnvFile := fmt.Sprintf("./%s", envFileName)
	if _, localEnvErr := os.Stat(localEnvFile); localEnvErr == nil {
		if loadErr := godotenv.Load(localEnvFile); loadErr != nil {
			fmt.Fprintf(os.Stderr, "Could not load env file <%s>: %s", localEnvFile, loadErr)
		}
	}

	// if envFileName exists in the user's home directory, load it
	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		homeEnvFile := fmt.Sprintf("%s/%s", homeDir, ".xmcenv")
		if _, homeEnvErr := os.Stat(homeEnvFile); homeEnvErr == nil {
			if loadErr := godotenv.Load(homeEnvFile); loadErr != nil {
				fmt.Fprintf(os.Stderr, "Could not load env file <%s>: %s", homeEnvFile, loadErr)
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
	if config.Outfile == "" {
		stdErr.Fatal("outfile is required.")
	}

	initializeClient()

	upDevices, downDevices := discoverManagedDevices()

	var rediscoveredDevices []string
	if config.RefreshDevices {
		rediscoveredDevices = rediscoverDevices(upDevices)
	} else {
		rediscoveredDevices = upDevices
	}
	if config.IncludeDown {
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

	var writeRows uint
	var writeErr error
	if config.OutXLSX {
		writeRows, writeErr = writeResultsXLSX(queryResults)
	} else {
		writeRows, writeErr = writeResultsCSV(queryResults)
	}
	if writeErr != nil {
		stdOut.Println(writeErr)
	}

	stdOut.Printf("%d rows written to <%s>.", writeRows, config.Outfile)
}
