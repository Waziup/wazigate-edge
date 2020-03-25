package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
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

// Container represents the container running the App
type Container struct {
	Names   []string `json:"Names" bson:"Names"`
	ID      string   `json:"Id" bson:"Id"` // Container Id given by docker
	Created uint64   `json:"Created" bson:"Created"`
	State   string   `json:"State" bson:"State"`
	Status  string   `json:"Status" bson:"Status"`
}

/*-----------------------------*/

var installAppStatus = ""

// GetApps implements GET /apps
func GetApps(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// data, err := json.Marshal(clouds.GetClouds())
	// if err != nil {
	// 	log.Printf("[ERR  ] Error %v", err)
	// 	http.Error(resp, "internal server error", http.StatusInternalServerError)
	// 	return
	// }
	// resp.Header().Set("Content-Type", "application/json")
	// resp.Write(data)

	qryParams := req.URL.Query()

	if _, ok := qryParams["install_logs"]; ok {

		resp.Write([]byte(installAppStatus))
		return
	}

	/*------------*/

	if _, ok := qryParams["available"]; ok {

		resp.Write([]byte("List of all available Apps"))
		return
	}

	/*------------*/

	out := getListOfInstalledApps()

	outJSON, err := json.Marshal(out)
	if err != nil {
		log.Printf("[Err   ] %s", err.Error())
	}

	resp.Write([]byte(outJSON))
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
			log.Println(app.Name())
			out = append(out, map[string]interface{}{"id": repo.Name() + "." + app.Name()})
		}
	}

	return out
}

/*-----------------------------*/

// GetApp implements GET /apps/{app_id}
func GetApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	appPath := strings.Replace(appID, ".", "/", 1)

	/*----------*/

	cmd := "docker inspect " + appID
	dockerJSONRaw := tools.ExecOnHostWithLogs(cmd, true)

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

	outJSON, err := json.Marshal(out)
	if err != nil {
		log.Printf("[Err   ] %s", err.Error())
		resp.WriteHeader(500)
		resp.Write([]byte(err.Error()))
	}

	resp.Write([]byte(outJSON))
}

/*-----------------------------*/

// PostApps implements POST /apps
func PostApps(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// imageName := "waziup/wazi-on-sensors:1.0.0"
	input, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("[Err   ] installing app [%v] error: %s ", input, err.Error())
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	imageName := string(input)

	installAppStatus = "Installing initialized\n"

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

	installAppStatus += "\nDownloading [ " + appName + " : " + tag + " ] \n"

	cmd := "docker pull " + imageName
	out := tools.ExecOnHostWithLogs(cmd, true)

	// Status: Downloaded newer image for waziup/wazi-on-sensors:1.0.0
	installAppStatus += out

	if strings.Contains(out, "Error") {
		resp.WriteHeader(400)
		resp.Write([]byte("Download Failed!"))
		return
	}

	cmd = "docker create " + imageName
	containerID := tools.ExecOnHostWithLogs(cmd, true)

	installAppStatus += "\nTermporary container created\n"

	cmd = "docker cp " + containerID + ":/index.zip " + appFullPath
	out = tools.ExecOnHostWithLogs(cmd, true)

	installAppStatus += out

	// Error: No such container:path....

	if strings.Contains(out, "Error") {
		resp.WriteHeader(400)
		resp.Write([]byte("`index.zip` file extraction failed!"))
		return
	}

	cmd = "docker rm " + containerID
	out = tools.ExecOnHostWithLogs(cmd, true)

	cmd = "unzip -o " + appFullPath + "/index.zip -d " + appFullPath
	out = tools.ExecOnHostWithLogs(cmd, true)

	if strings.Contains(out, "cannot find") {
		installAppStatus += out
		resp.WriteHeader(400)
		resp.Write([]byte("Could not unzip `index.zip`!"))
		return
	}

	/*outJson, err := json.Marshal( out)
	if( err != nil) {
		log.Printf( "[Err   ] %s", err.Error())
	}/**/

	installAppStatus += "\nAll done :)"
	resp.Write([]byte("Install successfull"))

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
		if appConfig.Action == "uninstall" {
			cmd = "docker stop " + appID
			cmd += " ; docker rm --force " + appID
			cmd += " ; rm -r \"" + appFullPath + "\""

		} else {

			cmd = "cd \"" + appFullPath + "\"; docker-compose " + appConfig.Action
		}

		out := tools.ExecOnHostWithLogs(cmd, true)
		if out == "" {
			out = "[ " + appConfig.Action + " ] done"
		}

		tools.SendJSON(resp, out)

		// resp.Write([]byte(out))
	}

	/*------*/

	if appConfig.Restart != "" {

		cmd := "docker update --restart=" + appConfig.Restart + " " + appID
		out := tools.ExecOnHostWithLogs(cmd, true)

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

	cmd := "docker stop " + appID
	cmd += " ; docker rm --force " + appID

	if !keepConfig {
		cmd += " ; rm -r \"" + appFullPath + "\""
	}

	out := tools.ExecOnHostWithLogs(cmd, true)
	if out == "" {
		if keepConfig {
			out = "Uninstallation done, but the config is not deleted"
		} else {
			out = "The App is completely removed."
		}
	}

	tools.SendJSON(resp, out)
}

/*-----------------------------*/
