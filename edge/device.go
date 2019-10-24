package edge

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Device represents a Waziup Device.
type Device struct {
	Name      string                 `json:"name" bson:"name"`
	ID        string                 `json:"id" bson:"_id"`
	Sensors   []*Sensor              `json:"sensors" bson:"sensors"`
	Actuators []*Actuator            `json:"actuators" bson:"actuators"`
	Modified  time.Time              `json:"modified" bson:"modified"`
	Created   time.Time              `json:"created" bson:"created"`
	Meta      map[string]interface{} `json:"meta" bson:"meta"`
}

var localID string

// LocalID returns the ID of this device
func LocalID() string {
	if localID != "" {
		return localID
	}

	interfs, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, interf := range interfs {
		addr := interf.HardwareAddr.String()
		if addr != "" {
			localID = strings.ReplaceAll(addr, ":", "")
			return localID
		}
	}
	return ""
}

// DeviceIterator iterates over devices. Call .Next() to get the next device.
type DeviceIterator struct {
	device Device
	dbIter *mgo.Iter
}

// Next returns the next device or nil.
func (iter *DeviceIterator) Next() (*Device, error) {

	if iter.dbIter.Next(&iter.device) {
		return &iter.device, iter.dbIter.Err()
	}
	return nil, io.EOF
}

// Close closes the iterator.
func (iter *DeviceIterator) Close() error {
	return iter.dbIter.Close()
}

// GetDevices returns an iterator over all devices.
func GetDevices() *DeviceIterator {

	iter := dbDevices.Find(nil).Iter()
	return &DeviceIterator{
		dbIter: iter,
	}
}

// GetDevice returns the Waziup device with that id.
func GetDevice(deviceID string) (*Device, error) {
	var device Device

	query := dbDevices.FindId(deviceID)
	if err := query.One(&device); err != nil {
		if err == mgo.ErrNotFound {
			return nil, errNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}
	return &device, nil
}

// GetDeviceName returns the name of that device.
func GetDeviceName(deviceID string) (string, error) {
	var device Device
	query := dbDevices.FindId(deviceID)
	query.Select("name")
	if err := query.One(&device); err != nil {
		if err == mgo.ErrNotFound {
			return "", errNotFound
		}
		return "", CodeError{500, "database error: " + err.Error()}
	}
	return device.Name, nil
}

// GetDeviceMeta returns the metadata of that device.
func GetDeviceMeta(deviceID string) (map[string]interface{}, error) {
	var device Device
	query := dbDevices.FindId(deviceID)
	query.Select("meta")
	if err := query.One(&device); err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}
	return device.Meta, nil
}

// PostDevice creates a new device a the database.
func PostDevice(device *Device) error {
	var err error

	err = dbDevices.Insert(&device)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) != 0 {
		sensors := make([]interface{}, 0, len(device.Sensors))
		for _, sensor := range device.Sensors {
			if sensor.Value != nil {
				sensors = append(sensors, sValue{
					ID:       newID(sensor.Time),
					DeviceID: device.ID,
					SensorID: sensor.ID,
					Value:    sensor.Value,
				})
			}
		}
		dbSensorValues.Insert(sensors...)
	}

	if len(device.Actuators) != 0 {
		actuators := make([]interface{}, 0, len(device.Actuators))
		for _, actuator := range device.Actuators {
			if actuator.Value != nil {
				actuators = append(actuators, aValue{
					ID:         newID(actuator.Time),
					DeviceID:   device.ID,
					ActuatorID: actuator.ID,
					Value:      actuator.Value,
				})
			}
		}
		dbActuatorValues.Insert(actuators...)
	}

	return nil
}

// SetDeviceName changes a device name.
func SetDeviceName(deviceID string, name string) error {

	err := dbDevices.UpdateId(deviceID, bson.M{
		"$set": bson.M{
			"modified": time.Now(),
			"name":     name,
		},
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			return errNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}
	return nil
}

// DeleteDevice removes the device and all sensor and actuator values from the database.
// This returns the removed device and the number of sensor and actuator values that were removed.
func DeleteDevice(deviceID string) (*Device, int, int, error) {

	var device Device
	err := dbDevices.FindId(deviceID).One(&device)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, 0, 0, errNotFound
		}
		return nil, 0, 0, CodeError{500, "database error: " + err.Error()}
	}

	err = dbDevices.RemoveId(deviceID)
	infoS, _ := dbSensorValues.RemoveAll(bson.M{"deviceId": deviceID})
	infoA, _ := dbActuatorValues.RemoveAll(bson.M{"deviceId": deviceID})
	numS := infoS.Removed
	numA := infoA.Removed

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, numS, numA, errNotFound
		}
		return nil, numS, numA, CodeError{500, "database error: " + err.Error()}
	}

	return nil, numS, numA, nil
}
