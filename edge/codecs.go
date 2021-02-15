package edge

import (
	"io"
	"net/http"
)

type Codec interface {
	UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error
	MarshalDevice(deviceID string, headers http.Header, w io.Writer) error
}

var Codecs = map[string]Codec{}
