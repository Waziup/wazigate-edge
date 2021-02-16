package coedc

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Waziup/wazigate-edge/edge"
)

func init() {
	edge.Codecs["application/json"] = JSONCodec{}
}

type JSONCodec struct{}

func (JSONCodec) CodecName() string {
	return "JSON"
}

func (JSONCodec) MarshalDevice(deviceID string, headers http.Header, w io.Writer) error {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(device)
}

func (JSONCodec) UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		return err
	}

	var device2 edge.Device
	decoder := json.NewDecoder(r)
	err = decoder.Decode(&device2)
	if err != nil {
		return err
	}

DEVIC2_SENSORS:
	for _, sensor2 := range device2.Sensors {
		for _, sensor := range device.Sensors {
			if sensor.ID == sensor2.ID {
				continue DEVIC2_SENSORS
			}
		}
		if err = edge.PostSensor(deviceID, sensor2); err != nil {
			return err
		}
	}
DEVIC2_ACTUATORS:
	for _, actuators2 := range device2.Actuators {
		for _, actuator := range device.Actuators {
			if actuator.ID == actuators2.ID {
				continue DEVIC2_ACTUATORS
			}
		}
		if err = edge.PostActuator(deviceID, actuators2); err != nil {
			return err
		}
	}

	return nil
}
