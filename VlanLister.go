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

const toolName string = "VlanLister.go"
const toolVersion string = "2.0.0-dev"
const httpUserAgent string = toolName + "/" + toolVersion
const gqlDeviceListQuery string = `query {
	network {
	  devices {
		up
		ip
	  }
	}
  }`
const gqlMutationQuery string = `mutation {
	network {
	  rediscoverDevices(input: {devices: [{ipAddress: "%s"}]}) {
		status
		message
	  }
	}
  }`
const gqlDeviceDataQuery string = `query {
	network {
	  device(ip: "%s") {
		id
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
  }`

// created with https://mholt.github.io/json-to-go/
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
type deviceData struct {
	Data struct {
		Network struct {
			Device struct {
				ID          int    `json:"id"`
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
type resultSet struct {
	ID          int
	BaseMac     string
	IP          string
	SysName     string
	SysLocation string
	IfName      string
	IfStatus    string
	Untagged    []string
	Tagged      []string
}

var stdOut = log.New(os.Stdout, "", log.LstdFlags)
var stdErr = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	var httpHost string
	var httpTimeoutSecs uint
	var noHTTPS bool
	var insecureHTTPS bool
	var username string
	var password string
	var clientID string
	var clientSecret string
	var refreshDevices bool
	var refreshWaitSecs uint
	var operationWaitMins uint
	var outfile string
	var printVersion bool

	flag.StringVar(&httpHost, "host", "localhost", "XMC Hostname / IP")
	flag.UintVar(&httpTimeoutSecs, "timeout", 5, "Timeout for HTTP(S) connections")
	flag.BoolVar(&noHTTPS, "nohttps", false, "Use HTTP instead of HTTPS")
	flag.BoolVar(&insecureHTTPS, "insecurehttps", false, "Do not validate HTTPS certificates")
	flag.StringVar(&username, "username", "admin", "Username for HTTP Basic Auth")
	flag.StringVar(&password, "password", "", "Password for HTTP Basic Auth")
	flag.StringVar(&clientID, "clientid", "", "Client ID for OAuth")
	flag.StringVar(&clientSecret, "clientsecret", "", "Client Secret for OAuth")
	flag.BoolVar(&refreshDevices, "refreshdevices", true, "Refresh (rediscover) devices")
	flag.UintVar(&refreshWaitSecs, "refreshwait", 5, "Seconds to wait between triggering each refresh")
	flag.UintVar(&operationWaitMins, "operationwait", 15, "Minutes to wait after refreshing devices")
	flag.StringVar(&outfile, "outfile", "", "File to write CSV data to")
	flag.BoolVar(&printVersion, "version", false, "Print version information and exit")
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

	if printVersion {
		fmt.Println(httpUserAgent)
		os.Exit(0)
	}

	if outfile == "" {
		stdErr.Fatal("outfile is required.")
	}

	client := xmcnbiclient.New(httpHost)
	client.SetUserAgent(httpUserAgent)
	client.UseHTTPS()
	if noHTTPS {
		client.UseHTTP()
	}
	client.UseBasicAuth(username, password)
	if clientID != "" && clientSecret != "" {
		client.UseOAuth(clientID, clientSecret)
	}
	client.UseSecureHTTPS()
	if insecureHTTPS {
		client.UseInsecureHTTPS()
	}
	timeoutErr := client.SetTimeout(httpTimeoutSecs)
	if timeoutErr != nil {
		stdErr.Fatalf("Could not set HTTP timeout: %s\n", timeoutErr)
	}

	stdOut.Println("Discovering active devices...")

	body, bodyErr := client.QueryAPI(gqlDeviceListQuery)
	if bodyErr != nil {
		stdErr.Fatalf("Could not fetch device list: %s\n", bodyErr)
	}

	devices := deviceList{}
	jsonErr := json.Unmarshal(body, &devices)
	if jsonErr != nil {
		stdOut.Fatalf("Could not decode JSON: %s\n", jsonErr)
	}

	var upDevices []string
	for _, d := range devices.Data.Network.Devices {
		if d.Up {
			upDevices = append(upDevices, d.IP)
		}
	}
	sort.Strings(upDevices)
	stdOut.Println("Finished discovering active devices.")

	var rediscoveredDevices []string
	if refreshDevices {
		for _, deviceIP := range upDevices {
			body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlMutationQuery, deviceIP))
			if bodyErr != nil {
				stdErr.Printf("Could not mutate device %s: %s\n", deviceIP, bodyErr)
				continue
			}

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

			stdOut.Printf("Waiting for %d second(s)...\n", refreshWaitSecs)
			time.Sleep(time.Second * time.Duration(refreshWaitSecs))
		}
		for i := operationWaitMins; i > 0; i-- {
			stdOut.Printf("Waiting for %d minute(s) to finish rediscover...\n", i)
			time.Sleep(time.Minute * time.Duration(1))
		}
	} else {
		rediscoveredDevices = upDevices
	}
	sort.Strings(rediscoveredDevices)

	queryResults := []resultSet{}
	for _, deviceIP := range rediscoveredDevices {
		body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlDeviceDataQuery, deviceIP, deviceIP))
		if bodyErr != nil {
			stdErr.Printf("Could not query device %s: %s\n", deviceIP, bodyErr)
			continue
		}

		jsonData := deviceData{}
		jsonErr := json.Unmarshal(body, &jsonData)
		if jsonErr != nil {
			stdErr.Printf("Could not decode JSON: %s\n", jsonErr)
			continue
		}

		device := jsonData.Data.Network.Device
		vlans := jsonData.Data.Network.DeviceVlans
		ports := jsonData.Data.Network.Device.EntityData.AllPorts

		stdOut.Printf("Fetched data for %s: Got %d VLANs and %d ports.\n", device.IP, len(vlans), len(ports))

		systemResult := resultSet{}
		systemResult.ID = device.ID
		systemResult.BaseMac = device.BaseMac
		systemResult.IP = device.IP
		systemResult.SysName = device.SysName
		systemResult.SysLocation = device.SysLocation
		systemResult.IfName = "SYSTEM"
		systemResult.IfStatus = "N/A"
		for _, vlan := range vlans {
			systemResult.Tagged = append(systemResult.Tagged, strconv.Itoa(vlan.Vid))
		}
		queryResults = append(queryResults, systemResult)

		for _, port := range ports {
			portResult := resultSet{}
			portResult.ID = device.ID
			portResult.BaseMac = device.BaseMac
			portResult.IP = device.IP
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
			queryResults = append(queryResults, portResult)
		}
	}

	fileHandle, fileErr := os.Create(outfile)
	if fileErr != nil {
		stdErr.Fatalf("Could not write outfile: %s\n", fileErr)
	}
	fileWriter := bufio.NewWriter(fileHandle)
	_, writeErr := fileWriter.WriteString("ID,BaseMac,IP,SysName,SysLocation,IfName,IfStatus,Untagged,Tagged\n")
	if writeErr != nil {
		stdErr.Fatalf("Could not write outfile: %s\n", writeErr)
	}
	for _, row := range queryResults {
		_, writeErr := fileWriter.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", row.ID, row.BaseMac, row.IP, row.SysName, row.SysLocation, row.IfName, row.IfStatus, strings.Join(row.Untagged, ","), strings.Join(row.Tagged, ",")))
		if writeErr != nil {
			stdErr.Fatalf("Could not write outfile: %s\n", writeErr)
		}
		fileWriter.Flush()
	}
	fileHandle.Sync()
	fileHandle.Close()
}
