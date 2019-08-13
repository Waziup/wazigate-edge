package clouds

import (
	"encoding/json"
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
	deviceID, sensorID string
}

type remote struct {
	time   time.Time
	exists bool
}

// Cloud represents a configuration to access a Waziup Cloud.
type Cloud struct {
	ID     string `json:"id"`
	Paused bool   `json:"paused"`
	REST   string `json:"rest"`
	MQTT   string `json:"mqtt"`

	Credentials struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	} `json:"credentials"`

	counter int
	Client  *mqtt.Client `json:"-"`
	Queue   *mqtt.Queue  `json:"queue"`

	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`

	remote      map[entity]*remote
	remoteMutex sync.Mutex
	sigDirty    chan struct{}
	auth        string
	pausing     bool
}

// Clouds lists all clouds that we synchronize.
// Changes must be made using CloudsMutex.
var clouds map[string]*Cloud

// CloudsMutex guards Clouds.
var cloudsMutex sync.RWMutex

func Flag(device string, sensor string, time time.Time) {
	cloudsMutex.RLock()
	for _, cloud := range clouds {
		cloud.Flag(device, sensor, time)
	}
	cloudsMutex.RUnlock()
}

func (cloud *Cloud) Flag(device string, sensor string, time time.Time) {

	cloud.remoteMutex.Lock()

	empty := len(cloud.remote) == 0
	if cloud.remote[entity{device, ""}] == nil {
		if cloud.remote[entity{device, sensor}] == nil {
			cloud.remote[entity{device, sensor}] = &remote{noTime, false}
		}
	}
	cloud.remoteMutex.Unlock()

	if empty {
		cloud.sigDirty <- struct{}{}
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
	log.Printf("[UP   ] Cloud Status: [%d] %s", code, strings.ReplaceAll(text, "\n", " - "))
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
