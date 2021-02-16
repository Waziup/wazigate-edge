package executor

import (
	"io"
	"net/http"

	"github.com/Waziup/wazigate-edge/edge"
)

func (JavaScriptExecutor) MarshalDevice(script *edge.ScriptCodec, deviceID string, headers http.Header, w io.Writer) error {

	return nil
}
