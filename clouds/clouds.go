package clouds

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Waziup/wazigate-edge/mqtt"
)

type entity struct {
	deviceID   string
	sensorID   string
	actuatorID string
}

type remote struct {
	time   time.Time
	exists bool
}

// Cloud represents a configuration to access a Waziup Cloud.
type Cloud struct {
	ID      string `json:"id"`
	Paused  bool   `json:"paused"`
	Pausing bool   `json:"pausing"`
	REST    string `json:"rest"`
	MQTT    string `json:"mqtt"`

	Credentials struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	} `json:"credentials"`

	client *mqtt.Client

	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`

	remote      map[entity]*remote
	remoteMutex sync.Mutex
	sigDirty    chan struct{}
	auth        string
}

// Clouds lists all clouds that we synchronize.
// Changes must be made using CloudsMutex.
var clouds map[string]*Cloud

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
		cloud.SetPaused(false)
	}
	delete(clouds, id)
	cloudsMutex.Unlock()
	return cloud != nil
}

// FlagDevice marks the device as dirty so that it will be synced wih the clouds.
func FlagDevice(deviceID string) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagDevice(deviceID)
	}
	cloudsMutex.RUnlock()
}

// FlagSensor marks the sensor as dirty so that it will be synced wih the clouds.
func FlagSensor(deviceID string, sensorID string, time time.Time) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagSensor(deviceID, sensorID, time)
	}
	cloudsMutex.RUnlock()
}

// FlagActuator marks the actuator as dirty so that it will be synced wih the clouds.
func FlagActuator(deviceID string, actuatorID string, time time.Time) {
	if len(clouds) == 0 {
		return
	}
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.FlagActuator(deviceID, actuatorID, time)
	}
	cloudsMutex.RUnlock()
}

// FlagDevice marks the device as dirty.
func (cloud *Cloud) FlagDevice(deviceID string) {
	cloud.flag(entity{deviceID, "", ""}, noTime)
}

// FlagSensor marks the sensor as dirty.
func (cloud *Cloud) FlagSensor(deviceID string, sensorID string, time time.Time) {
	cloud.flag(entity{deviceID, sensorID, ""}, time)
}

// FlagActuator marks the actuator as dirty.
func (cloud *Cloud) FlagActuator(deviceID string, actuatorID string, time time.Time) {
	cloud.flag(entity{deviceID, "", actuatorID}, time)
}

func (cloud *Cloud) flag(ent entity, time time.Time) {

	cloud.remoteMutex.Lock()
	empty := len(cloud.remote) == 0
	if ent.sensorID == "" && ent.actuatorID == "" {
		cloud.remote[ent] = &remote{time, false}
	} else {
		deviceEnt := entity{ent.deviceID, "", ""}
		if cloud.remote[deviceEnt] == nil {
			cloud.remote[ent] = &remote{time, time != noTime}
		}
	}
	cloud.remoteMutex.Unlock()

	if empty {
		select {
		case cloud.sigDirty <- struct{}{}:
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

// ReadCloudConfig reads clouds.json into the current configuration.
func ReadCloudConfig(r io.Reader) error {
	cloudsMutex.Lock()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		cloudsMutex.Unlock()
		return err
	}
	err = json.Unmarshal(data, &clouds)
	if err != nil {
		clouds = make(map[string]*Cloud)
	}
	for _, cloud := range clouds {
		if !cloud.Paused {
			go cloud.sync()
		}
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
