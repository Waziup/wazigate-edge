package codec

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/waziup/xlpp"
)

func (XLPPCodec) ReadActuators(deviceID string, headers http.Header, w io.Writer) error {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		return err
	}

	writer := xlpp.NewWriter(w)
	for _, actuator := range device.Actuators {
		channel := xlppChan(actuator.Meta)
		if channel == -1 {
			log.Printf("Err Codec XLPP: Actuator %s/%s: No xlppChan meta", device.ID, actuator.ID)
			continue
		}

		if channel := xlppChan(actuator.Meta); channel != -1 {
			quantity, _ := actuator.Meta["quantity"].(string)
			unit, _ := actuator.Meta["unit"].(string)
			t := typeFromDef(quantity, unit)
			if t == 255 {
				log.Printf("Err Codec XLPP: Actuator %s/%s: No type for (quantity:%q, unit:%q)", device.ID, actuator.ID, quantity, unit)
				continue
			}
			v := xlpp.Registry[t]()
			d, _ := json.Marshal(actuator.Value)
			err := json.Unmarshal(d, v)
			if err != nil {
				log.Printf("Err Codec XLPP: Actuator %s/%s: Can not assign value (quantity:%q, unit:%q) %q to %s", device.ID, actuator.ID, quantity, unit, actuator.Value, typeName(v))
				continue
			}
			writer.Add(channel, v)
		}
	}

	return nil
}

func typeFromValue(v interface{}) xlpp.Type {
	switch v.(type) {
	case bool:
		return xlpp.TypeBool
	case string:
		return xlpp.TypeString
	case int, int64:
		return xlpp.TypeInteger
	default:
		return 255 // unknown
	}
}
