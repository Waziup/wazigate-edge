package tools

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
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
