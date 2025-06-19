package lxcapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"maps"

	uuid "github.com/satori/go.uuid"
)

type Address struct {
	Family  string `json:"family"`
	Address string `json:"address"`
	Netmask string `json:"netmask"`
	Scope   string `json:"scope"`
}

type NetworkInfo struct {
	Counters  map[string]int64 `json:"counters"`
	Addresses []Address        `json:"addresses"`
}

type State struct {
	Status     string                      `json:"status"`
	StatusCode int                         `json:"status_code"`
	Disk       map[string]map[string]int64 `json:"disk"`
	Memory     map[string]int64            `json:"memory"`
	Network    map[string]*NetworkInfo     `json:"network"`
	Cpu        map[string]int64            `json:"cpu"`
	Pid        int64                       `json:"pid"`
	Processes  int                         `json:"processes"`
	StartedAt  string                      `json:"started_at"`
	OsInfo     any                         `json:"os_info"`
}

// get实例info
type InstanceMetadata struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Status          string            `json:"status"`
	StatusCode      int               `json:"status_code"`
	CreatedAt       time.Time         `json:"created_at"`
	LastUsedAt      time.Time         `json:"last_used_at"`
	Location        string            `json:"location"`
	Type            string            `json:"type"`
	Project         string            `json:"project"`
	Architecture    any               `json:"architecture"`
	Ephemeral       bool              `json:"ephemeral"`
	Stateful        bool              `json:"stateful"`
	Profiles        []string          `json:"profiles"`
	Config          InstanceConfig    `json:"config"`
	Devices         map[string]any    `json:"devices"`
	ExpandedConfig  map[string]any    `json:"expanded_config"`
	ExpandedDevices map[string]Device `json:"expanded_devices"`
	Backups         any               `json:"backups"`
	State           any               `json:"state"`
	Snapshots       any               `json:"snapshots"`
}

type InstanceConfig struct {
	ImageArchitecture any `json:"image.architecture"`
	ImageDescription  any `json:"image.description"`
	ImageLabel        any `json:"image.label"`
	ImageOS           any `json:"image.os"`
	ImageRelease      any `json:"image.release"`
	ImageSerial       any `json:"image.serial"`
	ImageType         any `json:"image.type"`
	ImageVersion      any `json:"image.version"`
}

type Device struct {
	Name    string `json:"name"`
	Network string `json:"network"`
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	Pool    string `json:"pool,omitempty"`
}

type ExecPayload struct {
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	User        int               `json:"user"`
	Group       int               `json:"group"`
}

var defaultExecEnv = map[string]string{
	"HOME": "/root",
	"LANG": "C.UTF-8",
	"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	"TERM": "xterm-256color",
	"USER": "root",
}

// InstancesHandler handles the synchronization request. It processes the HTTP request
// and sends the appropriate response back to the client.
func InstancesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path
	fmt.Println("Request Method:", r.Method, "|", "Request API:", path)
	parts := strings.Split(path, "/")
	var instanceName, instanceAction string
	var opType, opStatus, opSC, op, opEC, opE = "sync", "Success", 100, "", 0, ""
	var instanceData any
	var err error
	var requestData map[string]any
	if len(parts) == 4 {
		instanceName = parts[3]
	} else if len(parts) >= 4 {
		instanceName = parts[3]
		instanceAction = parts[4]
	}

	//recursion := r.URL.Query().Get("recursion")
	//project := r.URL.Query().Get("project")

	if len(r.TLS.PeerCertificates) > 0 {
		// 获取实例元数据
		if instanceName == "" && instanceAction == "" {
			instanceData, err = getInstanceInfo(instanceName)
		} else if instanceName != "" && instanceAction == "" {
			instanceData, err = getInstanceInfo(instanceName)
		} else if instanceName != "" && instanceAction == "forwards" {
			instanceData = []string{}
		} else if instanceName != "" && instanceAction == "state" && r.Method == http.MethodGet {
			instanceData, err = getInstanceInfo(instanceName)
		} else if instanceName != "" && instanceAction == "state" && r.Method == http.MethodPut {
			json.NewDecoder(r.Body).Decode(&requestData)
			action, _ := requestData["action"].(string)
			opType, opStatus, opSC, op, opEC, opE, instanceData, err = putInstanceAction(instanceName, action)
		} else if instanceName != "" && instanceAction == "exec" && r.Method == http.MethodPost {
			opType, opStatus, opSC, op, opEC, opE, instanceData, err = putInstanceExecAction(instanceName, r, false)
		} else if instanceName != "" && instanceAction == "console" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/octet-stream")
			data := []byte{0x54, 0x68, 0x69, 0x73, 0x20, 0x6c, 0x78, 0x63, 0x2d, 0x75, 0x69, 0x2d, 0x61, 0x70, 0x69, 0x21, 0x0a}
			w.Write(data)
			return
		} else if instanceName != "" && instanceAction == "console" && r.Method == http.MethodPost {
			opType, opStatus, opSC, op, opEC, opE, instanceData, err = putInstanceExecAction(instanceName, r, true)
		} else {
			instanceData, err = getNetworkInterfaceInfo(instanceName)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving network interfaces: %v", err), http.StatusInternalServerError)
			return
		}

		response := GeneralResponse{
			Type:       opType,
			Status:     opStatus,
			StatusCode: opSC,
			Operation:  op,
			ErrorCode:  opEC,
			Error:      opE,
			Metadata:   instanceData,
		}

		// 编码并写入响应
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "TLS certificate missing", http.StatusUnauthorized)
	}
}

func putInstanceAction(instanceName, action string) (string, string, int, string, int, string, any, error) {
	var cmd *exec.Cmd
	operationId := uuid.NewV4().String()
	description := charCases(action) + " instance"
	AddOperation(operationId, "task", "Running", instanceName, description, false)

	switch action {
	case "stop":
		cmd = exec.Command("sh", "-c", "lxc-unfreeze "+instanceName+" && lxc-stop "+instanceName)
	case "start":
		cmd = exec.Command("lxc-start", instanceName)
	case "restart":
		cmd = exec.Command("sh", "-c", "lxc-stop "+instanceName+" && lxc-start "+instanceName)
	case "freeze":
		cmd = exec.Command("lxc-freeze", instanceName)
	case "unfreeze":
		cmd = exec.Command("lxc-unfreeze", instanceName)
	default:
		return "", "Unsupported action", 400, "", 1, "Unsupported action", map[string]any{}, nil
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		SendInstanceResultToClient(operationId, instanceName, description, "Failure", 102)
		UpdateOperation(operationId, "Failure", "500")
		return "", "Failed to execute command", 500, "", 2, err.Error(), map[string]any{}, err
	}

	metadata := map[string]any{
		"id":          operationId,
		"class":       "task",
		"description": description,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
		"status":      "Running",
		"status_code": 103,
		"resources": map[string]any{
			"instances": []string{"/1.0/instances/" + instanceName},
		},
		"may_cancel": false,
		"err":        "",
		"location":   "none",
	}
	UpdateOperation(operationId, "Success", "")
	SendInstanceResultToClient(operationId, instanceName, description, "Success", 200)

	return "async", "Operation created", 100, "/1.0/operations/" + operationId, 0, "", metadata, nil
}

func putInstanceExecAction(instanceName string, r *http.Request, isConsole bool) (string, string, int, string, int, string, any, error) {
	operationId := uuid.NewV4().String()
	fdsData, _ := generateFds(128)
	fdsControl, _ := generateFds(128)
	body, _ := io.ReadAll(r.Body)
	finalEnv := make(map[string]string)
	defer r.Body.Close()
	var payload ExecPayload
	json.Unmarshal(body, &payload)
	maps.Copy(finalEnv, defaultExecEnv)
	maps.Copy(finalEnv, payload.Environment)

	AddOperation(operationId, "websocket", "Operation created", instanceName, "Executing command", isConsole)
	AddFds(operationId, fdsData, fdsControl, payload.Command, finalEnv, payload.User, payload.Group)

	metadata := map[string]any{
		"id":          operationId,
		"class":       "websocket",
		"description": "Executing command",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
		"status":      "Running",
		"status_code": 103,
		"resources": map[string]any{
			"instances": []string{"/1.0/instances/" + instanceName},
		},
		"may_cancel": false,
		"err":        "",
		"location":   "none",
		"metadata": MetadataInMetadata{
			Command:     payload.Command,
			Environment: finalEnv,
			Fds: map[string]string{
				"0":       fdsData,
				"control": fdsControl,
			},
			Interactive: true,
		},
	}
	UpdateOperation(operationId, "Success", "")
	SendInstanceAttachSessionCreatingResultToClient(operationId, instanceName, "Executing command", "Pending", 105, payload.Command, finalEnv)
	SendInstanceAttachSessionCreatingResultToClient(operationId, instanceName, "Executing command", "Running", 103, payload.Command, finalEnv)
	UpdateOperation(operationId, "Running", "")
	SendInstanceAttachSessionCreatedResultToClient(instanceName)

	return "async", "Operation created", 100, "/1.0/operations/" + operationId, 0, "", metadata, nil
}

func getInstanceInfo(instanceName string) (any, error) {
	cmd := exec.Command("lxc-ls", "-f")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		if instanceName != "" {
			return InstanceMetadata{}, err
		} else {
			return []InstanceMetadata{}, err
		}
	}

	lines := strings.Split(out.String(), "\n")
	var instances []InstanceMetadata
	var metadata InstanceMetadata

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		lxcState := parts[1]
		state := charCases(lxcState)

		cmdDetail := exec.Command("lxc-info", "-H", name)
		var outDetail bytes.Buffer
		cmdDetail.Stdout = &outDetail
		err := cmdDetail.Run()
		if err != nil {
			if instanceName != "" {
				return InstanceMetadata{}, err
			} else {
				return []InstanceMetadata{}, err
			}
		}

		instanceState := parseLxcInfo(outDetail.String())

		// Build InstanceMetadata
		instance := InstanceMetadata{
			Name:         name,
			Description:  "",
			Status:       state,
			StatusCode:   102,
			CreatedAt:    time.Now(),
			LastUsedAt:   time.Now(),
			Location:     "none",
			Type:         "container",
			Project:      "default",
			Architecture: nil,
			Ephemeral:    false,
			Stateful:     false,
			Profiles:     []string{"default"},
			Config: InstanceConfig{
				ImageArchitecture: nil,
				ImageDescription:  nil,
				ImageLabel:        nil,
				ImageOS:           nil,
				ImageRelease:      nil,
				ImageSerial:       nil,
				ImageType:         nil,
				ImageVersion:      nil,
			},
			Devices:        map[string]any{},
			ExpandedConfig: map[string]any{},
			ExpandedDevices: map[string]Device{
				"eth0": {
					Name:    "eth0",
					Network: "lxcbr0",
					Type:    "nic",
				},
				"root": {
					Path: "/",
					Pool: "default",
					Type: "disk",
				},
			},
			Backups:   nil,
			State:     instanceState,
			Snapshots: nil,
		}

		instances = append(instances, instance)

		if strings.TrimSpace(name) == strings.TrimSpace(instanceName) {
			metadata = instance
		}
	}

	if instanceName != "" {
		return metadata, nil
	} else {
		return instances, nil
	}
}

func parseLxcInfo(outDetail string) State {
	var state State
	state.Disk = make(map[string]map[string]int64)
	state.Memory = make(map[string]int64)
	state.Network = make(map[string]*NetworkInfo)
	state.Cpu = make(map[string]int64)
	state.Disk["root"] = make(map[string]int64)

	scanner := bufio.NewScanner(strings.NewReader(outDetail))
	var currentLink string
	var currentIps []string

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := fields[0]

		switch key {
		case "State:":
			state.Status = charCases(fields[1])
			state.StatusCode = 103
			state.Disk["root"]["usage"] = 0
			state.Disk["root"]["total"] = 0
			state.StartedAt = time.Now().UTC().Format(time.RFC3339)
			state.OsInfo = nil
		case "PID:":
			fmt.Sscanf(fields[1], "%d", &state.Pid)
		case "Processes:":
			fmt.Sscanf(fields[1], "%d", &state.Processes)
		case "Memory":
			if len(fields) >= 3 && fields[1] == "use:" {
				var usage int64
				fmt.Sscanf(fields[2], "%d", &usage)
				state.Memory["usage"] = usage
				state.Memory["usage_peak"] = 0
				state.Memory["total"] = 99
			}
		case "KMem":
			if len(fields) >= 3 && fields[1] == "use:" {
				var usage int64
				fmt.Sscanf(fields[2], "%d", &usage)
				state.Memory["swap_usage"] = usage
				state.Memory["swap_usage_peak"] = 0
			}
		case "CPU":
			if len(fields) >= 3 && fields[1] == "use:" {
				var cpuUsage int64
				fmt.Sscanf(fields[2], "%d", &cpuUsage)
				state.Cpu["usage"] = cpuUsage
				state.Cpu["allocated_time"] = 0
			}
		case "Link:":
			if currentLink == "" && len(currentIps) > 0 {
				var addresses []Address
				for _, ip := range currentIps {
					var family, netmask, scope string
					if strings.Contains(ip, ":") {
						family = "inet6"
						netmask = "64"
						scope = "global"
						if strings.HasPrefix(ip, "fe80") {
							scope = "link"
						}
					} else {
						family = "inet"
						netmask = "24"
						scope = "global"
					}
					addresses = append(addresses, Address{
						Family:  family,
						Address: ip,
						Netmask: netmask,
						Scope:   scope,
					})
				}
				state.Network[fields[1]] = &NetworkInfo{
					Counters:  make(map[string]int64),
					Addresses: addresses,
				}
			}
			currentLink = fields[1]
			currentIps = []string{}

		case "TX":
			if currentLink != "" {
				var txBytes int64
				fmt.Sscanf(fields[2], "%d", &txBytes)
				if _, exists := state.Network[currentLink]; !exists {
					state.Network[currentLink] = &NetworkInfo{
						Counters:  make(map[string]int64),
						Addresses: []Address{},
					}
				}
				state.Network[currentLink].Counters["bytes_sent"] = txBytes
			}
		case "RX":
			if currentLink != "" {
				var rxBytes int64
				state.Network[currentLink].Counters["bytes_received"] = rxBytes
			}
		case "IP:":
			currentIps = append(currentIps, fields[1])
		}
	}

	if currentLink != "" && len(currentIps) > 0 {
		var addresses []Address
		for _, ip := range currentIps {
			var family, netmask, scope string
			if strings.Contains(ip, ":") {
				family = "inet6"
				netmask = "64"
				scope = "global"
				if strings.HasPrefix(ip, "fe80") {
					scope = "link"
				}
			} else {
				family = "inet"
				netmask = "24"
				scope = "global"
			}
			addresses = append(addresses, Address{
				Family:  family,
				Address: ip,
				Netmask: netmask,
				Scope:   scope,
			})
		}

		if _, exists := state.Network[currentLink]; !exists {
			state.Network[currentLink] = &NetworkInfo{
				Counters:  make(map[string]int64),
				Addresses: []Address{},
			}
		}

		networkInfo := state.Network[currentLink]
		networkInfo.Addresses = addresses
	}
	return state
}
