/*
*	@author: mojtaba.eskandari@waziup.org Nov 25th 2019
*	@A deamon to execute commands on host
 */
package tools

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

/*-------------------------*/

// sockAddr represents the unix socket for this service
const sockAddr = "/tmp/wazigate-host.sock"

//const sockAddr = "./wazigate-host.sock"

/*-------------------------*/

var execPath = "/var/lib/wazigate"

func ServeHost() {

	if err := os.RemoveAll(sockAddr); err != nil {
		log.Fatal(err)
	}

	server := http.Server{
		Handler: http.HandlerFunc(serveCommand),
	}

	unixListener, err := net.Listen("unix", sockAddr)
	if err != nil {
		log.Fatal("[ERR  ] listen error:", err)
	}
	log.Printf("[INFO ] Serving... on socket: [%v]", sockAddr)

	defer unixListener.Close()
	server.Serve(unixListener)
}

func serveCommand(resp http.ResponseWriter, req *http.Request) {

	cmd, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "Bad Request", http.StatusBadRequest)
		return
	}

	out, err := ExecCommand(string(cmd), true)
	if err != nil {
		log.Printf("[ERR  ] executing [ %s ] command. \n\tError: [ %s ]", cmd, err.Error())
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Write([]byte(out))
}

func ExecCommand(cmd string, withLogs bool) (out string, err error) {
	log.Printf("[INFO ] executing [ %s ] ", cmd)

	if withLogs {
		log.Printf("[     ] > %s", cmd)
	}
	exe := exec.Command("sh", "-c", string(cmd))
	//  exe.Dir = execPath
	stdout, err := exe.Output()
	if withLogs {
		if err != nil {
			log.Printf("[ERR  ] %s", err)
		} else {
			log.Printf("[     ] < %s", stdout)
		}
	}

	out = strings.Trim(string(stdout), " \n\t\r")

	return out, err
}

func Shell(dir string, sh string) (string, error) {
	cmd := exec.Command("sh", "-c", sh)
	cmd.Dir = dir
	log.Printf("[     ] > %s", sh)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[ERR  ] > %s (%s)", out, err)
	} else {
		log.Printf("[     ] > %s", sh)
	}
	return string(out), err
}
