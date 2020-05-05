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
func queryDevice(client *xmcnbiclient.NBIClient, deviceIP string) (singleDevice, error) {
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
	sort.Slice(deviceResult.Vlans, func(i, j int) bool { return deviceResult.Vlans[i].ID < deviceResult.Vlans[j].ID })

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
		sort.Slice(portResult.UntaggedVlans, func(i, j int) bool { return portResult.UntaggedVlans[i] < portResult.UntaggedVlans[i] })
		sort.Slice(portResult.TaggedVlans, func(i, j int) bool { return portResult.TaggedVlans[i] < portResult.TaggedVlans[i] })
		deviceResult.Ports = append(deviceResult.Ports, portResult)
	}
	sort.Slice(deviceResult.Ports, func(i, j int) bool { return deviceResult.Ports[i].Index < deviceResult.Ports[j].Index })

	return deviceResult, nil
}
