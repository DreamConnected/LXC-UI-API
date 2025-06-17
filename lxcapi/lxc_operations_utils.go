package lxcapi

import (
	"fmt"
	"sync"
	"time"
)

type Operation struct {
	ID          string    `json:"id"`
	Class       string    `json:"class"`
	Status      string    `json:"status"`
	Instances   string    `json:"instances"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Err         string    `json:"err"`
	IsConsole   bool
}

type Fds struct {
	ID          string            `json:"id"`
	Data        string            `json:"0"`
	Control     string            `json:"control"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	User        int               `json:"user"`
	Group       int               `json:"group"`
}

var Operations = make(map[string]*Operation)
var Fdses = make(map[string]*Fds)
var mu, muFds sync.Mutex

func AddOperation(operationID, operationClass, status, instanceName, description string, isConsole bool) {
	mu.Lock()
	defer mu.Unlock()

	operation := &Operation{
		ID:          operationID,
		Class:       operationClass,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Err:         "",
		Instances:   instanceName,
		Description: description,
		IsConsole:   isConsole,
	}

	Operations[operationID] = operation
}

func DeleteOperation(operationID string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := Operations[operationID]; exists {
		delete(Operations, operationID)
		return nil
	}
	return fmt.Errorf("operation with ID %s not found", operationID)
}

func UpdateOperation(operationID, status, err string) error {
	mu.Lock()
	defer mu.Unlock()

	if operation, exists := Operations[operationID]; exists {
		operation.Status = status
		operation.Err = err
		operation.UpdatedAt = time.Now()
		return nil
	}
	return fmt.Errorf("operation with ID %s not found", operationID)
}

func GetOperation(operationID string) (*Operation, error) {
	mu.Lock()
	defer mu.Unlock()

	if operation, exists := Operations[operationID]; exists {
		return operation, nil
	}
	return nil, fmt.Errorf("operation with ID %s not found", operationID)
}

func ListOperations() ([]*Operation, error) {
	mu.Lock()
	defer mu.Unlock()

	var operationsList []*Operation
	for _, operation := range Operations {
		operationsList = append(operationsList, operation)
	}

	if len(operationsList) == 0 {
		return nil, fmt.Errorf("no operations found")
	}

	return operationsList, nil
}

// Fds
func AddFds(operationID, data, control string, command []string, env map[string]string, u, g int) {
	muFds.Lock()
	defer muFds.Unlock()

	fds := &Fds{
		ID:          operationID,
		Data:        data,
		Control:     control,
		Command:     command,
		Environment: env,
		User:        u,
		Group:       g,
	}

	Fdses[operationID] = fds
}

func DeleteFds(operationID string) error {
	muFds.Lock()
	defer muFds.Unlock()

	if _, exists := Fdses[operationID]; exists {
		delete(Fdses, operationID)
		return nil
	}
	return fmt.Errorf("fds operation with ID %s not found", operationID)
}

func GetFds(operationID string) (*Fds, error) {
	muFds.Lock()
	defer muFds.Unlock()

	if fds, exists := Fdses[operationID]; exists {
		return fds, nil
	}
	return nil, fmt.Errorf("operation with ID %s not found", operationID)
}

func ListFdses() ([]*Fds, error) {
	muFds.Lock()
	defer muFds.Unlock()

	var fdsesList []*Fds
	for _, fds := range Fdses {
		fdsesList = append(fdsesList, fds)
	}

	if len(fdsesList) == 0 {
		return nil, fmt.Errorf("no fdses found")
	}

	return fdsesList, nil
}
