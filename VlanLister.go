/*
Copyright (c) 2019 BELL Computer-Netzwerke GmbH
Copyright (c) 2019 Robert Weiler

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
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ToolName string = "BELL XMC NBI VlanLister"
const ToolVersion string = "1.2.1"
const HttpUserAgent string = ToolName + "/" + ToolVersion
const GqlDeviceListQuery string = `query {
	network {
	  devices {
		up
		ip
		sysName
		deviceData {
		  vendor
		  family
		  subFamily
		}
	  }
	}
  }`
const GqlMutationQuery string = `mutation {
	network {
	  rediscoverDevices(input: {devices: [{ipAddress: "%s"}]}) {
		status
		message
	  }
	}
  }`
const GqlDeviceDataQuery string = `query {
	network {
	  device(ip: "%s") {
		deviceData {
		  physicalPortCount
		  serialNumber
		}
		id
		sysName
		sysLocation
		nickName
		firmware
		baseMac
		ip
		entityData {
		  allPorts {
			ifIndex
			ifName
			ifOperStatus
			ifSpeed
			vlanList
		  }
		}
	  }
	  deviceVlans(ip: "%s") {
		name
		vid
		primaryIp
		netmask
		ipForward
	  }
	}
  }`

// created with https://mholt.github.io/json-to-go/
type DeviceList struct {
	Data struct {
		Network struct {
			Devices []struct {
				Up         bool   `json:"up"`
				IP         string `json:"ip"`
				SysName    string `json:"sysName"`
				DeviceData struct {
					Vendor    string `json:"vendor"`
					Family    string `json:"family"`
					SubFamily string `json:"subFamily"`
				} `json:"deviceData"`
			} `json:"devices"`
		} `json:"network"`
	} `json:"data"`
}
type MutationMessage struct {
	Data struct {
		Network struct {
			RediscoverDevices struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"rediscoverDevices"`
		} `json:"network"`
	} `json:"data"`
}
type DeviceData struct {
	Data struct {
		Network struct {
			Device struct {
				DeviceData struct {
					PhysicalPortCount int    `json:"physicalPortCount"`
					SerialNumber      string `json:"serialNumber"`
				} `json:"deviceData"`
				ID          int    `json:"id"`
				SysName     string `json:"sysName"`
				SysLocation string `json:"sysLocation"`
				NickName    string `json:"nickName"`
				Firmware    string `json:"firmware"`
				BaseMac     string `json:"baseMac"`
				IP          string `json:"ip"`
				EntityData  struct {
					AllPorts []struct {
						IfIndex      int      `json:"ifIndex"`
						IfName       string   `json:"ifName"`
						IfOperStatus string   `json:"ifOperStatus"`
						IfSpeed      string   `json:"ifSpeed"`
						VlanList     []string `json:"vlanList"`
					} `json:"allPorts"`
				} `json:"entityData"`
			} `json:"device"`
			DeviceVlans []struct {
				Name      string `json:"name"`
				Vid       int    `json:"vid"`
				PrimaryIP string `json:"primaryIp"`
				Netmask   string `json:"netmask"`
				IPForward int    `json:"ipForward"`
			} `json:"deviceVlans"`
		} `json:"network"`
	} `json:"data"`
}
type ResultSet struct {
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

func retrieveApiResult(httpClient *http.Client, apiUrl string, username string, password string, queryString string) []byte {
	req, err := http.NewRequest(http.MethodGet, apiUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", HttpUserAgent)
	req.SetBasicAuth(username, password)

	httpQuery := req.URL.Query()
	httpQuery.Add("query", queryString)
	req.URL.RawQuery = httpQuery.Encode()

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	return body
}

func main() {
	var host string
	var httpTimeout uint
	var insecureHttps bool
	var username string
	var password string
	var mutateDevices bool
	var mutationWait uint
	var mutationPause uint
	var outfile string

	flag.StringVar(&host, "host", "localhost", "XMC Hostname / IP")
	flag.UintVar(&httpTimeout, "httptimeout", 5, "Timeout for HTTP(S) connections")
	flag.BoolVar(&insecureHttps, "insecurehttps", false, "Do not validate HTTPS certificates")
	flag.StringVar(&username, "username", "admin", "Username for HTTP auth")
	flag.StringVar(&password, "password", "", "Password for HTTP auth")
	flag.BoolVar(&mutateDevices, "mutdevices", true, "Mutate (rediscover) devices")
	flag.UintVar(&mutationWait, "mutwait", 5, "Seconds to wait between mutations")
	flag.UintVar(&mutationPause, "mutpause", 15, "Minutes to wait after mutating devices")
	flag.StringVar(&outfile, "outfile", "", "File to write CSV data to")
	flag.Parse()

	stdOut := log.New(os.Stdout, "", log.LstdFlags)
	stdErr := log.New(os.Stderr, "", log.LstdFlags)

	if outfile == "" {
		stdErr.Fatal("outfile is required.")
	}

	stdOut.Println("Discovering active devices...")

	var apiUrl string = "https://" + host + ":8443/nbi/graphql"
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureHttps},
	}
	nbiClient := http.Client{
		Transport: httpTransport,
		Timeout:   time.Second * time.Duration(httpTimeout),
	}

	body := retrieveApiResult(&nbiClient, apiUrl, username, password, GqlDeviceListQuery)

	devices := DeviceList{}
	jsonErr := json.Unmarshal(body, &devices)
	if jsonErr != nil {
		stdOut.Fatal(jsonErr)
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
	if mutateDevices {
		for _, deviceIp := range upDevices {
			stdOut.Printf("Waiting for %d seconds...\n", mutationWait)
			time.Sleep(time.Second * time.Duration(mutationWait))

			body := retrieveApiResult(&nbiClient, apiUrl, username, password, fmt.Sprintf(GqlMutationQuery, deviceIp))

			mutation := MutationMessage{}
			jsonErr := json.Unmarshal(body, &mutation)
			if jsonErr != nil {
				stdErr.Println(jsonErr)
				continue
			}

			if mutation.Data.Network.RediscoverDevices.Status == "SUCCESS" {
				stdOut.Printf("Successfully triggered rediscover for %s.\n", deviceIp)
				rediscoveredDevices = append(rediscoveredDevices, deviceIp)
			} else {
				stdErr.Printf("Rediscover for %s failed: %s\n", deviceIp, mutation.Data.Network.RediscoverDevices.Message)
			}
		}
	} else {
		rediscoveredDevices = upDevices
	}
	sort.Strings(rediscoveredDevices)

	if mutateDevices {
		for i := mutationPause; i > 0; i-- {
			stdOut.Printf("Waiting for %d minutes to finish rediscover...\n", i)
			time.Sleep(time.Minute * time.Duration(1))
		}
	}

	queryResults := []ResultSet{}
	for _, deviceIp := range rediscoveredDevices {
		body := retrieveApiResult(&nbiClient, apiUrl, username, password, fmt.Sprintf(GqlDeviceDataQuery, deviceIp, deviceIp))

		jsonData := DeviceData{}
		jsonErr := json.Unmarshal(body, &jsonData)
		if jsonErr != nil {
			stdErr.Println(jsonErr)
			continue
		}

		device := jsonData.Data.Network.Device
		vlans := jsonData.Data.Network.DeviceVlans
		ports := jsonData.Data.Network.Device.EntityData.AllPorts

		stdOut.Printf("Fetched data for %s: Got %d VLANs and %d ports.", device.IP, len(vlans), len(ports))

		systemResult := ResultSet{}
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
			portResult := ResultSet{}
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
		stdErr.Fatalf("Could not write outfile: %s", fileErr)
	}
	fileWriter := bufio.NewWriter(fileHandle)
	_, writeErr := fileWriter.WriteString("ID,BaseMac,IP,SysName,SysLocation,IfName,IfStatus,Untagged,Tagged\n")
	if writeErr != nil {
		stdErr.Fatalf("Could not write outfile: %s", writeErr)
	}
	for _, row := range queryResults {
		_, writeErr := fileWriter.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", row.ID, row.BaseMac, row.IP, row.SysName, row.SysLocation, row.IfName, row.IfStatus, strings.Join(row.Untagged, ","), strings.Join(row.Tagged, ",")))
		if writeErr != nil {
			stdErr.Fatalf("Could not write outfile: %s", writeErr)
		}
		fileWriter.Flush()
	}
	fileHandle.Sync()
	fileHandle.Close()
}
