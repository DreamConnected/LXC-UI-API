package lxcapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var globalConn *websocket.Conn
var globalConnTerminal *websocket.Conn

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Do not check the source
		return true
	},
}

type Resources struct {
	Instances []string `json:"instances"`
}

type OperationMetadata struct {
	// operation
	ID          string    `json:"id"`
	Class       string    `json:"class"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
	StatusCode  int       `json:"status_code"`
	Resources   struct {
		Instances []string `json:"instances"`
	} `json:"resources"`
	Metadata  MetadataInMetadata `json:"metadata"`
	MayCancel bool               `json:"may_cancel"`
	Err       string             `json:"err"`
	Location  string             `json:"location"`
}

type LifecycleMetadata struct {
	// lifecycle
	Action  string `json:"action"`
	Source  string `json:"source"`
	Context struct {
		Command []string `json:"command"`
	} `json:"context"`
	Requestor struct {
		Username string `json:"username"`
		Protocol string `json:"protocol"`
		Address  string `json:"address"`
	} `json:"requestor"`
	Name    string `json:"name"`
	Project string `json:"project"`
}

type MetadataInMetadata struct {
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	Fds         map[string]string `json:"fds"`
	Interactive bool              `json:"interactive"`
}

type LifecycleResponse struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  LifecycleMetadata `json:"metadata"`
	Location  string            `json:"location"`
	Project   string            `json:"project"`
}

type OperationResponse struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  OperationMetadata `json:"metadata"`
	Location  string            `json:"location"`
	Project   string            `json:"project"`
}

// /1.0/operations/{opID}/websocket?secret={0}/{control}
func HandleOperationsWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	globalConn = conn
	log.Println("WebSocket connection established")

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
			break
		}

		log.Printf("Received: %s\n", p)

		// Control doesn't seem to require a response, but just in case, a response was added
		if err := conn.WriteMessage(messageType, []byte("Acknowledged")); err != nil {
			log.Println("Error sending message:", err)
			break
		}
	}
}

func HandleOperationsWebSocketTerminal(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	secret := r.URL.Query().Get("secret")
	var operationID string
	fmt.Println("Request Method:", r.Method, "|", "Request API:", path+"?secret="+secret)
	if len(parts) == 4 {
		operationID = parts[3]
	} else if len(parts) >= 4 {
		operationID = parts[3]
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	fds, _ := GetFds(operationID)
	operation, _ := GetOperation(operationID)
	globalConnTerminal = conn

	if fds.Data == secret && !operation.IsConsole {
		log.Println("WebSocket Terminal Data connection established")
		args := []string{operation.Instances}
		for key, value := range fds.Environment {
			args = append(args, "-v", fmt.Sprintf("%s=%s", key, value))
		}
		args = append(args, "-u", fmt.Sprint(fds.User), "-g", fmt.Sprint(fds.Group), "--clear-env", "--", "bin/"+fds.Command[0])
		cmd := exec.Command("lxc-attach", args...)
		log.Println(cmd)
		ptmx, err := pty.Start(cmd)
		if err != nil {
			UpdateOperation(operationID, "Failure", err.Error())
			return
		}
		defer func() {
			ptmx.Close()
			cmd.Process.Kill()
		}()

		// forward to WebSocket
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := ptmx.Read(buf)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						UpdateOperation(operationID, "Seccess", err.Error())
						return
					}
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						UpdateOperation(operationID, "Failure", err.Error())
						return
					}
				}
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				UpdateOperation(operationID, "Failure", err.Error())
				break
			}
			_, err = ptmx.Write(msg)
			if err != nil {
				UpdateOperation(operationID, "Failure", err.Error())
				break
			}
		}
	} else if fds.Data == secret && operation.IsConsole {
		log.Println("WebSocket Console Data connection established")
		cmd := exec.Command("lxc-console", operation.Instances)
		log.Println(cmd)
		ptmx, err := pty.Start(cmd)
		if err != nil {
			UpdateOperation(operationID, "Failure", err.Error())
			return
		}
		defer func() {
			ptmx.Close()
			cmd.Process.Kill()
		}()

		// forward to WebSocket
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := ptmx.Read(buf)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						UpdateOperation(operationID, "Seccess", err.Error())
						return
					}
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						UpdateOperation(operationID, "Failure", err.Error())
						return
					}
				}
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				UpdateOperation(operationID, "Failure", err.Error())
				break
			}
			_, err = ptmx.Write(msg)
			if err != nil {
				UpdateOperation(operationID, "Failure", err.Error())
				break
			}
		}
	} else if fds.Control == secret {
		log.Println("WebSocket Terminal Control connection established")
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					UpdateOperation(operationID, "Seccess", err.Error())
					return
				}
				break
			}

			log.Printf("Received: %s\n", p)

			if err := conn.WriteMessage(messageType, []byte("Acknowledged")); err != nil {
				log.Println("Error sending message:", err)
				break
			}
		}
	}

}

func SendInstanceResultToClient(operationId, instanceName, description, status string, statusCode int) error {
	if globalConn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	message := OperationResponse{
		Type:      "operation",
		Timestamp: time.Now(),
		Metadata: OperationMetadata{
			ID:          operationId,
			Class:       "task",
			Description: description,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Status:      status,
			StatusCode:  statusCode,
			Resources: Resources{
				Instances: []string{"/1.0/instances/" + instanceName},
			},
			Metadata:  MetadataInMetadata{},
			MayCancel: false,
			Err:       "",
			Location:  "none",
		},
		Location: "none",
		Project:  "default",
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}

	if err := globalConn.WriteMessage(websocket.TextMessage, messageData); err != nil {
		log.Println("Error sending message:", err)
		return err
	}

	log.Printf("Sent: %s\n", messageData)

	return nil
}

func SendInstanceAttachSessionCreatingResultToClient(operationId, instanceName, description, status string, statusCode int, command []string, env map[string]string) error {
	if globalConn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	fds, _ := GetFds(operationId)

	message := OperationResponse{
		Type:      "operation",
		Timestamp: time.Now().UTC(),
		Metadata: OperationMetadata{
			ID:          operationId,
			Class:       "websocket",
			Description: description,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Status:      status,
			StatusCode:  statusCode,
			Resources: Resources{
				Instances: []string{"/1.0/instances/" + instanceName},
			},
			Metadata: MetadataInMetadata{
				Command:     command,
				Environment: env,
				Fds: map[string]string{
					"0":       fds.Data,
					"control": fds.Control,
				},
				Interactive: true,
			},
			MayCancel: false,
			Err:       "",
			Location:  "none",
		},
		Location: "none",
		Project:  "default",
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}

	if err := globalConn.WriteMessage(websocket.TextMessage, messageData); err != nil {
		log.Println("Error sending message:", err)
		return err
	}

	log.Printf("Sent: %s\n", messageData)

	return nil
}

func SendInstanceAttachSessionCreatedResultToClient(instanceName string) error {
	if globalConn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	message := LifecycleResponse{
		Type:      "lifecycle",
		Timestamp: time.Now().UTC(),
		Metadata: LifecycleMetadata{
			Action: "instance-exec",
			Source: "/1.0/instances/" + instanceName,
			Context: struct {
				Command []string `json:"command"`
			}{
				Command: []string{"sh"},
			},
			Requestor: struct {
				Username string `json:"username"`
				Protocol string `json:"protocol"`
				Address  string `json:"address"`
			}{
				// If the auth method is TLS, this should be the certificate fingerprint.
				Username: "fff8465939e4813ea04338b40191f663d85e518aca8c5eb3661219bfdd325dea",
				Protocol: "tls",
				Address:  "0.0.0.0",
			},
			Name:    instanceName,
			Project: "default",
		},
		Location: "none",
		Project:  "default",
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return err
	}

	if err := globalConn.WriteMessage(websocket.TextMessage, messageData); err != nil {
		log.Println("Error sending message:", err)
		return err
	}

	log.Printf("Sent: %s\n", messageData)

	return nil
}
