package codec

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"
	"github.com/waziup/xlpp"
)

func (XLPPCodec) UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error {

	device, err := edge.GetDevice(deviceID)
	if err != nil {
		return err
	}

	t := time.Now()

	reader := xlpp.NewReader(r)
	for {
		channel, value, err := reader.Next()
		if value == nil {
			return err
		}
		switch v := value.(type) {
		case *xlpp.Delay:
			t = t.Add(-time.Duration(*v))
		case *xlpp.Actuators:
			for i, t := range *v {
				if err := createActuator(device, i, t); err != nil {
					return err
				}
			}
		case *xlpp.ActuatorsWithChannel:
			for _, a := range *v {
				if err := createActuator(device, a.Channel, a.Type); err != nil {
					return err
				}
			}
		default:
			if err := createSensor(device, channel, v, t); err != nil {
				return err
			}
		}
	}
}

func createActuator(device *edge.Device, channel int, t xlpp.Type) error {
	for _, actuator := range device.Actuators {
		if xlppChan(actuator.Meta) == channel {
			return nil
		}
	}
	d := actuatorMapping[t]
	actuator := &edge.Actuator{
		Name: typeName(xlpp.Registry[t]()),
		Meta: edge.Meta{
			"kind":      d.Kind,
			"quantity":  d.Quantity,
			"unit":      d.Unit,
			"xlppChan":  channel,
			"createdBy": "codec:xlpp",
		},
	}
	err := edge.PostActuator(device.ID, actuator)
	if err != nil {
		return err
	}
	var noTime time.Time
	clouds.FlagActuator(device.ID, actuator.ID, clouds.ActionCreate, noTime, actuator.Meta)
	return nil
}

func createSensor(device *edge.Device, channel int, value xlpp.Value, t time.Time) error {
	for _, sensor := range device.Sensors {
		if xlppChan(sensor.Meta) == channel {
			v := edge.NewValue(value, t)
			meta, err := edge.PostSensorValue(device.ID, sensor.ID, v)
			if err != nil {
				return err
			}
			clouds.FlagSensor(device.ID, sensor.ID, clouds.ActionSync, v.Time, meta)
			return nil
		}
	}
	d := sensorMapping[value.XLPPType()]
	sensor := &edge.Sensor{
		Name:  typeName(value),
		Value: value,
		Time:  &t,
		Meta: edge.Meta{
			"kind":      d.Kind,
			"quantity":  d.Quantity,
			"unit":      d.Unit,
			"xlppChan":  channel,
			"createdBy": "codec:xlpp",
		},
	}
	err := edge.PostSensor(device.ID, sensor)
	if err != nil {
		return err
	}
	var noTime time.Time
	clouds.FlagSensor(device.ID, sensor.ID, clouds.ActionCreate, noTime, sensor.Meta)
	return nil
}

func typeName(v interface{}) (name string) {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		name = t.Elem().Name()
	} else {
		name = t.Name()
	}
	return strings.ToLower(name)
}

func xlppChan(m edge.Meta) int {
	c, ok := m["xlppChan"]
	if ok {
		if d, ok := c.(float64); ok {
			return int(d)
		}
		if d, ok := c.(int); ok {
			return int(d)
		}
	}
	return -1
}
