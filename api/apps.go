package api

import (
	"context"
	"encoding/json"
	"regexp"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	tools "github.com/Waziup/wazigate-edge/tools"
	routing "github.com/julienschmidt/httprouter"
)

/*-----------------------------*/

// We may use env vars in future, this path is relative to wazigate-host
const appsDirectoryOnHost = "../apps/"

// The apps folder is also mapped to make it easier and faster for some operation
const appsDirectoryMapped = "/root/apps"

const dockerSocketAddress = "/var/run/docker.sock"

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

	// I keep it hard-coded because later we can update this via update the edge through the update mechanism ;)
	url := "https://raw.githack.com/Waziup/WaziApps/master/available-apps.json"
	var out []map[string]interface{}

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[ERR  ]: %s ", err.Error())
		return out
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Printf("[ERR  ]: %s ", err.Error())
		return out
	}

	err = json.Unmarshal(body, &out)
	if err != nil {
		log.Printf("[ERR  ]: %s ", err.Error())
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
		log.Printf("[ERR  ]: %s ", err.Error())
		return out
	}

	for _, repo := range repoList {
		appsList, err := ioutil.ReadDir(appsDirectoryMapped + "/" + repo.Name())
		if err != nil {
			log.Printf("[ERR  ]: %s ", err.Error())
			continue
		}
		for _, app := range appsList {

			appID := repo.Name() + "." + app.Name()
			appInfo := getAppInfo(appID)

			if appInfo != nil {
				out = append(out, appInfo)
			}
			// else {
			// 	out = append(out, map[string]interface{}{"id": repo.Name() + "." + app.Name()})
			// }
		}
	}

	return out
}

/*-----------------------------*/

// GetApp implements GET /apps/{app_id}
// GetApp implements GET /apps/{app_id}?install_logs
func GetApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")

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

	out := getAppInfo(appID)

	if out == nil {
		resp.Write([]byte("{}"))
		return
	}

	/*----------*/

	tools.SendJSON(resp, out)

}

/*-----------------------------*/

func getAppInfo(appID string) map[string]interface{} {

	appPath := strings.Replace(appID, ".", "/", 1)

	// cmd := "docker inspect " + appID
	// dockerJSONRaw, _ := tools.ExecOnHostWithLogs(cmd, true)

	dockerJSONRaw, _ := tools.SockGetReqest(dockerSocketAddress, "containers/"+appID+"/json")

	var dockerJSON struct {
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

	if dockerJSONRaw != nil {
		if err := json.Unmarshal(dockerJSONRaw, &dockerJSON); err != nil {

			log.Printf("[ERR  ] docker_inspect: %s", err.Error())

		} else {

			dockerState = map[string]interface{}{
				"status":        dockerJSON.State.Status,
				"running":       dockerJSON.State.Running,
				"paused":        dockerJSON.State.Paused,
				"error":         dockerJSON.State.Error,
				"startedAt":     dockerJSON.State.StartedAt,
				"finishedAt":    dockerJSON.State.FinishedAt,
				"health":        dockerJSON.State.Health.Status,
				"restartPolicy": dockerJSON.HostConfig.RestartPolicy.Name,
			}
		}
	}

	/*----------*/

	bytes, err := ioutil.ReadFile(appsDirectoryMapped + "/" + appPath + "/package.json")
	if err != nil {
		// resp.WriteHeader(404)

		log.Printf("[ERR  ] package.json: %s", err.Error())
		return nil
	}

	var appPkg map[string]interface{}

	if err := json.Unmarshal(bytes, &appPkg); err != nil {
		// resp.WriteHeader(404)

		log.Printf("[ERR  ] package.json: %s", err.Error())
		return nil
	}

	/*------*/

	return map[string]interface{}{
		"id":          appID,
		"name":        appPkg["name"],
		"author":      appPkg["author"],
		"version":     appPkg["version"],
		"description": appPkg["description"],
		"homepage":    appPkg["homepage"],
		"state":       dockerState,
		"package":     appPkg["wazigate"],
	}

}

/*-----------------------------*/

// PostApps implements POST /apps
// It installs a new app
func PostApps(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// imageName := "waziup/wazi-on-sensors:beta"
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("[ERR  ] installing app [%v] error: %s ", body, err.Error())
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

	// out, err := tools.SockPostReqest( dockerSocketAddress, "containers/create", imageName)
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

	// out, err = tools.SockPostReqest( dockerSocketAddress, "images/create", "{\"Image\": \""+ imageName +"\"}")

	cmd = "docker create " + imageName
	containerID, err := tools.ExecOnHostWithLogs(cmd, true)

	if err != nil {
		resp.WriteHeader(400)
		installingAppStatus[appStatusIndex].done = true
		tools.SendJSON(resp, err.Error())
		return
	}

	// dockerJSONRaw, _ := tools.SockGetReqest( dockerSocketAddress, "containers/"+ appID +"/json" )

	// var dockerJSON struct {
	// 	Image	string `json:"Image"`
	// }

	// if dockerJSONRaw != nil {
	// 	if err := json.Unmarshal(dockerJSONRaw, &dockerJSON); err == nil {
	// 		appImageID = dockerJSON.Image;
	// 	}
	// }

	// containerID :=

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

	cmd = "rm -f " + appFullPath + "/index.zip"
	out, _ = tools.ExecOnHostWithLogs(cmd, true)

	/*-----------*/

	/*outJson, err := json.Marshal( out)
	if( err != nil) {
		log.Printf( "[ERR  ] %s", err.Error())
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
		Action  string `json:"action" bson:"action"`   //"start" | "stop" | "first-start"
		Restart string `json:"restart" bson:"restart"` // "always" | "on-failure" | "unless-stopped" | "no"
	}

	var appConfig _appConfig
	err = json.Unmarshal(body, &appConfig)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if appConfig.Action != "" {

		// /containers/{id}/start // docker-compose is much simpler to use than docker APIs

		cmd := "cd \"" + appFullPath + "\"; docker-compose " + appConfig.Action

		if appConfig.Action == "first-start" {
			cmd = "cd \"" + appFullPath + "\"; docker-compose pull ; docker-compose up -d --no-build"
		}

		out, err := tools.ExecOnHostWithLogs(cmd, true)
		if err != nil {
			log.Printf("[ERR  ] %s ", err.Error())
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

		// cmd := "docker update --restart=" + appConfig.Restart + " " + appID
		// out, err := tools.ExecOnHostWithLogs(cmd, true)

		updateStr := fmt.Sprintf(`{"RestartPolicy": { "Name": "%s"}}`, appConfig.Restart)
		out, err := tools.SockPostReqest(dockerSocketAddress, "containers/"+appID+"/update", updateStr)

		if err != nil {
			log.Printf("[ERR  ] %s ", err.Error())
			out = []byte(err.Error())
		}

		if out == nil {
			out = []byte("Restart policy set to [ " + appConfig.Restart + " ]")
		}

		tools.SendJSON(resp, out)

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

	appImageID := ""

	dockerJSONRaw, _ := tools.SockGetReqest(dockerSocketAddress, "containers/"+appID+"/json")

	var dockerJSON struct {
		Image string `json:"Image"`
	}

	if dockerJSONRaw != nil {
		if err := json.Unmarshal(dockerJSONRaw, &dockerJSON); err == nil {
			appImageID = dockerJSON.Image
		}
	}

	/*------*/

	tools.SockDeleteReqest(dockerSocketAddress, "containers/"+appID+"?force=true")

	if appImageID != "" {
		tools.SockDeleteReqest(dockerSocketAddress, "images/"+appImageID+"?force=true")
	}

	// Note: for the apps that have multiple containers and images, we need to find another way.
	// Like this: docker-compose rm -fs

	cmd := ""
	if !keepConfig {
		cmd = "rm -r \"" + appFullPath + "\""

	} else {

		cmd = "rm \"" + appFullPath + "/package.json\""
	}

	out, err := tools.ExecOnHostWithLogs(cmd, true)
	if err != nil {
		log.Printf("[ERR  ] %s ", err.Error())
		out = err.Error()
	}

	log.Printf("[APP  ] DELETE App: %s\n\t%v\n", appID, out)

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

// HandleAppProxyRequest implements GET, POST, PUT and DELETE /apps/{app_id}/*file_path
func HandleAppProxyRequest(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	//TODO: We need a security mechanism here in order to prevent calls to internal parts

	appID := params.ByName("app_id")

	socketAddr := appsDirectoryMapped + "/" + strings.Replace(appID, ".", "/", 1) + "/proxy.sock"

	proxy := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				// the proxy uses linux sockets that are created by each app
				return net.Dial("unix", socketAddr)
			},
			MaxIdleConns:    50,
			IdleConnTimeout: 4 * 60 * time.Second,
		},
	}

	// remove /apps/{id} from the URI
	proxyURI := req.URL.RequestURI()[len(appID)+6:]

	proxyURL := "http://localhost" + proxyURI

	proxyReq, err := http.NewRequest(req.Method, proxyURL, req.Body)
	if err != nil {
		log.Printf("[APP  ] Err %v", err)
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(err.Error()))
		return
	}

	log.Printf("[APP  ] >> %q %s %s", appID, req.Method, proxyURI)

	proxyReq.Header = req.Header

	proxyResp, err := proxy.Do(proxyReq)
	if err != nil {
		log.Printf("[APP  ] Err %v", err)
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(err.Error()))
		return
	}

	for key, value := range proxyResp.Header {
		resp.Header()[key] = value
	}
	resp.WriteHeader(proxyResp.StatusCode)

	var written int64
	if proxyResp.Body != nil {
		written, _ = io.Copy(resp, proxyResp.Body)
	}
	log.Printf("[APP  ] << %d %s (%d B)", proxyResp.StatusCode, proxyResp.Status, written)
}

/*-----------------------------*/

func handleAppProxyError(appID string) string {

	appInfo := getAppInfo(appID)

	appName := appID
	if appInfo["name"] != nil {
		appName = appInfo["name"].(string)
	}

	errMsg := ""
	if appInfo["package"] == nil {

		errMsg = "This app is not installed!"

	} else if appInfo["state"] == nil {

		errMsg = "This app has not launched yet!"

	} else {

		errMsg = "This app is not running!"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
	<html>
		<head>
			<link rel="stylesheet" href="/dist/main.css">
		</head>
		<body>
			<div class="error">
				<h2>%s</h2>
				<h4>Error on loading the app [ %s ]<h4>
			</div>
		</body>
	</html>`, errMsg, appName)

}

/*-----------------------------*/

// GetUpdateApp implements GET /update/:app_id
func GetUpdateApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	newUpdate := false;

	images, err := getAppImages( appID)
	if err != nil {
		log.Printf("[APP  ] Err %v", err)
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(err.Error()))
		return
	}

	for _, image := range images{

		/*-------*/

		remoteImageInfoRaw, _ := tools.SockGetReqest(dockerSocketAddress, "distribution/"+ image +"/json")
		if err != nil{
			log.Printf("[APP  ] %v", err)
			continue;
		}

		var remoteImageInfo struct {
			Descriptor struct {
				Digest		string	`json:"Digest"`
				Size		int64	`json:"Size"`
			} `json:"Descriptor"`
		}
	
		if remoteImageInfoRaw == nil {
			continue;
		}
		if err := json.Unmarshal(remoteImageInfoRaw, &remoteImageInfo); err != nil {
			log.Printf("[APP  ] Err %v", err)
			continue;
		}

		/*-------*/

		localImageInfoRaw, _ := tools.SockGetReqest(dockerSocketAddress, "images/"+ image +"/json")
		if err != nil{
			log.Printf("[APP  ] %v", err)
			continue;
		}

		var localImageInfo struct {
			Digests		[]string	`json:"RepoDigests"`
		}
	
		if localImageInfoRaw == nil {
			continue;
		}
		if err := json.Unmarshal(localImageInfoRaw, &localImageInfo); err != nil {
			log.Printf("[APP  ] Err %v", err)
			continue;
		}

		/*-------*/

		localImageDigest := "";
		if len( localImageInfo.Digests) > 0{
			re := regexp.MustCompile(`[^@]+@`)
			localImageDigest = re.ReplaceAllString(localImageInfo.Digests[0], "")
		}

		// Even if the local digest does not exist (due to building it instead of pulling), we update the app
		if( localImageDigest != remoteImageInfo.Descriptor.Digest){
			// New update is available
			newUpdate = true;
			break;
		}
	}

	/*------------*/

	out := map[string]interface{}{
		"newUpdate":  newUpdate,
	}

	tools.SendJSON(resp, out)
}

/*-----------------------------*/

func getAppImages( appID string) ([]string, error){

	appFullPath := appsDirectoryMapped + "/" + strings.Replace(appID, ".", "/", 1)
	var out []string;

	yamlFile, err := ioutil.ReadFile( appFullPath + "/docker-compose.yml")
    if err != nil {
		log.Printf("[APP  ] docker-compose.yml : %v ", err)
		return out, err
	}

	// err = yaml.Unmarshal( yamlFile, &dockerCompose) // it did not work without giving the service name

	re := regexp.MustCompile(`image[\s]*:[\s]*([a-zA-Z0-9/\:\-]+)`)

	submatchall := re.FindAllStringSubmatch( string( yamlFile), -1)
	for _, element := range submatchall {
		out = append( out, element[1])
	}		

	return out, nil
}

/*-----------------------------*/

// out = getListOfInstalledApps()