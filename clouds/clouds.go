package clouds

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
)

// A Entity is either a Device, Sensor or Actuator.
type Entity struct {
	Device   string `json:"device,omitempty"`
	Sensor   string `json:"sensor,omitempty"`
	Actuator string `json:"actuator,omitempty"`
}

func (ent Entity) String() string {
	if ent.Sensor == "" {
		if ent.Actuator == "" {
			return "/devices/" + ent.Device
		}
		return "/devices/" + ent.Device + "/actuators/" + ent.Actuator
	}
	return "/devices/" + ent.Device + "/sensors/" + ent.Sensor
}

// A Action is performed by the cloud manager to resolve differences between the local stat and the cloud.
type Action int

const (
	// ActionError indicates that there was an error with this entity.
	ActionError Action = 1 << iota
	// ActionSync will synchronoice values from a sensor or actuator.
	ActionSync
	// ActionModify modifiy changes names and metadata.
	ActionModify
	// ActionCreate creates (declares) the device/sensor/actuator at the cloud.
	ActionCreate
)

// Status describes a single entity.
type Status struct {
	// Remote is the time the cloud is synced to.
	Remote time.Time `json:"remote"`
	// Action to perform.
	Action Action `json:"action"`

	// Wakeup Time
	Wakeup time.Time `json:"wakeup"`
	// Sleep durtion
	Sleep time.Duration `json:"sleep"`

	// Error, if any
	Error error `json:"error,omitempty"`
}

// NewStatus creates a new Status object.
func NewStatus(action Action, remote time.Time) *Status {
	return &Status{
		Remote: remote,
		Action: action,
		Wakeup: time.Now(),
		Sleep:  0,
	}
}

// Cloud represents a configuration to access a Waziup Cloud.
type Cloud struct {
	ID          string `json:"id"`
	Paused      bool   `json:"paused"`
	Pausing     bool   `json:"pausing"`
	PausingMQTT bool   `json:"pausing_mqtt"`
	REST        string `json:"rest"`
	MQTT        string `json:"mqtt"`

	Credentials struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	} `json:"credentials"`

	client    *mqtt.Client
	mqttMutex sync.Mutex
	devices   map[string]struct{}

	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`

	Status      map[Entity]*Status `json:"-"`
	statusMutex sync.Mutex

	sigDirty chan Entity
	auth     string
}

// Clouds lists all clouds that we synchronize.
// Changes must be made using CloudsMutex.
var clouds = map[string]*Cloud{}

// CloudsMutex guards Clouds.
var cloudsMutex sync.RWMutex

// GetCloud returns the Cloud with the given ID.
func GetCloud(id string) *Cloud {
	return clouds[id]
}

// GetClouds returns the cloud atlas.
func GetClouds() map[string]*Cloud {
	return clouds
}

var errCloudExists = errors.New("a cloud with that id already exists")

// AddCloud inserts the Cloud to the cloud atlas.
func AddCloud(cloud *Cloud) error {

	cloudsMutex.Lock()
	if _, exists := clouds[cloud.ID]; exists {
		cloudsMutex.Unlock()
		return errCloudExists
	}
	clouds[cloud.ID] = cloud
	cloudsMutex.Unlock()

	if !cloud.Paused {
		go cloud.sync()
	}
	return nil
}

// RemoveCloud pauses the cloud with that id and removes it from the cloud atlas.
func RemoveCloud(id string) bool {
	cloudsMutex.Lock()
	cloud := clouds[id]
	if cloud != nil {
		cloud.SetPaused(true)
	}
	delete(clouds, id)
	cloudsMutex.Unlock()
	return cloud != nil
}

// FlagDevice marks the device as dirty so that it will be synced wih the clouds.
func FlagDevice(deviceID string, action Action) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagDevice(deviceID, action)
	}
	cloudsMutex.RUnlock()
}

// FlagSensor marks the sensor as dirty so that it will be synced wih the clouds.
func FlagSensor(deviceID string, sensorID string, action Action, time time.Time) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagSensor(deviceID, sensorID, action, time)
	}
	cloudsMutex.RUnlock()
}

// FlagActuator marks the actuator as dirty so that it will be synced wih the clouds.
func FlagActuator(deviceID string, actuatorID string, action Action, time time.Time) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagActuator(deviceID, actuatorID, action, time)
	}
	cloudsMutex.RUnlock()
}

// FlagDevice marks the device as dirty.
func (cloud *Cloud) FlagDevice(deviceID string, action Action) {
	cloud.flag(Entity{deviceID, "", ""}, action, noTime)
}

// FlagSensor marks the sensor as dirty.
func (cloud *Cloud) FlagSensor(deviceID string, sensorID string, action Action, time time.Time) {
	cloud.flag(Entity{deviceID, sensorID, ""}, action, time)
}

// FlagActuator marks the actuator as dirty.
func (cloud *Cloud) FlagActuator(deviceID string, actuatorID string, action Action, time time.Time) {
	cloud.flag(Entity{deviceID, "", actuatorID}, action, time)
}

// ResetStatus clears the status field.
func (cloud *Cloud) ResetStatus() {
	cloud.statusMutex.Lock()
	cloud.Status = make(map[Entity]*Status)
	cloud.statusMutex.Unlock()
}

func (cloud *Cloud) Errorf(format string, code int, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	log.Printf("[UP   ] > [%3d] %s", code, str)
}

func (cloud *Cloud) Printf(format string, code int, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	log.Printf("[UP   ] > [%3d] %s", code, str)
}

func (cloud *Cloud) flag(ent Entity, action Action, remote time.Time) {
	var needsSig bool
	var status *Status
	cloud.statusMutex.Lock()
	if cloud.Status == nil {
		needsSig = false
	} else {
		needsSig = len(cloud.Status) == 0
		if status = cloud.Status[ent]; status != nil {
			if action == 0 {
				status.Remote = remote
				status.Wakeup = time.Now().Add(status.Sleep)
			} else if action < 0 {
				status.Action = status.Action ^ -action
				if status.Action == 0 {
					if status.Sleep == 0 {
						delete(cloud.Status, ent)
					} else {
						status.Wakeup = time.Now().Add(status.Sleep)
					}
				}
			} else {
				status.Action = status.Action | action
			}
		} else {
			status = NewStatus(action, remote)
			cloud.Status[ent] = status
		}
	}
	cloud.statusMutex.Unlock()

	log.Printf("[UP   ] Status %s: %s", ent, status.Action)

	if needsSig {
		select {
		case cloud.sigDirty <- ent:
		default: // channel full
		}
	}
}

func (cloud *Cloud) getRESTAddr() string {
	u, _ := url.Parse(cloud.REST)
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.RawQuery = ""
	u.Fragment = ""
	if strings.HasSuffix(u.RawPath, "/") {
		u.RawPath = u.RawPath[:len(u.RawPath)-1]
	}
	return u.String()
}

func (cloud *Cloud) getMQTTAddr() string {
	var u *url.URL
	if cloud.MQTT == "" {
		u, _ = url.Parse(cloud.REST)
	} else {
		u, _ = url.Parse(cloud.MQTT)
	}
	if u.Port() == "" {
		return u.Host + ":1883"
	}
	return u.Host
}

func (cloud *Cloud) setStatus(code int, text string) {
	text = strings.TrimSpace(text)
	log.Printf("[UP   ] [%d] %s", code, strings.ReplaceAll(text, "\n", " - "))
	cloud.StatusCode = code
	cloud.StatusText = text
}

var errCloudNoPause = errors.New("cloud pausing or not paused")

// SetCredentials changes the credentials.
func (cloud *Cloud) SetCredentials(username string, token string) (int, error) {

	if !cloud.Paused || cloud.Pausing {
		return http.StatusLocked, errCloudNoPause
	}
	addr := cloud.getRESTAddr()
	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, token}
	body, _ := json.Marshal(credentials)
	resp := fetch(addr+"/auth/token", fetchInit{
		method: http.MethodPost,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
		body: bytes.NewBuffer(body),
	})
	if resp.status == 0 {
		return http.StatusAccepted, nil
	}
	if !resp.ok {
		return resp.status, fmt.Errorf("can not login: server says: %v", resp.status)
	}

	return resp.status, nil
}

// ReadCloudConfig reads clouds.json into the current configuration.
func ReadCloudConfig(r io.Reader) error {
	cloudsMutex.Lock()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		cloudsMutex.Unlock()
		return err
	}
	err = json.Unmarshal(data, &clouds)
	if err == nil {
		for _, cloud := range clouds {
			if !cloud.Paused {
				go cloud.sync()
			}
		}
	} else {
		clouds = make(map[string]*Cloud)
	}
	cloudsMutex.Unlock()
	return err
}

// WriteCloudConfig writes the current configurations back to clouds.json.
func WriteCloudConfig(w io.Writer) error {
	cloudsMutex.RLock()
	encoder := json.NewEncoder(w)
	err := encoder.Encode(clouds)
	cloudsMutex.RUnlock()
	return err
}

// MarshalJSON implements json.Marshaler
func (a Action) MarshalJSON() ([]byte, error) {
	var astr [4]string
	str := astr[:0]
	if a&ActionCreate != 0 {
		str = append(str, "create")
	}
	if a&ActionModify != 0 {
		str = append(str, "modify")
	}
	if a&ActionSync != 0 {
		str = append(str, "sync")
	}
	if a&ActionError != 0 {
		str = append(str, "error")
	}
	return json.Marshal(str)
}

func (a Action) String() string {
	var astr [4]string
	str := astr[:0]
	if a&ActionCreate != 0 {
		str = append(str, "create")
	}
	if a&ActionModify != 0 {
		str = append(str, "modify")
	}
	if a&ActionSync != 0 {
		str = append(str, "sync")
	}
	if a&ActionError != 0 {
		str = append(str, "error")
	}
	return strings.Join(str, ", ")
}

// UnmarshalJSON implements json.Unmarshaler
func (a *Action) UnmarshalJSON(data []byte) error {
	var astr [4]string
	str := astr[:0]
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	*a = 0
	for _, s := range str {
		switch s {
		case "create":
			*a |= ActionCreate
		case "sync":
			*a |= ActionSync
		case "modify":
			*a |= ActionModify
		case "error":
			*a |= ActionError
		default:
			return fmt.Errorf("unknown action: %.12q", data)
		}
	}
	return nil
}
