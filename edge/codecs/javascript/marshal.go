package executor

import (
	"io"
	"net/http"

	"github.com/Waziup/wazigate-edge/edge"
)

var errNotImplemented = edge.NewError(500, "JavaScript Encoder is not implemented yet")

func (JavaScriptExecutor) MarshalDevice(script *edge.ScriptCodec, deviceID string, headers http.Header, w io.Writer) error {

	return errNotImplemented
}
