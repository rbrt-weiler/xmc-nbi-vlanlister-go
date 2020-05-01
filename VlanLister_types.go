package main

import (
	"strconv"
	"strings"
)

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

// Old data storage construct >>

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

// << Old data storage construct
// New data storage construct >>

type deviceVlan struct {
	Type      string `json:"type"`
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PrimaryIP string `json:"primaryIp"`
	Netmask   string `json:"netmask"`
}

type devicePort struct {
	Index         int    `json:"index"`
	MACAddress    string `json:"macAddress"`
	Name          string `json:"name"`
	AdminStatus   string `json:"adminStatus"`
	OperStatus    string `json:"operStatus"`
	UntaggedVlans []int  `json:"untaggedVlans"`
	TaggedVlans   []int  `json:"taggedVlans"`
}

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

type deviceWrapper struct {
	Devices []singleDevice `json:"devices"`
}

// << New data storage construct
