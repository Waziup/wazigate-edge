package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"

	tools "github.com/Waziup/wazigate-edge/tools"
	routing "github.com/julienschmidt/httprouter"
)

/*-----------------------------*/

// We may use env vars in future, this path is relative to wazigate-host
const appsDirectoryOnHost = "../apps/"

// The apps folder is also mapped to make it easier and faster for some operation
const appsDirectoryMapped = "./apps"

/*-----------------------------*/

type installingAppStatusType struct {
	id   string
	done bool
	log  string
}

var installingAppStatus []installingAppStatusType

// GetApps implements GET /apps
func GetApps(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	qryParams := req.URL.Query()

	if _, ok := qryParams["install_logs"]; ok {

		tools.SendJSON(resp, installingAppStatus)
		return
	}

	/*------------*/

	var out []map[string]interface{}

	if _, ok := qryParams["available"]; ok {

		out = getListOfAvailableApps()

	} else {

		out = getListOfInstalledApps()

	}

	/*------------*/

	tools.SendJSON(resp, out)
}

/*-----------------------------*/

func getListOfAvailableApps() []map[string]interface{} {

	url := "http://localhost/available-apps.json"
	var out []map[string]interface{}

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return out
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return out
	}

	err = json.Unmarshal(body, &out)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
	}

	// for _, app := range appsList {
	// 	out = append(out, map[string]interface{}{
	// 		"id":    app["id"],
	// 		"image": app["image"],
	// 		// "status": ,
	// 	})
	// }

	return out
}

/*-----------------------------*/

func getListOfInstalledApps() []map[string]interface{} {

	var out []map[string]interface{}

	repoList, err := ioutil.ReadDir(appsDirectoryMapped)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return out
	}

	for _, repo := range repoList {
		appsList, err := ioutil.ReadDir(appsDirectoryMapped + "/" + repo.Name())
		if err != nil {
			log.Printf("[Err   ]: %s ", err.Error())
			continue
		}
		for _, app := range appsList {
			out = append(out, map[string]interface{}{"id": repo.Name() + "." + app.Name()})
		}
	}

	return out
}

/*-----------------------------*/

/*-----------------------------*/

// GetApp implements GET /apps/{app_id}
// GetApp implements GET /apps/{app_id}?install_logs
func GetApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	appPath := strings.Replace(appID, ".", "/", 1)

	/*----------*/

	qryParams := req.URL.Query()

	if _, ok := qryParams["install_logs"]; ok {
		for i := range installingAppStatus {
			if installingAppStatus[i].id == appID {

				out := map[string]interface{}{
					"log":  installingAppStatus[i].log,
					"done": installingAppStatus[i].done,
				}

				tools.SendJSON(resp, out)
				return
			}
		}

		tools.SendJSON(resp, installingAppStatusType{})
		return
	}

	/*----------*/

	cmd := "docker inspect " + appID
	dockerJSONRaw, _ := tools.ExecOnHostWithLogs(cmd, true)

	var dockerJSON []struct {
		State struct {
			Status     string `json:"Status"`
			Running    bool   `json:"Running"`
			Paused     bool   `json:"Paused"`
			Error      string `json:"Error"`
			StartedAt  string `json:"StartedAt"`
			FinishedAt string `json:"FinishedAt"`
			Health     struct {
				Status string `json:"Status"`
			} `json:"Health"`
		} `json:"State"`
		HostConfig struct {
			RestartPolicy struct {
				Name string `json:"Name"`
			} `json:"RestartPolicy"`
		} `json:"HostConfig"`
	}

	var dockerState map[string]interface{}

	if dockerJSONRaw != "" {
		if err := json.Unmarshal([]byte(dockerJSONRaw), &dockerJSON); err != nil {

			log.Printf("[Err   ] docker_inspect: %s", err.Error())

		} else {

			dockerState = map[string]interface{}{
				"Status":        dockerJSON[0].State.Status,
				"Running":       dockerJSON[0].State.Running,
				"Paused":        dockerJSON[0].State.Paused,
				"Error":         dockerJSON[0].State.Error,
				"StartedAt":     dockerJSON[0].State.StartedAt,
				"FinishedAt":    dockerJSON[0].State.FinishedAt,
				"Health":        dockerJSON[0].State.Health.Status,
				"RestartPolicy": dockerJSON[0].HostConfig.RestartPolicy.Name,
			}
		}
	}

	/*----------*/

	bytes, err := ioutil.ReadFile(appsDirectoryMapped + "/" + appPath + "/package.json")
	if err != nil {
		// resp.WriteHeader(404)
		resp.Write([]byte("{}"))
		log.Printf("[Err   ] package.json: %s", err.Error())
		return
	}

	var appPkg map[string]interface{}

	if err := json.Unmarshal(bytes, &appPkg); err != nil {
		// resp.WriteHeader(404)
		resp.Write([]byte("{}"))
		log.Printf("[Err   ] package.json: %s", err.Error())
		return
	}

	/*------*/

	out := map[string]interface{}{
		"id":          appID,
		"name":        appPkg["name"],
		"author":      appPkg["author"],
		"version":     appPkg["version"],
		"description": appPkg["description"],
		"homepage":    appPkg["homepage"],
		"state":       dockerState,
		"package":     appPkg["wazigate"],
	}

	tools.SendJSON(resp, out)

	// outJSON, err := json.Marshal(out)
	// if err != nil {
	// 	log.Printf("[Err   ] %s", err.Error())
	// 	resp.WriteHeader(500)
	// 	resp.Write([]byte(err.Error()))
	// }

	// resp.Write([]byte(outJSON))
}

/*-----------------------------*/

// PostApps implements POST /apps
func PostApps(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// imageName := "waziup/wazi-on-sensors:1.0.0"
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("[Err   ] installing app [%v] error: %s ", body, err.Error())
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	var imageName string
	err = json.Unmarshal(body, &imageName)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	//<!-- Get the App information

	sp1 := strings.Split(imageName, ":")

	tag := ""
	if len(sp1) == 2 {
		tag = sp1[1] //Image tag
	}

	sp2 := strings.Split(sp1[0], "/")

	repoName := sp2[0]
	appName := repoName + "_app" // some random default name in case of error
	if len(sp2) > 1 {
		appName = sp2[1]
	}

	appFullPath := appsDirectoryOnHost + repoName + "/" + appName

	//-->

	/*-----------*/
	appID := repoName + "." + appName
	appStatusIndex := -1
	for i := range installingAppStatus {
		if installingAppStatus[i].id == appID {
			appStatusIndex = i
		}
	}
	if appStatusIndex == -1 {
		installingAppStatus = append(installingAppStatus, installingAppStatusType{appID, false, ""})
		appStatusIndex = len(installingAppStatus) - 1
	}

	/*-----------*/

	installingAppStatus[appStatusIndex].log = "Installing initialized\n"

	installingAppStatus[appStatusIndex].log += "\nDownloading [ " + appName + " : " + tag + " ] \n"

	cmd := "docker pull " + imageName
	out, err := tools.ExecOnHostWithLogs(cmd, true)

	installingAppStatus[appStatusIndex].log += out

	if err != nil {
		resp.WriteHeader(400)
		installingAppStatus[appStatusIndex].done = true
		tools.SendJSON(resp, "Download Failed!")
		return
	}

	/*-----------*/

	cmd = "docker create " + imageName
	containerID, err := tools.ExecOnHostWithLogs(cmd, true)

	if err != nil {
		resp.WriteHeader(400)
		installingAppStatus[appStatusIndex].done = true
		tools.SendJSON(resp, err.Error())
		return
	}

	installingAppStatus[appStatusIndex].log += "\nTermporary container created\n"

	/*-----------*/

	cmd = "mkdir -p \"" + appsDirectoryOnHost + repoName + "\" ;"
	cmd = "mkdir -p \"" + appFullPath + "\""
	out, err = tools.ExecOnHostWithLogs(cmd, true)
	if err != nil {
		resp.WriteHeader(400)
		installingAppStatus[appStatusIndex].done = true
		tools.SendJSON(resp, err.Error())
		return
	}

	/*-----------*/

	cmd = "docker cp " + containerID + ":/index.zip " + appFullPath + "/"
	out, err = tools.ExecOnHostWithLogs(cmd, true)

	installingAppStatus[appStatusIndex].log += out

	if err != nil {
		resp.WriteHeader(400)
		installingAppStatus[appStatusIndex].done = true
		tools.SendJSON(resp, "`index.zip` file extraction failed!")
		return
	}

	/*-----------*/

	cmd = "docker rm " + containerID
	out, _ = tools.ExecOnHostWithLogs(cmd, true)

	/*-----------*/

	cmd = "unzip -o " + appFullPath + "/index.zip -d " + appFullPath
	out, err = tools.ExecOnHostWithLogs(cmd, true)

	if err != nil {
		installingAppStatus[appStatusIndex].log += out
		installingAppStatus[appStatusIndex].done = true
		resp.WriteHeader(400)
		tools.SendJSON(resp, "Could not unzip `index.zip`!")
		return
	}

	/*-----------*/

	/*outJson, err := json.Marshal( out)
	if( err != nil) {
		log.Printf( "[Err   ] %s", err.Error())
	}/**/

	installingAppStatus[appStatusIndex].log += "\nAll done :)"
	installingAppStatus[appStatusIndex].done = true
	tools.SendJSON(resp, "Install successfull")
}

/*-----------------------------*/

// PostApp implements POST /apps/{app_id}   action={start | stop}
// PostApp implements POST /apps/{app_id}   restart={"always" | "on-failure" | "unless-stopped" | "no"}
func PostApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	appFullPath := appsDirectoryOnHost + strings.Replace(appID, ".", "/", 1)

	/*------*/

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	type _appConfig struct {
		Action  string `json:"action" bson:"action"`   //"start" | "stop" | "uninstall"
		Restart string `json:"restart" bson:"restart"` // "always" | "on-failure" | "unless-stopped" | "no"
	}

	var appConfig _appConfig
	err = json.Unmarshal(body, &appConfig)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if appConfig.Action != "" {

		cmd := ""
		// if appConfig.Action == "uninstall" {
		// 	cmd = "docker stop " + appID
		// 	cmd += " ; docker rm --force " + appID
		// 	cmd += " ; rm -r \"" + appFullPath + "\""

		// } else {

		cmd = "cd \"" + appFullPath + "\"; docker-compose " + appConfig.Action
		// }

		out, err := tools.ExecOnHostWithLogs(cmd, true)
		if err != nil {
			log.Printf("[Err   ] %s ", err.Error())
			out = err.Error()
		}
		if out == "" {
			out = "[ " + appConfig.Action + " ] done"
		}

		tools.SendJSON(resp, out)

		// resp.Write([]byte(out))
	}

	/*------*/

	if appConfig.Restart != "" {

		cmd := "docker update --restart=" + appConfig.Restart + " " + appID
		out, err := tools.ExecOnHostWithLogs(cmd, true)

		if err != nil {
			log.Printf("[Err   ] %s ", err.Error())
			out = err.Error()
		}

		if out == "" {
			out = "Restart policy set to [ " + appConfig.Restart + " ]"
		}

		tools.SendJSON(resp, out)

		// resp.Write([]byte(out))
	}
}

/*-----------------------------*/

// DeleteApp implements DELETE /apps/{app_id}?keepConfig={true | false}
func DeleteApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	appFullPath := appsDirectoryOnHost + strings.Replace(appID, ".", "/", 1)

	/*------*/

	qryParams := req.URL.Query()
	keepConfig := true

	if value, ok := qryParams["keepConfig"]; ok {
		keepConfig = value[0] == "true"
	}

	/*------*/

	cmd := "docker stop \"" + appID + "\"; "
	cmd += "docker rm --force \"" + appID + "\"; "

	if !keepConfig {
		cmd += "rm -r \"" + appFullPath + "\""

	} else {

		cmd += "rm \"" + appFullPath + "/package.json\""
	}

	out, err := tools.ExecOnHostWithLogs(cmd, true)
	if err != nil {
		log.Printf("[Err   ] %s ", err.Error())
		out = err.Error()
	}

	log.Printf("[Info   ] DELETE App: %s\n\t%v\n", appID, out)

	if len(out) == 0 {
		if keepConfig {
			out = "Uninstallation done, but the config is not deleted"
		} else {
			out = "The App is completely removed."
		}
	}

	tools.SendJSON(resp, out)
}

/*-----------------------------*/

// HandleAppProxyRequest implements GET and POST /apps/{app_id}/*file_path
func HandleAppProxyRequest(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	//TODO: We need a security mechanism here in order to prevent calls to internal parts

	appID := params.ByName("app_id")

	socketAddr := appsDirectoryMapped + "/" + strings.Replace(appID, ".", "/", 1) + "/conf/socket.sock"

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketAddr)
			},
		},
	}

	filePath := params.ByName("file_path")

	var socketResponse *http.Response
	var err error

	switch req.Method {
	case http.MethodGet:
		socketResponse, err = httpc.Get("http://localhost/" + filePath)
		break

	case http.MethodPost:
		socketResponse, err = httpc.Post(
			"http://localhost/"+filePath,
			req.Header.Get("Content-Type"),
			req.Body)
		break

	}

	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		resp.WriteHeader(500)
		resp.Write([]byte(err.Error()))
		return
	}

	if socketResponse.StatusCode != 200 {
		log.Printf("[Err]: Status Code: %v ", socketResponse.StatusCode)
		resp.WriteHeader(socketResponse.StatusCode)
		resp.Write([]byte(socketResponse.Status))
		return
	}

	body, err := ioutil.ReadAll(socketResponse.Body)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		resp.WriteHeader(500)
		resp.Write([]byte(err.Error()))
		return
	}

	// We have issue that unix socket does not set corret mime type for css
	if strings.HasSuffix(filePath, ".css") {
		resp.Header().Set("Content-Type", "text/css; charset=utf-8")
	}

	resp.Write(body)
}

/*-----------------------------*/
