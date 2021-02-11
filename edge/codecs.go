package edge

import (
	"io"
	"net/http"
)

type Codec interface {
	WriteDevice(deviceID string, headers http.Header, r io.Reader) error
	ReadActuators(deviceID string, headers http.Header, w io.Writer) error
}

var Codecs = map[string]Codec{}
