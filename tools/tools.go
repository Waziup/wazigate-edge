package tools

import (
	"archive/tar"
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

// GetMACAddr gets the MAC hardware
// address of the host machine
// from https://gist.github.com/rucuriousyet/ab2ab3dc1a339de612e162512be39283
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

// GetIPAddr returns the non loopback local IP of the container
func GetIPAddr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

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

	log.Printf("[     ] Proxy `%s` %s \"%s\"", socketAddr, method, url)

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
		log.Printf("[     ] Proxy Err %s ", err.Error())
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	response, err := httpc.Do(req)

	if err != nil {
		log.Printf("[     ] Proxy Err %s ", err.Error())
		return nil, err
	}

	if response.StatusCode != 200 {
		err = fmt.Errorf("Err: " + response.Status)
	}

	return response, err
}

/*-----------------------------*/

type RequestBodyContextKey struct{}

func SetRequestBody(req *http.Request, body interface{}) {
	i := req.Context().Value(RequestBodyContextKey{})
	if i != nil {
		if b, ok := i.(*interface{}); ok {
			*b = body
		}
	}
}

/*-----------------------------*/

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
