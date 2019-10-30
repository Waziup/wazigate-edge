package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/globalsign/mgo/bson"

	routing "github.com/julienschmidt/httprouter"
)

// GetClouds implements GET /clouds
func GetClouds(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	data, err := json.Marshal(clouds.GetClouds())
	if err != nil {
		log.Printf("[ERR  ] Error %v", err)
		http.Error(resp, "internal server error", http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.Write(data)
}

// PostClouds implements POST /clouds
func PostClouds(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloud := &clouds.Cloud{}
	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(cloud); err != nil {
		http.Error(resp, "bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if cloud.ID == "" {
		cloud.ID = bson.NewObjectId().Hex()
	}

	if _, err := url.Parse(cloud.REST); err != nil {
		http.Error(resp, "bad request: mal formatted REST address", http.StatusBadRequest)
		return
	}

	if cloud.MQTT != "" {
		if _, err := url.Parse(cloud.MQTT); err != nil {
			http.Error(resp, "bad request: mal formatted MQTT address", http.StatusBadRequest)
			return
		}
	}

	if err := clouds.AddCloud(cloud); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CLOUD] Created %q.", cloud.ID)

	writeCloudFile()
	resp.Write([]byte(cloud.ID))
}

// DeleteCloud implements DELETE /clouds/{cloudID}
func DeleteCloud(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	if !clouds.RemoveCloud(cloudID) {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	log.Printf("[CLOUD] Deleted.")
	writeCloudFile()
}

// GetCloud implements GET /clouds/{cloudID}
func GetCloud(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "no found: no cloud with that id", http.StatusNotFound)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(cloud)
	resp.Write(data)
}

// PostCloudRESTAddr implements POST /clouds/{cloudID}/rest
func PostCloudRESTAddr(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "no found: no cloud with that id", http.StatusNotFound)
		return
	}

	if !cloud.Paused || cloud.Pausing {
		http.Error(resp, "bad request: cloud is paused or pausing", http.StatusBadRequest)
		return
	}

	var addr string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&addr); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := url.Parse(addr); err != nil {
		http.Error(resp, "bad request: mal formatted address", http.StatusBadRequest)
		return
	}

	cloud.REST = addr
	log.Printf("[CLOUD] Changed REST addr %q", cloud.REST)
	writeCloudFile()
}

// PostCloudMQTTAddr implements POST /clouds/{cloudID}/mqtt
func PostCloudMQTTAddr(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "no found: no cloud with that id", http.StatusNotFound)
		return
	}

	if !cloud.Paused || cloud.Pausing {
		http.Error(resp, "bad request: cloud is paused or pausing", http.StatusBadRequest)
		return
	}

	var addr string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&addr); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if addr != "" {
		if _, err := url.Parse(addr); err != nil {
			http.Error(resp, "bad request: mal formatted address", http.StatusBadRequest)
			return
		}
	}

	cloud.MQTT = addr
	log.Printf("[CLOUD] Changed MQTT addr %q", cloud.MQTT)
	writeCloudFile()
}

// PostCloudUsername implements POST /clouds/{cloudID}/username
func PostCloudUsername(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "no found: no cloud with that id", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(req.Body)
	var username string
	err := decoder.Decode(&username)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	status, err := cloud.SetUsername(username)
	if err != nil {
		http.Error(resp, err.Error(), status)
		return
	}
	resp.WriteHeader(status)
	writeCloudFile()
}

// PostCloudToken implements POST /clouds/{cloudID}/token
func PostCloudToken(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "no found: no cloud with that id", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(req.Body)
	var token string
	err := decoder.Decode(&token)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	status, err := cloud.SetToken(token)
	if err != nil {
		http.Error(resp, err.Error(), status)
		return
	}
	resp.WriteHeader(status)
	writeCloudFile()
}

// PostCloudPaused implements POST /clouds/{cloudID}/paused
func PostCloudPaused(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "no found: no cloud with that id", http.StatusNotFound)
		return
	}

	var paused bool
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&paused)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	status, err := cloud.SetPaused(paused)
	resp.WriteHeader(status)
	if err != nil {
		resp.Write([]byte(err.Error()))
	}
	writeCloudFile()
}

////////////////////////////////////////////////////////////////////////////////

func getCloudsFile() string {
	cloudsFile := os.Getenv("WAZIUP_CLOUDS_FILE")
	if cloudsFile == "" {
		return "clouds.json"
	}
	return cloudsFile
}

func writeCloudFile() {
	cloudsFile := getCloudsFile()
	file, err := os.Create(cloudsFile)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
	}
	err = clouds.WriteCloudConfig(file)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
	}
}
