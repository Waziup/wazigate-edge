package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
)

var errNodeUnavaliable = edge.NewError(500, "the Node.js 'node' cmd was not found")
var errNoOutput = edge.NewError(500, "the script exited prematurely without producing any output")
var errInvalidOutput = edge.NewError(500, "the script exited prematurely")

const scriptDeadline = 0 // time.Second // 200 * time.Millisecond // 0

const scriptFooter1 = `
//*/
if(typeof Decoder !== "function") {
	process.stdout.write("\n$!\x03"+(typeof Decoder)+"\n");
	process.exit(0);
}
const o=Decoder(new Uint8Array([`
const scriptFooter2 = `]), `
const scriptFooter3 = `);
process.stdout.write("\n$!\x01"+JSON.stringify(o)+"\n");
process.exit(0);`

var outRegexp = regexp.MustCompile(`\n\$\!.+\n`)

func (JavaScriptExecutor) UnmarshalDevice(script *edge.ScriptCodec, deviceID string, headers http.Header, r io.Reader) error {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		return edge.NewError(404, "no device with that id")
	}

	ctx := context.Background()
	if scriptDeadline != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "node")
	var stdin bytes.Buffer
	var stdout bytes.Buffer

	stdin.WriteString(script.Script)
	stdin.WriteString(scriptFooter1)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	for i, d := range data {
		if i != 0 {
			stdin.WriteString(", ")
		}
		stdin.WriteString(strconv.Itoa(int(d)))
	}

	stdin.WriteString(scriptFooter2)

	port, _ := strconv.Atoi(headers.Get("X-LoRaWAN-FPort"))
	// will be '0' on any error
	stdin.WriteString(strconv.Itoa(port))
	stdin.WriteString(scriptFooter3)

	cmd.Stdin = &stdin
	cmd.Stderr = &stdout
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if err == exec.ErrNotFound {
			return errNodeUnavaliable
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			return edge.NewErrorf(500, "the 'node' cmd returned with status %d:\n%s", exitError.ExitCode(), prefix("> ", stdout.String()))
		}
		return edge.NewErrorf(500, "exec failed: %s", err.Error())
	}
	if stdout.Len() == 0 {
		return errNoOutput
	}
	log := stdout.Bytes()
	i := outRegexp.FindIndex(log)
	if i == nil {
		return edge.NewErrorf(500, "Err executing script: The script exited prematurely\n%s", prefix("> ", string(log)))
	}
	match := log[i[0]:i[1]]
	log = append(log[:i[0]], log[i[1]:]...)

	switch match[3] {
	case 0x03:
		typeOfDecoder := string(match[4 : len(match)-1])
		if typeOfDecoder == "undefined" {
			return edge.NewErrorf(500, "Err executing script: Missing 'Decoder' function.\n%s", prefix("> ", string(log)))
		}
		return edge.NewErrorf(500, "Err executing script: 'Decoder' is not a function, it's '%s'.\n%s", typeOfDecoder, prefix("> ", string(log)))

	case 0x01:
		outputJSON := match[4 : len(match)-1]
		var output = map[string]interface{}{}
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			return edge.NewErrorf(500, "Err executing script: 'Decoder' returned an invalid object:\n> %s.\n%s", outputJSON, prefix("> ", string(log)))
		}
		now := time.Now()

	OUTPUT:
		for sensorID, value := range output {
			for _, sensor := range device.Sensors {
				if sensor.ID == sensorID {
					_, err = edge.PostSensorValue(deviceID, sensorID, edge.NewValue(value, now))
					if err != nil {
						return edge.NewErrorf(500, "Can not create sensor value: %s", err)
					}
					continue OUTPUT
				}
			}
			err = edge.PostSensor(deviceID, &edge.Sensor{
				ID:   sensorID,
				Name: sensorID,
				Meta: edge.Meta{
					"createdBy": "ScriptCodec:" + script.ID,
				},
				Value: value,
			})
			if err != nil {
				return edge.NewErrorf(500, "Can not create sensor: %s", err)
			}
		}

	default:
		return edge.NewErrorf(500, "Err executing script: Unknown code 0x%02x.\n%s", prefix("> ", stdout.String()))
	}
	return nil
}

func prefix(prefix string, str string) string {
	var b strings.Builder
	if str == "" {
		return ""
	}
	for true {
		b.WriteString(prefix)
		i := strings.IndexByte(str, '\n')
		if i == -1 {
			b.WriteString(str)
			return b.String()
		}
		b.WriteString(str[:i])
		str = str[i+1:]
	}
	return "" // dead
}
