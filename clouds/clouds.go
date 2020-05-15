package clouds

import (
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

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/mqtt"
)

// MaxCloudEvents is the number of cloud events to keep in memory.
var MaxCloudEvents = 10

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
	// ActionNoSync indicates that this entity should not be synced.
	ActionNoSync
	// ActionSync will synchronoice values from a sensor or actuator.
	ActionSync
	// ActionModify modifiy changes names and metadata.
	ActionModify
	// ActionCreate creates (declares) the device/sensor/actuator at the cloud.
	ActionCreate
	// ActionDelete delete the device/sensor/actuator
	ActionDelete
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

// Event repesents cloud events.
type Event struct {
	Code    int       `json:"code"`
	Message string    `json:"msg"`
	Time    time.Time `json:"time"`
}

// Cloud represents a configuration to access a Waziup Cloud.
type Cloud struct {
	ID          string `json:"id"`
	Paused      bool   `json:"paused"`
	Pausing     bool   `json:"pausing"`
	PausingMQTT bool   `json:"pausing_mqtt"`
	REST        string `json:"rest"`
	MQTT        string `json:"mqtt"`

	Registered bool `json:"registered"`

	Events []Event `json:"events"`

	Username string `json:"username"`
	Token    string `json:"token"`

	client    *mqtt.Client
	mqttMutex sync.Mutex
	devices   map[string]struct{}

	Status      map[Entity]*Status `json:"-"`
	StatusMutex sync.Mutex         `json:"-"`

	wakeup chan struct{}
	auth   string
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
func FlagDevice(deviceID string, action Action, meta edge.Meta) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagDevice(deviceID, action, meta)
	}
	cloudsMutex.RUnlock()
}

// FlagSensor marks the sensor as dirty so that it will be synced wih the clouds.
func FlagSensor(deviceID string, sensorID string, action Action, time time.Time, meta edge.Meta) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagSensor(deviceID, sensorID, action, time, meta)
	}
	cloudsMutex.RUnlock()
}

// FlagActuator marks the actuator as dirty so that it will be synced wih the clouds.
func FlagActuator(deviceID string, actuatorID string, action Action, time time.Time, meta edge.Meta) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagActuator(deviceID, actuatorID, action, time, meta)
	}
	cloudsMutex.RUnlock()
}

// FlagDevice marks the device as dirty.
func (cloud *Cloud) FlagDevice(deviceID string, action Action, meta edge.Meta) {
	cloud.flag(Entity{deviceID, "", ""}, action, noTime, meta)
	cloud.Wakeup()
}

// FlagSensor marks the sensor as dirty.
func (cloud *Cloud) FlagSensor(deviceID string, sensorID string, action Action, time time.Time, meta edge.Meta) {
	cloud.flag(Entity{deviceID, sensorID, ""}, action, time, meta)
	cloud.Wakeup()
}

// FlagActuator marks the actuator as dirty.
func (cloud *Cloud) FlagActuator(deviceID string, actuatorID string, action Action, time time.Time, meta edge.Meta) {
	cloud.flag(Entity{deviceID, "", actuatorID}, action, time, meta)
	cloud.Wakeup()
}

// ResetStatus clears the status field.
func (cloud *Cloud) ResetStatus() {
	cloud.StatusMutex.Lock()
	cloud.Status = make(map[Entity]*Status)
	cloud.StatusMutex.Unlock()
}

// Printf logs some events for this cloud.
func (cloud *Cloud) Printf(format string, code int, a ...interface{}) {
	event := Event{
		Code:    code,
		Message: fmt.Sprintf(format, a...),
		Time:    time.Now(),
	}
	log.Printf("[UP   ] (%3d) %s", code, event.Message)

	if len(cloud.Events) == MaxCloudEvents {
		cloud.Events = append(cloud.Events[:0], cloud.Events[1:]...)
		cloud.Events = append(cloud.Events, event)
	} else {
		cloud.Events = append(cloud.Events, event)
	}

	if eventCallback != nil {
		eventCallback(cloud, event)
	}
}

func (cloud *Cloud) flag(ent Entity, action Action, remote time.Time, meta edge.Meta) {
	var status *Status
	now := time.Now()
	cloud.StatusMutex.Lock()
	if cloud.Status != nil {
		if status = cloud.Status[ent]; status != nil {
			if action == -ActionDelete {
				delete(cloud.Status, ent)
			} else if action == 0 {
				status.Wakeup = now.Add(status.Sleep)
				status.Remote = remote
			} else {
				if action < 0 {
					status.Action = status.Action ^ -action
					// if action == -ActionSync {
					// 	status.Wakeup = now.Add(status.Sleep)
					// }
					if status.Action == 0 {
						if !now.Before(status.Wakeup) {
							delete(cloud.Status, ent)
							status = nil
						}
					}
				} else {
					if action == ActionModify {
						sleep := meta.SyncInterval()
						status.Wakeup = status.Wakeup.Add(sleep - status.Sleep)
						status.Sleep = sleep
					}
					status.Action = status.Action | action
				}
			}
		} else {
			sleep := meta.SyncInterval()
			status = &Status{
				Remote: remote,
				Action: action,
				Wakeup: time.Now(),
				Sleep:  sleep,
			}
			cloud.Status[ent] = status
		}
	}
	cloud.StatusMutex.Unlock()

	if status == nil {
		log.Printf("[UP   ] Status %q: released", ent)
	} else {
		log.Printf("[UP   ] Status %q: %s", ent, status.Action)
	}
	if statusCallback != nil {
		statusCallback(cloud, ent, status)
	}
}

func (cloud *Cloud) Wakeup() {
	select {
	case cloud.wakeup <- struct{}{}:
	default: // channel full
	}
}

// Meta repesents synchronization instructions based on entity metadata (`.meta` fields).
type Meta struct {
	NoSync       bool
	SyncInterval time.Duration
}

// NewMeta reads the json object and exracts a Meta object.
func NewMeta(json map[string]interface{}) (meta Meta) {
	if json != nil {
		if m := json["syncInterval"]; m != nil {
			switch i := m.(type) {
			case string:
				if j, err := time.ParseDuration(i); err == nil {
					meta.SyncInterval = j
				}
			}
		}
		if m := json["doNotSync"]; m != nil {
			switch i := m.(type) {
			case bool:
				meta.NoSync = i
			case int:
				meta.NoSync = i != 0
			}
		}
	}
	return
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

var errCloudNoPause = errors.New("cloud pausing or not paused")

// SetUsername changes the uswername that is used for authentication with the waziup cloud.
func (cloud *Cloud) SetUsername(username string) (int, error) {
	if !cloud.Paused || cloud.Pausing {
		return http.StatusLocked, errCloudNoPause
	}
	cloud.Username = username
	return 200, nil
}

// SetToken changes the token (password) that is used for authentication with the waziup cloud.
func (cloud *Cloud) SetToken(token string) (int, error) {
	if !cloud.Paused || cloud.Pausing {
		return http.StatusLocked, errCloudNoPause
	}
	cloud.Token = token
	return 200, nil
}

////////////////////////////////////////////////////////////////////////////////

// StatusCallback is called when a cloud updates its status.
type StatusCallback func(cloud *Cloud, ent Entity, status *Status)

var statusCallback StatusCallback

// OnStatus sets the global StatusCallback handler.
func OnStatus(cb StatusCallback) {
	statusCallback = cb
}

// EventCallback is called when a cloud changes its state.
type EventCallback func(cloud *Cloud, event Event)

var eventCallback EventCallback

// OnEvent sets the global EventCallback handler.
func OnEvent(cb EventCallback) {
	eventCallback = cb
}

// Downstream handles incomming MQTT packets.
type Downstream interface {
	Publish(sender mqtt.Sender, msg *mqtt.Message) int
}

var downstream Downstream = nil

// SetDownstream changes the global Downstream handler.
func SetDownstream(ds Downstream) {
	downstream = ds
}

////////////////////////////////////////////////////////////////////////////////

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
			cloud.Pausing = false
			cloud.PausingMQTT = false
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

////////////////////////////////////////////////////////////////////////////////

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
