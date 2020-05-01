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
	"encoding/json"
	"fmt"
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
					baseMac
					ip
					sysName
					sysLocation
					nickName
					entityData {
						allPorts {
							ifIndex
							ifPhysAddress
							ifName
							ifAdminStatus
							ifOperStatus
							vlanList
						}
					}
				}
				deviceVlans(ip: "%s") {
					type
					vid
					name
					primaryIp
					netmask
				}
			}
		}
	`
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

// Fetches the complete list of managed devices from XMC
func discoverManagedDevices(client *xmcnbiclient.NBIClient) ([]string, []string) {
	stdErr.Println("Discovering managed devices...")

	body, bodyErr := client.QueryAPI(gqlDeviceListQuery)
	if bodyErr != nil {
		stdErr.Fatalf("Could not fetch device list: %s\n", bodyErr)
	}
	proactiveTokenRefresh(client)

	devices := xmcDeviceList{}
	jsonErr := json.Unmarshal(body, &devices)
	if jsonErr != nil {
		stdErr.Fatalf("Could not decode JSON: %s\n", jsonErr)
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
	stdErr.Println("Finished discovering managed devices.")

	return upDevices, downDevices
}

// Triggers a rediscover for a list of devices
func rediscoverDevices(client *xmcnbiclient.NBIClient, ipList []string) []string {
	var rediscoveredDevices []string
	for _, deviceIP := range ipList {
		body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlMutationQuery, deviceIP))
		if bodyErr != nil {
			stdErr.Printf("Could not mutate device %s: %s\n", deviceIP, bodyErr)
			continue
		}
		proactiveTokenRefresh(client)

		mutation := xmcMutationMessage{}
		jsonErr := json.Unmarshal(body, &mutation)
		if jsonErr != nil {
			stdErr.Printf("Could not decode JSON: %s\n", jsonErr)
			continue
		}

		if mutation.Data.Network.RediscoverDevices.Status == "SUCCESS" {
			stdErr.Printf("Successfully triggered rediscover for %s.\n", deviceIP)
			rediscoveredDevices = append(rediscoveredDevices, deviceIP)
		} else {
			stdErr.Printf("Rediscover for %s failed: %s\n", deviceIP, mutation.Data.Network.RediscoverDevices.Message)
		}

		stdErr.Printf("Waiting for %d second(s)...\n", config.RefreshInterval)
		time.Sleep(time.Second * time.Duration(config.RefreshInterval))
	}
	for i := config.RefreshWait; i > 0; i-- {
		proactiveTokenRefresh(client)
		stdErr.Printf("Waiting for %d minute(s) to finish rediscover...\n", i)
		time.Sleep(time.Minute * time.Duration(1))
	}
	return rediscoveredDevices
}

// Fetches the detailed data for a single device from XMC
func queryDevice(client *xmcnbiclient.NBIClient, deviceIP string) ([]resultSet, error) {
	var deviceResult []resultSet

	body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlDeviceDataQuery, deviceIP, deviceIP))
	if bodyErr != nil {
		return deviceResult, fmt.Errorf("Could not query device %s: %s", deviceIP, bodyErr)
	}
	proactiveTokenRefresh(client)

	jsonData := xmcDeviceData{}
	jsonErr := json.Unmarshal(body, &jsonData)
	if jsonErr != nil {
		return deviceResult, fmt.Errorf("Could not decode JSON: %s", jsonErr)
	}

	device := jsonData.Data.Network.Device
	vlans := jsonData.Data.Network.DeviceVlans
	ports := jsonData.Data.Network.Device.EntityData.AllPorts

	stdErr.Printf("Fetched data for %s: Got %d VLANs and %d ports.\n", device.IP, len(vlans), len(ports))

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

func queryDeviceNew(client *xmcnbiclient.NBIClient, deviceIP string) (singleDevice, error) {
	var deviceResult singleDevice

	deviceResult.QueriedAt = time.Now().Format(time.RFC3339)

	body, bodyErr := client.QueryAPI(fmt.Sprintf(gqlDeviceDataQuery, deviceIP, deviceIP))
	if bodyErr != nil {
		return deviceResult, fmt.Errorf("Could not query device %s: %s", deviceIP, bodyErr)
	}
	proactiveTokenRefresh(client)

	jsonData := xmcDeviceData{}
	jsonErr := json.Unmarshal(body, &jsonData)
	if jsonErr != nil {
		return deviceResult, fmt.Errorf("Could not decode JSON: %s", jsonErr)
	}

	device := jsonData.Data.Network.Device
	vlans := jsonData.Data.Network.DeviceVlans
	ports := jsonData.Data.Network.Device.EntityData.AllPorts

	stdErr.Printf("Fetched data for %s: Got %d VLANs and %d ports.\n", device.IP, len(vlans), len(ports))

	deviceResult.ID = device.ID
	deviceResult.Up = device.Up
	deviceResult.BaseMAC = device.BaseMac
	deviceResult.IPAddress = device.IP
	deviceResult.SysName = device.SysName
	deviceResult.SysLocation = device.SysLocation
	deviceResult.NickName = device.NickName

	for _, vlan := range vlans {
		vlanResult := deviceVlan{}
		vlanResult.Type = vlan.Type
		vlanResult.ID = vlan.Vid
		vlanResult.Name = vlan.Name
		vlanResult.PrimaryIP = vlan.PrimaryIP
		vlanResult.Netmask = vlan.Netmask
		deviceResult.Vlans = append(deviceResult.Vlans, vlanResult)
	}

	for _, port := range ports {
		portResult := devicePort{}
		portResult.Index = port.IfIndex
		portResult.MACAddress = port.IfPhysAddress
		portResult.Name = port.IfName
		portResult.AdminStatus = port.IfAdminStatus
		portResult.OperStatus = port.IfOperStatus
		for _, vlan := range port.VlanList {
			vid, vidError := strconv.Atoi(strings.Split(vlan, "[")[0])
			if vidError != nil {
				stdErr.Printf("Could not convert VLAN ID: %s\n", vidError)
				continue
			}
			if strings.Contains(vlan, "Untagged") {
				portResult.UntaggedVlans = append(portResult.UntaggedVlans, vid)
			} else if strings.Contains(vlan, "Tagged") {
				portResult.TaggedVlans = append(portResult.TaggedVlans, vid)
			}
		}
		deviceResult.Ports = append(deviceResult.Ports, portResult)
	}

	return deviceResult, nil
}
