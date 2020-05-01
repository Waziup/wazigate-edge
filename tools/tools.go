package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
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

	socketAddr := os.Getenv("WAZIGATE_HOST_ADDR")
	if socketAddr == "" {
		socketAddr = "/var/run/wazigate-host.sock" // Default address for the Host
	}

	out, err := SockPostReqest(socketAddr, "cmd", cmd)
	return string(out), err
}

/*-----------------------------*/

// // exeCmdWithLogs runs bash commands in the container
// func exeCmd( cmd string) ( string, error) {

// 	exe := exec.Command( "sh", "-c", cmd)
//     stdout, err := exe.Output()

//     if( err != nil) {
//         return "", err
// 	}
// 	return strings.Trim( string( stdout), " \n\t\r"), nil
// }

/*-----------------------------*/

// SockDeleteReqest makes a DELETE request to a unix socket
// ex:	SockDeleteReqest( "/var/run/wazigate-host.sock", "containers/waziup.wazigate-test")
func SockDeleteReqest(socketAddr string, API string) ([]byte, error) {

	response, err := SocketReqest(socketAddr, API, "DELETE", "", nil)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, err
	}

	resBody, err := ioutil.ReadAll(response.Body)

	if response != nil {
		response.Body.Close()
	}

	if err != nil {
		return nil, err
	}
	return resBody, nil
}

/*-----------------------------*/

// SockGetReqest makes a GET request to a unix socket
// ex:	SockGetReqest( "/var/run/wazigate-host.sock", "/")
func SockGetReqest(socketAddr string, API string) ([]byte, error) {

	response, err := SocketReqest(socketAddr, API, "GET", "", nil)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, err
	}

	resBody, err := ioutil.ReadAll(response.Body)
	if response != nil {
		response.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return resBody, nil
}

/*-----------------------------*/

// SockPostReqest makes a POST request to a unix socket
// ex (post Request):	SockPostReqest( "/var/run/wazigate-host.sock", "cmd", "ls -a")
func SockPostReqest(socketAddr string, API string, postValues string) ([]byte, error) {

	response, err := SocketReqest(socketAddr, API, "POST", "application/json", strings.NewReader(postValues))

	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		return nil, err
	}

	resBody, err := ioutil.ReadAll(response.Body)
	if response != nil {
		response.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	return resBody, nil
}

/*-----------------------------*/

// SocketReqest makes a request to a unix socket
func SocketReqest(socketAddr string, url string, method string, contentType string, body io.Reader) (*http.Response, error) {

	log.Printf("[APP  ] Proxy `%s` %s \"%s\"", socketAddr, method, url)

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketAddr)
			},
			MaxIdleConns:    50,
			IdleConnTimeout: 4 * 60 * time.Second,
		},
	}

	req, err := http.NewRequest(method, "http://localhost/"+url, body)

	if err != nil {
		log.Printf("[APP  ] Proxy Err %s ", err.Error())
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	response, err := httpc.Do(req)

	if err != nil {
		log.Printf("[APP  ] Proxy Err %s ", err.Error())
		return nil, err
	}

	return response, nil
}

/*-----------------------------*/
