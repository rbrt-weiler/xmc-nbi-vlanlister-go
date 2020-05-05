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
	"strconv"
	"strings"
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
	csvFormatString string = `"%s","%s","%s","%s","%s","%s","%s","%s","%s","%s"`
)

var (
	// Columns used in outfiles
	csvColumns = [...]string{"ID", "BaseMac", "IP", "SysUpDown", "SysName", "SysLocation", "IfName", "IfStatus", "Untagged", "Tagged"}
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

// Stores an array of strings.
// Used for storing multiple "outfile" CLI arguments. flags needs the associated functions.
type stringArray []string

// Returns a concatenated representation of all elements.
func (oa *stringArray) String() string {
	return strings.Join(*oa, ",")
}

// Appends a new element.
func (oa *stringArray) Set(value string) error {
	*oa = append(*oa, value)
	return nil
}

// Stores configuration data used throughout the app.
type appConfig struct {
	XMCHost         string
	XMCPort         uint
	XMCPath         string
	HTTPTimeout     uint
	NoHTTPS         bool
	InsecureHTTPS   bool
	BasicAuth       bool
	XMCUserID       string
	XMCSecret       string
	XMCQuery        string
	NoRefresh       bool
	RefreshInterval uint
	RefreshWait     uint
	IncludeDown     bool
	Outfile         stringArray
	PrintVersion    bool
}

// Used for parsing the list of devices returned by XMC.
type xmcDeviceList struct {
	Data struct {
		Network struct {
			Devices []struct {
				Up bool   `json:"up"`
				IP string `json:"ip"`
			} `json:"devices"`
		} `json:"network"`
	} `json:"data"`
}

// Used to parse the result returned by XMC for each single mutation (device refresh).
type xmcMutationMessage struct {
	Data struct {
		Network struct {
			RediscoverDevices struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"rediscoverDevices"`
		} `json:"network"`
	} `json:"data"`
}

// Used to parse the result returned by XMC for each single device.
type xmcDeviceData struct {
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
						IfIndex       int      `json:"ifIndex"`
						IfPhysAddress string   `json:"ifPhysAddress"`
						IfName        string   `json:"ifName"`
						IfAdminStatus string   `json:"ifAdminStatus"`
						IfOperStatus  string   `json:"ifOperStatus"`
						VlanList      []string `json:"vlanList"`
					} `json:"allPorts"`
				} `json:"entityData"`
			} `json:"device"`
			DeviceVlans []struct {
				Type      string `json:"type"`
				Vid       int    `json:"vid"`
				Name      string `json:"name"`
				PrimaryIP string `json:"primaryIp"`
				Netmask   string `json:"netmask"`
			} `json:"deviceVlans"`
		} `json:"network"`
	} `json:"data"`
}

// Stores data related to the VLANs configured on a device.
type deviceVlan struct {
	Type      string `json:"type"`
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PrimaryIP string `json:"primaryIp"`
	Netmask   string `json:"netmask"`
}

// Stores data related to the ports of a device.
type devicePort struct {
	Index         int    `json:"index"`
	MACAddress    string `json:"macAddress"`
	Name          string `json:"name"`
	AdminStatus   string `json:"adminStatus"`
	OperStatus    string `json:"operStatus"`
	UntaggedVlans []int  `json:"untaggedVlans"`
	TaggedVlans   []int  `json:"taggedVlans"`
}

// Stores all data related to a single device.
type singleDevice struct {
	ID          int          `json:"id"`
	QueriedAt   string       `json:"queriedAt"`
	Up          bool         `json:"up"`
	BaseMAC     string       `json:"baseMac"`
	IPAddress   string       `json:"ipAddress"`
	SysName     string       `json:"sysName"`
	SysLocation string       `json:"sysLocation"`
	NickName    string       `json:"nickName"`
	Vlans       []deviceVlan `json:"vlans"`
	Ports       []devicePort `json:"ports"`
}

// Stores multiple devices.
type devicesWrapper struct {
	Devices []singleDevice `json:"devices"`
}

/*
######## ##    ## ########  ########    ######## ##     ## ##    ##  ######   ######
   ##     ##  ##  ##     ## ##          ##       ##     ## ###   ## ##    ## ##    ##
   ##      ####   ##     ## ##          ##       ##     ## ####  ## ##       ##
   ##       ##    ########  ######      ######   ##     ## ## ## ## ##        ######
   ##       ##    ##        ##          ##       ##     ## ##  #### ##             ##
   ##       ##    ##        ##          ##       ##     ## ##   ### ##    ## ##    ##
   ##       ##    ##        ########    ##        #######  ##    ##  ######   ######
*/

// Transforms a singleDevice struct into an array of strings representing CSV output.
func (sd *singleDevice) ToCSVRows() ([]string, error) {
	var result []string
	var sysID string
	var sysUpDown string
	var systemVlanIDs []string

	sysID = strconv.Itoa(sd.ID)
	sysUpDown = "down"
	if sd.Up {
		sysUpDown = "up"
	}
	for _, vlan := range sd.Vlans {
		systemVlanIDs = append(systemVlanIDs, strconv.Itoa(vlan.ID))
	}
	result = append(result, fmt.Sprintf(csvFormatString, sysID, sd.BaseMAC, sd.IPAddress, sysUpDown, sd.SysName, sd.SysLocation, "SYSTEM", "N/A", "", strings.Join(systemVlanIDs, ",")))
	for _, port := range sd.Ports {
		var portUntaggedVlans []string
		var portTaggedVlans []string
		for untaggedID := range port.UntaggedVlans {
			portTaggedVlans = append(portTaggedVlans, strconv.Itoa(untaggedID))
		}
		for taggedID := range port.TaggedVlans {
			portTaggedVlans = append(portTaggedVlans, strconv.Itoa(taggedID))
		}
		result = append(result, fmt.Sprintf(csvFormatString, sysID, sd.BaseMAC, sd.IPAddress, sysUpDown, sd.SysName, sd.SysLocation, port.Name, port.OperStatus, strings.Join(portUntaggedVlans, ","), strings.Join(portTaggedVlans, ",")))
	}

	return result, nil
}

// Transforms a devicesWrapper struct into a string representing CSV output.
func (dw *devicesWrapper) ToCSV() (string, error) {
	var result []string

	result = append(result, fmt.Sprintf(csvFormatString, csvColumns[0], csvColumns[1], csvColumns[2], csvColumns[3], csvColumns[4], csvColumns[5], csvColumns[6], csvColumns[7], csvColumns[8], csvColumns[9]))

	for _, dev := range dw.Devices {
		csvRows, csvRowsError := dev.ToCSVRows()
		if csvRowsError != nil {
			return "", fmt.Errorf("Could not convert device to CSV: %s", csvRowsError)
		}
		for row := range csvRows {
			result = append(result, csvRows[row])
		}
	}

	return strings.Join(result, "\n"), nil
}

// Transforms a devicesWrapper struct into a string representing JSON output.
func (dw *devicesWrapper) ToJSON() (string, error) {
	json, jsonErr := json.MarshalIndent(dw, "", "    ")
	if jsonErr != nil {
		return "", fmt.Errorf("Could not encode JSON: %s", jsonErr)
	}
	return string(json), nil
}
