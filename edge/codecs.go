package edge

import (
	"io"
	"net/http"

	"github.com/waziup/xlpp"
)

type Codec interface {
	WriteDevice(deviceID string, headers http.Header, r io.Reader) error
}

var codecs = map[string]Codec{
	"application/x-xlpp": XLPPCodec{},
	"application/x-lpp":  XLPPCodec{LagacyMode: true},
}

type XLPPCodec struct {
	LagacyMode bool
}

func (XLPPCodec) WriteDevice(deviceID string, headers http.Header, r io.Reader) error {

	reader := xlpp.NewReader(r)
	for {
		channel, value, err := reader.Next()

	}

	return nil
}
