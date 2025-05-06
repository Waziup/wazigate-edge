package coedc

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Waziup/wazigate-edge/clouds"
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

var noTime time.Time

func (JSONCodec) UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error {
	now := time.Now()
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

				if sensor2.Value != nil {
					t := sensor2.Time
					if t == nil || t.IsZero() {
						t = &now
					}
					if _, err = edge.PostSensorValue(deviceID, sensor2.ID, edge.NewValue(sensor2.Value, *t)); err != nil {
						return err
					}
					clouds.FlagSensor(device.ID, sensor.ID, clouds.ActionSync, *t, sensor.Meta)
				}

				continue DEVIC2_SENSORS
			}
		}
		if err = edge.PostSensor(deviceID, sensor2); err != nil {
			return err
		}

		clouds.FlagSensor(device.ID, sensor2.ID, clouds.ActionCreate, noTime, sensor2.Meta)
	}
DEVIC2_ACTUATORS:
	for _, actuators2 := range device2.Actuators {
		for _, actuator := range device.Actuators {
			if actuator.ID == actuators2.ID {

				if actuators2.Value != nil {
					t := actuators2.Time
					if t == nil || t.IsZero() {
						t = &now
					}
					if _, err = edge.PostActuatorValue(deviceID, actuators2.ID, edge.NewValue(actuators2.Value, *t)); err != nil {
						return err
					}
					clouds.FlagActuator(device.ID, actuators2.ID, clouds.ActionSync, *t, actuator.Meta)
				}

				continue DEVIC2_ACTUATORS
			}
		}
		if err = edge.PostActuator(deviceID, actuators2); err != nil {
			return err
		}
		clouds.FlagActuator(deviceID, actuators2.ID, clouds.ActionCreate, noTime, actuators2.Meta)
	}

	return nil
}
