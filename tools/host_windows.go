/*
*	@author: mojtaba.eskandari@waziup.org Nov 25th 2019
*	@A deamon to execute commands on host
 */
package tools

import (
	"log"
	"net/http"
	"os/exec"
)

/*-------------------------*/

// sockAddr represents the unix socket for this service
const sockAddr = "/var/run/wazigate-host.sock"

//const sockAddr = "./wazigate-host.sock"

/*-------------------------*/

var execPath = "/var/lib/wazigate"

func ServeHost() {

	log.Println("[     ] Diese Funktion wird unter Windows nicht unterstützt.")
}

func serveCommand(resp http.ResponseWriter, req *http.Request) {

	log.Println("[     ] Diese Funktion wird unter Windows nicht unterstützt.")
}

func ExecCommand(cmd string, withLogs bool) (out string, err error) {
	log.Println("[     ] Diese Funktion wird unter Windows nicht unterstützt.")

	return "", nil
}

// Shell executes a shell command in a given directory.
// It returns the output and an error if any.
func Shell(dir string, sh string) (string, error) {
	cmd := exec.Command("cmd", "/c", sh)
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
