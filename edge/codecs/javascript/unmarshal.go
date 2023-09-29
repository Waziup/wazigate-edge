package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
)

type DecoderOutput struct {
	Data     map[string]any `json:"data"`
	Warnings []string       `json:"warnings"`
	Errors   []string       `json:"errors"`
}

var errNodeUnavaliable = edge.NewError(500, "the Node.js 'node' cmd was not found")
var errNoOutput = edge.NewError(500, "the script exited prematurely without producing any output")
var errInvalidOutput = edge.NewError(500, "the script exited prematurely")

const scriptDeadline = 0 // time.Second // 200 * time.Millisecond // 0

const scriptFooter1 = `
//*/
function main() {
  const bytes = new Uint8Array([`
const scriptFooter2 = `]);
  const fPort = `
const scriptFooter3 = `;

  if(typeof Decoder === "function") {
	const o = Decoder(bytes, fPort);
    print("\n$!\x01"+JSON.stringify(o));
    return;
  }
  if(typeof Decode === "function") {
	const o = Decode(fPort, bytes);
    print("\n$!\x01"+JSON.stringify(o));
    return;
  }
  if(typeof decodeUplink ==="function") {
	const o = decodeUplink({fPort,bytes});
	print("\n$!\x02"+JSON.stringify(o));
  }
  

  print("\n$!\x04");
}
main();`

var outRegexp = regexp.MustCompile(`\n\$\!.+\n`)

func PostSensorValues(output DecoderOutput, script *edge.ScriptCodec, device *edge.Device, deviceID string) error {
	now := time.Now()
	// // TEST
	// output = DecoderOutput{
	// 	Data: map[string]interface{}{
	// 		"temperature": 50,   // Fahrenheit
	// 		"windSpeed":   17.5, // knots
	// 	},
	// 	Warnings: []string{"1st_warning", "2nd_warning"},
	// 	Errors:   []string{"1st_err", "2nd_err"},
	// }

OUTPUT:
	for sensorID, value := range output.Data {
		for _, sensor := range device.Sensors {
			if sensor.ID == sensorID {
				// Sensor values
				_, err := edge.PostSensorValue(deviceID, sensorID, edge.NewValue(value, now))
				if err != nil {
					return edge.NewErrorf(500, "Can not create sensor value: %s", err)
				}
				// Meta data
				metadata := edge.Meta{
					"Warnings": strings.Join(output.Warnings, " "),
					"Errors":   strings.Join(output.Errors, " "),
				}
				err = edge.SetSensorMeta(deviceID, sensorID, metadata)
				if err != nil {
					return edge.NewErrorf(500, "Can not set sensor meta: %s", err)
				}
				continue OUTPUT
			}
		}
		err := edge.PostSensor(deviceID, &edge.Sensor{
			ID:   sensorID,
			Name: sensorID,
			Meta: edge.Meta{
				"createdBy": "ScriptCodec:" + script.ID,
				"warnings":  nil,
				"errors":    nil,
			},
			Value: value,
		})
		if err != nil {
			return edge.NewErrorf(500, "Can not create sensor: %s", err)
		}
	}
	return nil
}

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

	tempScript, err := os.CreateTemp("", "codec")
	if err != nil {
		return err
	}
	defer os.Remove(tempScript.Name())

	tempScript.WriteString(script.Script)
	tempScript.WriteString(scriptFooter1)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	for i, d := range data {
		if i != 0 {
			tempScript.WriteString(", ")
		}
		tempScript.WriteString(strconv.Itoa(int(d)))
	}

	tempScript.WriteString(scriptFooter2)

	port, _ := strconv.Atoi(headers.Get("X-LoRaWAN-FPort"))
	// will be '0' on any error
	tempScript.WriteString(strconv.Itoa(port))
	tempScript.WriteString(scriptFooter3)

	if err := tempScript.Close(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "qjs", "--script", tempScript.Name())
	var stdout bytes.Buffer
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

	// Decode, Decoder
	case 0x01:
		outputJSON := match[4 : len(match)-1]
		var data = map[string]interface{}{}

		if err := json.Unmarshal(outputJSON, &data); err != nil {
			return edge.NewErrorf(500, "Err executing script: 'Decoder' returned an invalid object:\n> %s.\n%s", outputJSON, prefix("> ", string(log)))
		}
		if PostSensorValues(DecoderOutput{Data: data}, script, device, deviceID); err != nil {
			return edge.NewErrorf(500, "Err posting unmarshaled values to wazigate-edge:\n> %s.\n%s", outputJSON, prefix("> ", string(log)))
		}

	// DecodeUplink
	case 0x02:
		outputJSON := match[4 : len(match)-1]
		var output DecoderOutput
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			return edge.NewErrorf(500, "Err executing script: 'Decoder' returned an invalid object:\n> %s.\n%s", outputJSON, prefix("> ", string(log)))
		}
		if PostSensorValues(output, script, device, deviceID); err != nil {
			return edge.NewErrorf(500, "Err posting unmarshaled values to wazigate-edge:\n> %s.\n%s", outputJSON, prefix("> ", string(log)))
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
