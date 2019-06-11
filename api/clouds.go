package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/globalsign/mgo/bson"

	"github.com/Waziup/wazigate-edge/mqtt"

	routing "github.com/julienschmidt/httprouter"
)

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
}

// Clouds lists all clouds that we synchronize.
// Changes must be made using CloudsMutex.
var Clouds map[string]*Cloud

// CloudsMutex guards Clouds.
var CloudsMutex sync.RWMutex

func getCloudsFile() string {
	cloudsFile := os.Getenv("WAZIUP_CLOUDS_FILE")
	if cloudsFile == "" {
		return "clouds.json"
	}
	return cloudsFile
}

func (c *Cloud) getRESTAddr() string {
	u, _ := url.Parse(c.REST)
	u.Scheme = "https"
	u.RawQuery = ""
	u.Fragment = ""
	if strings.HasSuffix(u.RawPath, "/") {
		u.RawPath = u.RawPath[:len(u.RawPath)-1]
	}
	return u.String()
}

func (c *Cloud) getMQTTAddr() string {
	var u *url.URL
	if c.MQTT == "" {
		u, _ = url.Parse(c.REST)
	} else {
		u, _ = url.Parse(c.MQTT)
	}
	if u.Port() == "" {
		return u.Host + ":1883"
	}
	return u.Host
}

// ReadCloudConfig reads clouds.json into the current configuration.
func ReadCloudConfig() error {
	CloudsMutex.Lock()
	defer CloudsMutex.Unlock()

	config, err := ioutil.ReadFile(getCloudsFile())
	if err == nil {
		for _, cloud := range Clouds {
			cloud.endSync()
		}
		err = json.Unmarshal(config, &Clouds)
		if err == nil {
			log.Printf("[CLOUD] %d clouds from config:", len(Clouds))
			for _, cloud := range Clouds {
				cloud.Queue = mqtt.NewQueue(cloud.ID)
				log.Printf("[CLOUD] %q %q (pause:%v)", cloud.ID, cloud.REST, cloud.Paused)
				if !cloud.Paused {
					cloud.counter++
					go cloud.beginSync(cloud.counter)
				}
			}
		}
	}

	if err != nil {
		log.Println("[CLOUD] Can not read config:", err)
		Clouds = make(map[string]*Cloud)
	}

	return err
}

// WriteCloudConfig writes the current configurations back to clouds.json.
func WriteCloudConfig() error {
	CloudsMutex.RLock()
	defer CloudsMutex.RUnlock()

	data, _ := json.Marshal(Clouds)
	err := ioutil.WriteFile(getCloudsFile(), data, 0666)

	if err != nil {
		log.Println("[CLOUD] Can not write config:", err)
		Clouds = make(map[string]*Cloud)
	}

	return err
}

// GetClouds implements GET /clouds
func GetClouds(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.RLock()
	defer CloudsMutex.RUnlock()

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(Clouds)
	resp.Write(data)
}

// PostClouds implements POST /clouds
func PostClouds(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.Lock()

	cloud := &Cloud{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(cloud)
	if err != nil {
		CloudsMutex.Unlock()
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cloud.ID == "" {
		cloud.ID = bson.NewObjectId().Hex()
	}

	_, err = url.Parse(cloud.REST)
	if err != nil {
		CloudsMutex.Unlock()
		http.Error(resp, "Bad Request: Mal formatted REST address.", http.StatusBadRequest)
		return
	}

	if cloud.MQTT != "" {
		if _, err = url.Parse(cloud.MQTT); err != nil {
			CloudsMutex.Unlock()
			http.Error(resp, "Bad Request: Mal formatted MQTT address.", http.StatusBadRequest)
			return
		}
	}

	if _, exists := Clouds[cloud.ID]; exists {
		CloudsMutex.Unlock()
		http.Error(resp, "Bad Request: A cloud with that ID already exists.", http.StatusBadRequest)
		return
	}
	mqtt.DeleteQueue(cloud.ID)
	cloud.Queue = mqtt.NewQueue(cloud.ID)
	Clouds[cloud.ID] = cloud
	log.Printf("[CLOUD] Created %q: %q", cloud.ID, cloud.REST)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'"'})
	resp.Write([]byte(cloud.ID))
	resp.Write([]byte{'"'})

	CloudsMutex.Unlock()

	if !cloud.Paused {
		cloud.counter++
		go cloud.beginSync(cloud.counter)
	}

	WriteCloudConfig()
}

// DeleteCloud implements DELETE /clouds/{cloudID}
func DeleteCloud(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.Lock()

	cloudID := params.ByName("cloud_id")
	cloud, exists := Clouds[cloudID]
	if !exists {
		CloudsMutex.Unlock()
		http.Error(resp, "Not Found: There is no cloud with that ID.", http.StatusNotFound)
		return
	}

	delete(Clouds, cloudID)
	log.Printf("[CLOUD] Deleted %q: %q", cloud.ID, cloud.REST)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte("true"))

	CloudsMutex.Unlock()

	cloud.Paused = true
	cloud.endSync()

	WriteCloudConfig()
}

// GetCloud implements GET /clouds/{cloudID}/config
func GetCloud(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.RLock()
	defer CloudsMutex.RUnlock()

	resp.Header().Set("Content-Type", "application/json")

	cloudID := params.ByName("cloud_id")
	cloud, exists := Clouds[cloudID]

	if !exists {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(cloud)
	resp.Write(data)
}

// PostCloudRESTAddr implements POST /clouds/{cloudID}/rest
func PostCloudRESTAddr(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.Lock()

	cloudID := params.ByName("cloud_id")
	cloud, exists := Clouds[cloudID]

	if !exists {
		CloudsMutex.Unlock()
		http.Error(resp, "Not Found: There is no cloud with that ID.", http.StatusNotFound)
		return
	}

	var addr string
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&addr)
	if err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CLOUD] Changed REST Addr %q", cloud.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte("true"))

	CloudsMutex.Unlock()

	if !cloud.Paused {
		cloud.endSync()
		cloud.counter++
		cloud.REST = addr
		go cloud.beginSync(cloud.counter)
	} else {
		cloud.REST = addr
	}

	WriteCloudConfig()
}

// PostCloudMQTTAddr implements POST /clouds/{cloudID}/mqtt
func PostCloudMQTTAddr(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.Lock()

	cloudID := params.ByName("cloud_id")
	cloud, exists := Clouds[cloudID]

	if !exists {
		CloudsMutex.Unlock()
		http.Error(resp, "Not Found: There is no cloud with that ID.", http.StatusNotFound)
		return
	}

	var addr string
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&addr)
	if err != nil {
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CLOUD] Changed MQTT Addr %q", cloud.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte("true"))

	CloudsMutex.Unlock()

	if !cloud.Paused {
		cloud.endSync()
		cloud.counter++
		cloud.MQTT = addr
		go cloud.beginSync(cloud.counter)
	} else {
		cloud.MQTT = addr
	}

	WriteCloudConfig()
}

// PostCloudCredentials implements POST /clouds/{cloudID}/credentials
func PostCloudCredentials(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.Lock()

	cloudID := params.ByName("cloud_id")
	cloud, exists := Clouds[cloudID]

	if !exists {
		CloudsMutex.Unlock()
		http.Error(resp, "Not Found: There is no cloud with that ID.", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&cloud.Credentials)
	if err != nil {
		CloudsMutex.Unlock()
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CLOUD] Changed credentials %q", cloud.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte("true"))

	CloudsMutex.Unlock()

	if !cloud.Paused {
		cloud.endSync()
		cloud.counter++
		go cloud.beginSync(cloud.counter)
	}

	WriteCloudConfig()
}

// PostCloudPaused implements POST /clouds/{cloudID}/paused
func PostCloudPaused(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	CloudsMutex.Lock()

	cloudID := params.ByName("cloud_id")
	cloud, exists := Clouds[cloudID]

	if !exists {
		CloudsMutex.Unlock()
		http.Error(resp, "Not Found: There is no cloud with that ID.", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&cloud.Paused)
	if err != nil {
		CloudsMutex.Unlock()
		http.Error(resp, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CLOUD] Changed paused %q", cloud.ID)

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte("true"))

	CloudsMutex.Unlock()

	if !cloud.Paused {
		cloud.endSync()
		cloud.counter++
		go cloud.beginSync(cloud.counter)
	}

	WriteCloudConfig()
}
