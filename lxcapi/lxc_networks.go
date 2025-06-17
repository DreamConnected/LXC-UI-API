package lxcapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// NetworksHandler handles the synchronization request. It processes the HTTP request
// and sends the appropriate response back to the client.
type NetworkConfig struct {
	IPv4Address string `json:"ipv4.address"`
	IPv4Nat     string `json:"ipv4.nat"`
	IPv6Address string `json:"ipv6.address"`
	IPv6Nat     string `json:"ipv6.nat"`
}

type NetworkMetadata struct {
	Config      NetworkConfig `json:"config"`
	Description string        `json:"description"`
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	UsedBy      []string      `json:"used_by"`
	Managed     bool          `json:"managed"`
	Status      string        `json:"status"`
	Locations   []string      `json:"locations"`
	Project     string        `json:"project"`
}

type NetworkStateMetadata struct {
	Name      string `json:"name"`
	Hwaddr    string `json:"hwaddr"`
	Mtu       int    `json:"mtu"`
	State     string `json:"state"`
	Type      string `json:"type"`
	Addresses []struct {
		Family  string `json:"family"`
		Address string `json:"address"`
		Netmask string `json:"netmask"`
		Scope   string `json:"scope"`
	} `json:"addresses"`
	Counters struct {
		BytesReceived   int `json:"bytes_received"`
		BytesSent       int `json:"bytes_sent"`
		PacketsReceived int `json:"packets_received"`
		PacketsSent     int `json:"packets_sent"`
	} `json:"counters"`
	Bridge struct {
		ID            string   `json:"id"`
		Stp           bool     `json:"stp"`
		ForwardDelay  int      `json:"forward_delay"`
		VlanDefault   int      `json:"vlan_default"`
		VlanFiltering bool     `json:"vlan_filtering"`
		UpperDevices  []string `json:"upper_devices"`
	} `json:"bridge"`
	Vlan any `json:"vlan"`
	Ovn  any `json:"ovn"`
}

type GeneralResponse struct {
	Type       string `json:"type"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	Operation  string `json:"operation"`
	ErrorCode  int    `json:"error_code"`
	Error      string `json:"error"`
	Metadata   any    `json:"metadata"`
}

func NetworksHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	fmt.Println("Request Method:", r.Method, "|", "Request API:", path)
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(path, "/")
	var networkName, networkAction string
	var networkData any
	var err error
	if len(parts) == 4 {
		networkName = parts[3]
	} else if len(parts) >= 4 {
		networkName = parts[3]
		networkAction = parts[4]
	}

	if len(r.TLS.PeerCertificates) > 0 {
		if networkName == "" && networkAction == "" {
			networkData, err = getNetworkInterfaces()
		} else if networkName != "" && networkAction == "forwards" {
			networkData = []string{}
		} else if networkName != "" && networkAction == "state" {
			networkData, err = getNetworkInterfaceAction(networkName)
		} else {
			networkData, err = getNetworkInterfaceInfo(networkName)
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving network interfaces: %v", err), http.StatusInternalServerError)
			return
		}

		response := GeneralResponse{
			Type:       "sync",
			Status:     "Success",
			StatusCode: 200,
			Operation:  "",
			ErrorCode:  0,
			Error:      "",
			Metadata:   networkData,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "TLS certificate missing", http.StatusUnauthorized)
	}
}

func getNetworkInterfaces() ([]NetworkMetadata, error) {
	var metadataList []NetworkMetadata
	var ifType, ifManaged = "", false

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// while up
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		if iface.Name == "lo" {
			ifType = "loopback"
			ifManaged = false
		} else if iface.Name == "lxcbr0" {
			ifType = "bridge"
			ifManaged = true
		} else {
			ifType = "physical"
			ifManaged = false
		}

		metadata := NetworkMetadata{
			Name:        iface.Name,
			Type:        ifType,
			Managed:     ifManaged,
			Status:      "Created",
			Locations:   []string{"none"},
			Project:     "default",
			UsedBy:      []string{"/1.0/profiles/default"},
			Description: fmt.Sprintf("Network interface %s", iface.Name),
		}

		var ipv4Addr, ipv4Mask, ipv6Addr, ipv6Mask, v4Addr, v6Addr string
		var hasValidV4Addr, hasValidV6Addr bool
		for _, addr := range addrs {
			// v4 addresses
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				ipv4Addr = ipNet.IP.String()
				ipv4Mask = convertMaskToCIDR(ipNet.Mask)
				v4Addr = fmt.Sprintf("%s/%s", ipv4Addr, ipv4Mask)
				hasValidV4Addr = true
			}
			// v6 address
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To16() != nil && ipNet.IP.To4() == nil {
				ipv6Addr = ipNet.IP.String()

				if strings.HasPrefix(ipv6Addr, "fe80") {
					continue
				}

				ipv6Mask = convertMaskToCIDR(ipNet.Mask)
				v6Addr = fmt.Sprintf("%s/%s", ipv6Addr, ipv6Mask)
				hasValidV6Addr = true
			}
		}

		if !hasValidV4Addr {
			v4Addr = "-"
		}
		if !hasValidV6Addr {
			v6Addr = "-"
		}

		metadata.Config = NetworkConfig{
			IPv4Address: v4Addr,
			IPv4Nat:     "true",
			IPv6Address: v6Addr,
			IPv6Nat:     "true",
		}

		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func getNetworkInterfaceInfo(networkName string) (any, error) {
	interfaces, err := getNetworkInterfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if iface.Name == networkName {
			return iface, nil
		}
	}

	return nil, fmt.Errorf("network interface not found: %s", networkName)
}

func getNetworkInterfaceAction(networkName string) (any, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	var iface net.Interface
	for _, i := range interfaces {
		if i.Name == networkName {
			iface = i
			break
		}
	}
	if iface.Name == "" {
		return nil, fmt.Errorf("network interface not found: %s", networkName)
	}

	// Mac address
	hwaddr := iface.HardwareAddr.String()

	// Assuming all interfaces are up, but theoretically the down interfaces have been filtered out
	state := "up"

	addresses := []struct {
		Family  string `json:"family"`
		Address string `json:"address"`
		Netmask string `json:"netmask"`
		Scope   string `json:"scope"`
	}{}

	// v4 & v6
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface addresses: %v", err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil {
			addresses = append(addresses, struct {
				Family  string `json:"family"`
				Address string `json:"address"`
				Netmask string `json:"netmask"`
				Scope   string `json:"scope"`
			}{
				Family:  "inet",
				Address: ipNet.IP.String(),
				Netmask: convertMaskToCIDR(ipNet.Mask),
				Scope:   "global",
			})
		} else if ipNet.IP.To16() != nil {
			addresses = append(addresses, struct {
				Family  string `json:"family"`
				Address string `json:"address"`
				Netmask string `json:"netmask"`
				Scope   string `json:"scope"`
			}{
				Family:  "inet6",
				Address: ipNet.IP.String(),
				Netmask: convertMaskToCIDR(ipNet.Mask),
				Scope:   "global",
			})
		}
	}

	stateData := NetworkStateMetadata{
		Name:   iface.Name,
		Hwaddr: hwaddr,
		Mtu:    iface.MTU,
		State:  state,
		Type:   "broadcast",

		Addresses: addresses,

		Counters: struct {
			BytesReceived   int `json:"bytes_received"`
			BytesSent       int `json:"bytes_sent"`
			PacketsReceived int `json:"packets_received"`
			PacketsSent     int `json:"packets_sent"`
		}{
			BytesReceived:   0,
			BytesSent:       0,
			PacketsReceived: 0,
			PacketsSent:     0,
		},

		Bridge: struct {
			ID            string   `json:"id"`
			Stp           bool     `json:"stp"`
			ForwardDelay  int      `json:"forward_delay"`
			VlanDefault   int      `json:"vlan_default"`
			VlanFiltering bool     `json:"vlan_filtering"`
			UpperDevices  []string `json:"upper_devices"`
		}{
			ID:            "8000.10666a7b6731", // this is brid, can be obtained from brctl
			Stp:           false,
			ForwardDelay:  1500,
			VlanDefault:   1,
			VlanFiltering: true,
			UpperDevices:  []string{},
		},
	}

	return stateData, nil
}
