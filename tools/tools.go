package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() error {
	return nil
}

func ReadAll(rc io.ReadCloser) ([]byte, error) {
	defer rc.Close()

	if cb, ok := rc.(*ClosingBuffer); ok {
		return cb.Bytes(), nil
	}

	return ioutil.ReadAll(rc)
}

func SendJSON(resp http.ResponseWriter, obj interface{}) {

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		http.Error(resp, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write(data)
}

////////////////////////////////////////////////////////////////////////////////

// from https://gist.github.com/rucuriousyet/ab2ab3dc1a339de612e162512be39283
// getMacAddr gets the MAC hardware
// address of the host machine
func GetMACAddr() (addr string) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				// Don't use random as we have a real address
				addr = i.HardwareAddr.String()
				break
			}
		}
	}
	return
}

/*-----------------------------*/

// ExecOnHostWithLogs runs bash commands on the host through a unix socket
func ExecOnHostWithLogs(cmd string, withLogs bool) (string, error) {

	if withLogs {
		log.Printf("[Exec  ]: Host Command [ %s ]", cmd)
	}

	//Later we may change this with an env var
	return SockPostReqest("/var/run/wazigate-host.sock", "cmd", cmd)
}

/*-----------------------------*/

// SockGetReqest makes a request to a unix socket
// ex:	SockGetReqest( "/var/run/wazigate-host.sock", "/")
func SockGetReqest(socketAddr string, API string) (string, error) {

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketAddr)
			},
		},
	}

	response, err := httpc.Get("http://localhost/" + API)

	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return "", err
	}

	if response.StatusCode != 200 {
		log.Printf("[Err]: Status Code: %v ", response.StatusCode)
		return "", errors.New(response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return "", err
	}
	return string(body), nil
}

/*-----------------------------*/

// SockPostReqest makes a POST request to a unix socket
// ex (post Request):	SockPostReqest( "/var/run/wazigate-host.sock", "cmd", "ls -a")
func SockPostReqest(socketAddr string, API string, postValues string) (string, error) {

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketAddr)
			},
		},
	}

	response, err := httpc.Post("http://localhost/"+API, "application/json", strings.NewReader(postValues))

	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return "", err
	}

	if response.StatusCode != 200 {
		log.Printf("[Err]: Status Code: %v ", response.StatusCode)
		return "", errors.New(response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("[Err   ]: %s ", err.Error())
		return "", err
	}
	return string(body), nil
}

/*-----------------------------*/
