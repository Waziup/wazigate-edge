package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/tools"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	routing "github.com/julienschmidt/httprouter"
)

/*-----------------------------*/

// const edgeVersion = "2.1.3"

// We may use env vars in future, path to waziapps folder
// const appsDir = "/var/lib/wazigate/apps/"

// The apps folder is also mapped to make it easier and faster for some operation
const appsDir = "apps"

const dockerSocketAddress = "/var/run/docker.sock"

func init() {
	if err := os.Mkdir(appsDir, 0755); err != nil {
		if !os.IsExist(err) {
			log.Fatalf("The Wazigate Apps directory could not be created: %v", err)
		}
	}
}

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

		getUpdateEdgeStatus() // we call this because for Edge update the procedure is different
		tools.SendJSON(resp, installingAppStatus)

		return
	}

	/*------------*/

	var out []map[string]interface{}
	var err error

	if _, ok := qryParams["available"]; ok {

		out, err = getListOfAvailableApps()

	} else {

		out, err = getListOfInstalledApps(true /*withDockerStatus*/)
	}

	/*------------*/

	if err != nil {
		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(500)
		log.Printf("[ERR  ]: %s ", err.Error())
	}

	if out == nil {
		tools.SendJSON(resp, []map[string]interface{}{})
		return
	}

	tools.SendJSON(resp, out)
}

/*-----------------------------*/

// Shows the apps from Market Place
func getListOfAvailableApps() ([]map[string]interface{}, error) {

	var out, appsList []map[string]interface{}

	// I keep it hard-coded because later we can update this via update the edge through the update mechanism ;)
	url := "https://raw.githack.com/Waziup/WaziApps/master/available-apps.json"
	resp, err := http.Get(url)
	if err != nil { // IF it fails we will use the backup URL
		url = "https://raw.githubusercontent.com/Waziup/WaziApps/master/available-apps.json"
		resp, err = http.Get(url)
	}

	if err != nil {
		return out, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return out, err
	}

	err = json.Unmarshal(body, &appsList)

	/*---------*/

	installedAppsIface, err := getListOfInstalledApps(false /*No need to get the status of containers*/)
	if err != nil {
		// Do nothing for the moment
	}

	installedAppsList := make(map[string]interface{})
	for _, app := range installedAppsIface {
		installedAppsList[app["id"].(string)] = 1
	}

	/*---------*/

	// Filter out the installed apps
	for _, app := range appsList {
		if _, ok := installedAppsList[app["id"].(string)]; !ok {
			out = append(out, app)
		}
	}

	return out, err
}

/*-----------------------------*/

func getListOfInstalledApps(withDockerStatus bool) ([]map[string]interface{}, error) {

	var out []map[string]interface{}

	// We need to add edge as an app to have a unify update interface in the ui
	// However we treat it differently
	out = append(out, getAppInfo("wazigate-edge", withDockerStatus))

	appsList, err := ioutil.ReadDir(appsDir)
	if err != nil {
		return out, err
	}
	for _, app := range appsList {
		appId := app.Name()
		log.Printf("Checking app '%s' ...", appId)
		appInfo := getAppInfo(appId, withDockerStatus)
		if appInfo != nil {
			out = append(out, appInfo)
		}
	}

	return out, nil
}

/*-----------------------------*/

// GetApp implements GET /apps/{app_id}
// GetApp implements GET /apps/{app_id}?install_logs
func GetApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")

	/*----------*/

	if appID == "wazigate-edge" {
		getUpdateEdgeStatus()
	}

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

	out := getAppInfo(appID, true /* withDockerStatus */)

	if out == nil {
		resp.Write([]byte("{}"))
		return
	}

	/*----------*/

	tools.SendJSON(resp, out)

}

/*-----------------------------*/

func getAppInfo(appID string, withDockerStatus bool) map[string]interface{} {

	// appPath := strings.Replace(appID, ".", "/", 1)

	var dockerState map[string]interface{}
	if withDockerStatus {

		// cmd := "docker inspect " + appID
		// dockerJSONRaw, _ := tools.ExecCommand(cmd, true)

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
			Config struct {
				Image string `json:"Image"`
			} `json:"Config"`
		}

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
					"image":         dockerJSON.Config.Image,
				}
			}
		}
	}

	/*----------*/

	var appPkg map[string]interface{}

	if appID == "wazigate-edge" {

		edgeVersion := os.Getenv("EDGE_VERSION") // It is defined and changed on build, in the Dockerfile
		if edgeVersion == "" {
			edgeVersion = "N/A"
		}

		appPkg = map[string]interface{}{
			"name":        "Wazigate Edge Framework",
			"author":      map[string]interface{}{"name": "Waziup"},
			"version":     edgeVersion,
			"description": "Waziup firmware for Edge computing",
			"homepage":    "https://www.waziup.io/",
			"waziapp":     map[string]interface{}{"icon": "img/waziup.svg", "menu": nil},
		}

	} else {

		appPkgRaw, err := ioutil.ReadFile(filepath.Join(appsDir, appID, "package.json"))
		if err != nil {
			// resp.WriteHeader(404)

			log.Printf("[ERR  ] package.json: %s", err.Error())
			return nil
		}

		if err := json.Unmarshal(appPkgRaw, &appPkg); err != nil {
			// resp.WriteHeader(404)
			log.Printf("[ERR  ] package.json: %s", err.Error())
			return nil
		}
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
		"waziapp":     appPkg["waziapp"],
	}

}

/*-----------------------------*/

// PostApps implements POST /apps
// It installs a new app
func PostApps(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	resp.Header().Set("Content-Type", "application/json")

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

	out, err := installApp(imageName)
	if err != nil {
		log.Printf("[ERR  ] installing app [%v] error: %s ", imageName, err.Error())
		resp.WriteHeader(400)
		// tools.SendJSON(resp, out)
		// http.Error(resp, err.Error(), http.StatusBadRequest)
		// return
	}

	tools.SendJSON(resp, out)
}

/*-----------------------------*/

// PostApp implements POST /apps/{app_id}   action={start | stop}
// PostApp implements POST /apps/{app_id}   restart={"always" | "on-failure" | "unless-stopped" | "no"}
func PostApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	appFullPath := filepath.Join(appsDir, appID)

	resp.Header().Set("Content-Type", "application/json")

	/*------*/

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		log.Printf("[Err  ] PostApp: %s", err.Error())
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
		log.Printf("[Err  ] PostApp: %s", err.Error())
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if appConfig.Action != "" {

		// /containers/{id}/start // docker-compose is much simpler to use than docker APIs

		cmd := "docker-compose " + appConfig.Action
		if appConfig.Action == "first-start" {
			cmd = "docker-compose pull && docker-compose up -d --no-build"
		}
		out, err := tools.Shell(appFullPath, cmd)
		if err != nil {
			log.Printf("[Err  ] PostApp: %s", err.Error())
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
		// out, err := tools.ExecCommand(cmd, true)

		updateStr := fmt.Sprintf(`{"RestartPolicy": { "Name": "%s"}}`, appConfig.Restart)
		out, err := tools.SockPostReqest(dockerSocketAddress, "containers/"+appID+"/update", updateStr)

		if err != nil {
			log.Printf("[Err  ] PostApp: %s", err.Error())
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

	/*------*/

	qryParams := req.URL.Query()
	keepConfig := true

	if value, ok := qryParams["keepConfig"]; ok {
		keepConfig = value[0] == "true"
	}

	/*------*/

	err := uninstallApp(appID, keepConfig)

	out := ""
	if err != nil {

		log.Printf("[ERR  ] %s ", err.Error())
		out = err.Error()

	} else {

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

	socketAddr := appsDir + "/" + appID + "/proxy.sock"

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
		resp.Write([]byte(handleAppProxyError(appID, err.Error()))) //Showing a nice user friendly error msg
		return
	}

	log.Printf("[APP  ] >> %q %s %s", appID, req.Method, proxyURI)

	// We need to pass these values in order to let the Apps work properly (I had issues with a Python based service)
	proxyReq.Header = req.Header
	proxyReq.TransferEncoding = []string{"identity"}
	proxyReq.ContentLength = req.ContentLength

	proxyResp, err := proxy.Do(proxyReq)
	if err != nil {
		log.Printf("[APP  ] Err %v", err)
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(handleAppProxyError(appID, err.Error())))
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

func handleAppProxyError(appID string, moreInfo string) string {

	appInfo := getAppInfo(appID, true /* withDockerStatus */)

	appName := appID
	if appInfo["name"] != nil {
		appName = appInfo["name"].(string)
	}

	errMsg := ""
	if appInfo["waziapp"] == nil {

		errMsg = "This app is not installed!"

	} else if appInfo["state"] == nil {

		errMsg = "This app has not launched yet!"

	} else {

		errMsg = "This app is not running!"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
	<html>
		<head>
			<style type="text/css">
				.error{padding: 24px;margin-top: 50px;z-index: 1;position: relative;background-color: #ffb294;
					border-radius: 5px;font-family: "Roboto", "Helvetica", "Arial", sans-serif;}
				.error p{border: 1px solid #ca4e1d;padding: 10px;border-radius: 3px;}
				.error h2{font-size: 3.75rem;font-weight: 300;line-height: 1.2;}
				.error svg{top: 82px;color: #c7917c;right: 18px;width: 90px;
					height: 90px;z-index: -1;position: absolute;fill: currentColor;
					display: inline-block;font-size: 1.5rem;flex-shrink: 0;user-select: none;
					transition: fill 200ms cubic-bezier(0.4, 0, 0.2, 1) 0ms;}
			</style>
		</head>
		<body>
			<div class="error">
				<h2>%s</h2>
				<h4>Error on loading the app [ %s ]</h4>
				<svg focusable="false" viewBox="0 0 24 24" aria-hidden="true">
					<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"></path>
				</svg>
				<p>%s</p>
			</div>
		</body>
	</html>`, errMsg, appName, moreInfo)

}

/*-----------------------------*/

// GetUpdateApp implements GET /update/:app_id
func GetUpdateApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")
	newUpdate := false

	resp.Header().Set("Content-Type", "application/json")

	images, err := getAppImages(appID)
	if err != nil {
		log.Printf("[APP  ] Err %v", err)
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(err.Error()))
		return
	}

	for _, image := range images {

		/*-------*/

		remoteImageInfoRaw, _ := tools.SockGetReqest(dockerSocketAddress, "distribution/"+image+"/json")
		if err != nil {
			log.Printf("[APP  ] %v", err)
			continue
		}

		var remoteImageInfo struct {
			Descriptor struct {
				Digest string `json:"Digest"`
				Size   int64  `json:"Size"`
			} `json:"Descriptor"`
		}

		if remoteImageInfoRaw == nil {
			continue
		}
		if err := json.Unmarshal(remoteImageInfoRaw, &remoteImageInfo); err != nil {
			log.Printf("[APP  ] Err %v", err)
			continue
		}

		/*-------*/

		localImageInfoRaw, _ := tools.SockGetReqest(dockerSocketAddress, "images/"+image+"/json")
		if err != nil {
			log.Printf("[APP  ] %v", err)
			continue
		}

		var localImageInfo struct {
			Digests []string `json:"RepoDigests"`
		}

		if localImageInfoRaw == nil {
			continue
		}
		if err := json.Unmarshal(localImageInfoRaw, &localImageInfo); err != nil {
			log.Printf("[APP  ] Err %v", err)
			continue
		}

		/*-------*/

		localImageDigest := ""
		if len(localImageInfo.Digests) > 0 {
			re := regexp.MustCompile(`[^@]+@`)
			localImageDigest = re.ReplaceAllString(localImageInfo.Digests[0], "")
		}

		// Even if the local digest does not exist (due to building it instead of pulling), we update the app
		if localImageDigest != remoteImageInfo.Descriptor.Digest {
			// New update is available
			newUpdate = true
			break
		}

	}

	/*------------*/

	out := map[string]interface{}{
		"newUpdate": newUpdate,
	}

	tools.SendJSON(resp, out)
}

/*-----------------------------*/

func getAppImages(appID string) ([]string, error) {

	appFullPath := appsDir + "/" + appID
	var out []string

	if appID == "wazigate-edge" {
		cmd := "cd ../ && CNTS=$(sudo docker-compose ps -q) && for cId in $CNTS; do cImage=$(sudo docker ps --format '{{.Image}}' -f id=${cId}); echo $cImage; done;"
		stdout, err := tools.ExecCommand(cmd, true)
		out = strings.Split(strings.TrimSpace(stdout), "\n")
		return out, err
	}

	yamlFile, err := ioutil.ReadFile(appFullPath + "/docker-compose.yml")
	if err != nil {
		log.Printf("[APP  ] docker-compose.yml : %v ", err)
		return out, err
	}

	// err = yaml.Unmarshal( yamlFile, &dockerCompose) // it did not work without giving the service name

	re := regexp.MustCompile(`image[\s]*:[\s]*([a-zA-Z0-9/\:\_\.\-]+)`)

	subMatchAll := re.FindAllStringSubmatch(string(yamlFile), -1)
	for _, element := range subMatchAll {
		out = append(out, element[1])
	}

	return out, nil
}

/*-----------------------------*/

func dockerHubAccessible() bool {

	cmd := "timeout 3 curl -Is https://hub.docker.com/ | head -n 1 | awk '{print $2}'"
	rCode, err := tools.ExecCommand(cmd, true)

	if err != nil {
		log.Printf("[ERR  ] Docker Hub Accesibility Error: %s\n", err.Error())
		return false
	}

	return rCode == "200"
}

/*-----------------------------*/

// PostUpdateApp implements POST /update/:app_id
// it updates the given app by pulling the latest images from docker hub and replace with the current one (uninstall the current version and install a new one ;])
// please note that it will replace docker-compose.yml and package.json files with the new version as well.
func PostUpdateApp(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	appID := params.ByName("app_id")

	//<!-- Checking the connectivity

	if !dockerHubAccessible() {
		err := "Update failed. Please check your connectivity!"
		tools.SendJSON(resp, err)
		log.Printf("[ERR  ] updating app [%v] error: %s ", appID, err)
		return
	}

	//-->

	//<!-- Updating the Edge, it is an exeption because I have to stop myself,
	//	then download the latest version of myself, remove my older version and then start myself ;)

	if appID == "wazigate-edge" {

		err := updateEdge()
		if err != nil {
			tools.SendJSON(resp, err.Error())
			log.Printf("[ERR  ] updating the Edge error: %s ", err.Error())
			return
		}
		tools.SendJSON(resp, "Update Done.")
		return
	}

	//-->

	//<!-- Finding the main image name of the app

	appInfo := getAppInfo(appID, true /* withDockerStatus */)
	if appInfo == nil {
		resp.WriteHeader(400)
		err := "App image name cannot be found!"
		tools.SendJSON(resp, err)
		log.Printf("[ERR  ] updating app [%v] error: %s ", appID, err)
		return
	}

	imageName := ""
	if state, ok := appInfo["state"]; ok {
		if image, ok := state.(map[string]interface{})["image"].(string); ok {
			imageName = image
		}
	}

	if imageName == "" {

		resp.WriteHeader(400)
		err := "App Image Name cannot be found in the inspection!"
		tools.SendJSON(resp, err)
		log.Printf("[ERR  ] updating app [%v] error: %s ", appID, err)
		return
	}

	//-->

	// Update begins here:

	err := uninstallApp(appID, true /* Keep config and data */)
	if err != nil {
		msg := "Removing the old version failed!"
		tools.SendJSON(resp, msg)
		log.Printf("[ERR  ] updating app [%v] error: %s ", appID, msg)
	}

	out, err := installApp(imageName)
	if err != nil {
		log.Printf("[ERR  ] installing App update [%v] error: %s ", imageName, err.Error())
		// http.Error(resp, err.Error(), http.StatusBadRequest)
		resp.WriteHeader(400)
		tools.SendJSON(resp, out)
		return
	}

	tools.SendJSON(resp, "The App is updated successfully")
}

/*-----------------------------*/
var updateEdgeInProgress = false

func updateEdge() error {

	updateEdgeInProgress = true

	cmd := "sudo bash ../../update.sh | sudo tee update.logs &" // Run it and unlock the thing
	stdout, err := tools.ExecCommand(cmd, true)

	log.Printf("[INFO ] Updating the edge: %s", stdout)

	updateEdgeInProgress = false
	return err

	// out = strings.Split(strings.TrimSpace(stdout), "\n")

}

/*-----------------------------*/

func getUpdateEdgeStatus() {

	if !updateEdgeInProgress {
		return
	}

	appID := "wazigate-edge"
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

	cmd := "cat update.logs"
	stdout, err := tools.ExecCommand(cmd, false)
	if err != nil {
		stdout = ""
	}

	/*-----------*/

	installingAppStatus[appStatusIndex].log = stdout
}

/*-----------------------------*/

func installApp(imageName string) (string, error) {

	var msg string
	var err error

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

	//-->

	/*-----------*/

	appID := repoName + "." + appName

	appFullPath := filepath.Join(appsDir, appID)

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

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(out)
	outp := buf.String()

	installingAppStatus[appStatusIndex].log += outp

	if err != nil {
		installingAppStatus[appStatusIndex].done = true
		msg = "Download Failed!"
		return msg, err
	}

	/*-----------*/

	// out, err = tools.SockPostReqest( dockerSocketAddress, "images/create", "{\"Image\": \""+ imageName +"\"}")
	cmd := "docker create " + imageName
	containerID, err := tools.ExecCommand(cmd, true)

	if err != nil {
		installingAppStatus[appStatusIndex].done = true

		msg = err.Error()
		return msg, err
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
	if err := os.MkdirAll(appFullPath, 0755); err != nil {
		if !os.IsExist(err) {
			return "", fmt.Errorf("the Wazigate App directory could not be created %w", err)
		}
	}

	if err != nil {
		installingAppStatus[appStatusIndex].done = true

		msg = err.Error()
		return msg, err
	}

	/*-----------*/

	filecontent, _, err := cli.CopyFromContainer(context.Background(), containerID, "/var/lib/waziapp/")

	err = tools.Untar(appFullPath+"/", filecontent)

	//installingAppStatus[appStatusIndex].log += outp

	if err != nil {
		installingAppStatus[appStatusIndex].done = true

		msg = "untar file extraction failed!"
		return msg, err
	}

	/*-----------*/

	err = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{})

	if err != nil {
		installingAppStatus[appStatusIndex].done = true

		msg = "remove temporary container failed"
		return msg, err
	}
	/*-----------*/

	cmd = "unzip -o " + appFullPath + "/index.zip -d " + appFullPath
	outp, err = tools.ExecCommand(cmd, true)

	if err != nil {
		installingAppStatus[appStatusIndex].log += outp
		installingAppStatus[appStatusIndex].done = true

		msg = "Could not unzip `index.zip`!"
		return msg, err
	}

	/*-----------*/

	cmd = "rm -f " + appFullPath + "/index.zip"
	outp, _ = tools.ExecCommand(cmd, true)

	/*-----------*/

	// Pulling the dependencies
	cmd = "cd \"" + appFullPath + "\" && docker-compose pull && docker-compose up -d --no-build"
	outp, err = tools.ExecCommand(cmd, true)

	installingAppStatus[appStatusIndex].log += "\nDownloading the dependencies...\n"
	installingAppStatus[appStatusIndex].log += outp

	if err != nil {
		installingAppStatus[appStatusIndex].done = true

		msg = "Failed to download the dependencies!"
		return msg, err
	}

	/*-----------*/

	/*outJson, err := json.Marshal( out)
	if( err != nil) {
		log.Printf( "[ERR  ] %s", err.Error())
	}/**/

	installingAppStatus[appStatusIndex].log += "\nAll done :)"
	installingAppStatus[appStatusIndex].done = true

	return "Install successfull", nil

	/*-----------------------------*/
}

/*-----------------------------*/

func uninstallApp(appID string, keepConfig bool) error {

	appFullPath := appsDir + appID

	cmd := "cd \"" + appFullPath + "\" && IMG=$(docker-compose images -q) && docker-compose rm -fs && docker rmi -f $IMG; "
	if keepConfig {

		cmd += "rm ./package.json;"

	} else {

		cmd += "docker system prune -f && rm -r ../../" + appID
		//We use this path to make sure to delete the app folder if it really exist and not to delete the entire app folder or something else
	}

	out, err := tools.ExecCommand(cmd, true)

	log.Printf("[APP  ] DELETE App: %s\n\t%v\n", appID, out)

	return err
}

/*-----------------------------*/
