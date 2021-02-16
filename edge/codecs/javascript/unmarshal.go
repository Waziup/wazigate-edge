package executor

import (
	"io"
	"net/http"

	"github.com/Waziup/wazigate-edge/edge"
)

func (JavaScriptExecutor) UnmarshalDevice(script *edge.ScriptCodec, deviceID string, headers http.Header, r io.Reader) error {

	return nil
}
