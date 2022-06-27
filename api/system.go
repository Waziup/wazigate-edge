package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/globalsign/mgo/bson"
	routing "github.com/julienschmidt/httprouter"
)

var Version string
var Branch string
var Buildtime int64

// SysClearAll implements PUT /sys/clear_all
func SysClearAll(resp http.ResponseWriter, req *http.Request, params routing.Params) {

}

var startTime time.Time

func init() {
	startTime = time.Now()
}

// SysGetUptime implements GET /sys/uptime
func SysGetUptime(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	uptime := time.Since(startTime)
	resp.Write([]byte(uptime.String()))
}

// SysGetLogs implements GET /sys/logs
func SysGetLogs(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	files, err := ioutil.ReadDir("./log")
	if err != nil {
		http.Error(resp, "Error: "+err.Error(), http.StatusInternalServerError)
	}
	resp.Write([]byte{'['})
	first := true

	var logFile struct {
		Name string    `json:"name"`
		Time time.Time `json:"time"`
		Size int64     `json:"size"`
	}

	encoder := json.NewEncoder(resp)

	for _, file := range files {
		name := file.Name()
		if logNameRegexp.MatchString(name) { // bson id (24) + ".txt" (4)
			if first == false {
				resp.Write([]byte{','})
			}
			first = false
			id := bson.ObjectIdHex(name[0:24])
			logFile.Name = name
			logFile.Size = file.Size()
			logFile.Time = id.Time()
			encoder.Encode(&logFile)
		}
	}
	resp.Write([]byte{']'})
}

var logNameRegexp = regexp.MustCompile(`^[0-9a-f]{24}\.txt$`)

// SysGetLog implements GET /sys/log/{log_name}
func SysGetLog(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	name := params.ByName("log_id")
	if !logNameRegexp.MatchString(name) {
		http.Error(resp, "Error: Bad log name.", http.StatusBadRequest)
		return
	}

	http.ServeFile(resp, req, "./log/"+name)
}

// SysDeleteLog implements DELETE /sys/log/{log_name}
func SysDeleteLog(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	name := params.ByName("log_id")
	if !logNameRegexp.MatchString(name) {
		http.Error(resp, "Error: Bad log name.", http.StatusBadRequest)
		return
	}

	err := os.Remove("./log/" + name)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(resp, "Error: Log file not found.", http.StatusNotFound)
			return
		}
		http.Error(resp, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func SysGetVersion(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	resp.Write([]byte(Version))
}

type Info struct {
	Version   string `json:"version"`
	Branch    string `json:"branch"`
	Buildtime int64  `json:"buildtime"`
}

func SysGetInfo(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	info := Info{
		Version:   Version,
		Branch:    Branch,
		Buildtime: Buildtime,
	}
	data, _ := json.Marshal(&info)
	resp.Write(data)
}
