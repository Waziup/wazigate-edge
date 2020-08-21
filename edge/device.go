package edge

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Device represents a Waziup Device.
type Device struct {
	Name      string      `json:"name" bson:"name"`
	ID        string      `json:"id" bson:"_id"`
	Sensors   []*Sensor   `json:"sensors" bson:"sensors"`
	Actuators []*Actuator `json:"actuators" bson:"actuators"`
	Modified  time.Time   `json:"modified" bson:"modified"`
	Created   time.Time   `json:"created" bson:"created"`
	Meta      Meta        `json:"meta" bson:"meta"`

	jsonSelect []string
}

func (device *Device) SetJSONSelect(s []string) {
	device.jsonSelect = s
}

// DevicesQuery is used to range or limit query results.
type Query struct {
	Limit  int64
	Size   int64
	Meta   []string
	Select []string
}

func (device *Device) MarshalJSON() ([]byte, error) {
	clone := map[string]interface{}{}
	t := reflect.ValueOf(device).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.CanSet() {
			key := t.Type().Field(i).Tag.Get("json")
			if key != "id" {
				if device.jsonSelect != nil {
					selected := false
					for _, field := range device.jsonSelect {
						if field == key || strings.HasPrefix(field, key+".") {
							selected = true
							break
						}
					}
					if !selected {
						continue
					}
					if key == "sensors" {
						var jsonSelect []string
						for _, field := range device.jsonSelect {
							if strings.HasPrefix(field, "sensors.") {
								jsonSelect = append(jsonSelect, field[8:])
							}
						}
						sensors := f.Interface().([]*Sensor)
						for _, sensor := range sensors {
							sensor.jsonSelect = jsonSelect
						}
					}
					if key == "actuators" {
						var jsonSelect []string
						for _, field := range device.jsonSelect {
							if strings.HasPrefix(field, "actuators.") {
								jsonSelect = append(jsonSelect, field[8:])
							}
						}
						actuators := f.Interface().([]*Actuator)
						for _, actuator := range actuators {
							actuator.jsonSelect = jsonSelect
						}
					}
				}
			}
			val := f.Interface()
			clone[key] = val
		}
	}
	return json.Marshal(clone)
}

// Parse parses the HTTP Request into the Query parameters.
var errBadLimitParam = errors.New("query ?limit=.. is mal formatted")
var errBadSizeParam = errors.New("query ?size=.. is mal formatted")

// Parse reads url.Values into the DevicesQuery.
func (query *Query) Parse(values url.Values) error {
	var param string
	var err error
	if param = values.Get("limit"); param != "" {
		query.Limit, err = strconv.ParseInt(param, 10, 64)
		if err != nil {
			return errBadLimitParam
		}
	}
	if param = values.Get("size"); param != "" {
		query.Size = parseSize(param)
		if query.Size == -1 {
			return errBadSizeParam
		}
	}
	if param = values.Get("meta"); param != "" {
		query.Meta = strings.Split(param, ",")
		for i, str := range query.Meta {
			query.Meta[i] = strings.TrimSpace(str)
		}
	}
	if param = values.Get("select"); param != "" {
		query.Select = strings.Split(param, ",")
		for i, str := range query.Select {
			query.Select[i] = strings.TrimSpace(str)
		}
	}
	return nil
}

var localID string

// LocalID returns the ID of this device
func LocalID() string {
	if localID != "" {
		return localID
	}

	localID = os.Getenv("WAZIGATE_ID")
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
	jsonSelect := iter.device.jsonSelect
	if iter.dbIter.Next(&iter.device) {
		iter.device.jsonSelect = jsonSelect
		return &iter.device, iter.dbIter.Err()
	}
	return nil, io.EOF
}

// Close closes the iterator.
func (iter *DeviceIterator) Close() error {
	return iter.dbIter.Close()
}

// GetDevices returns an iterator over all devices.
func GetDevices(query *Query) *DeviceIterator {

	sel := bson.M{}
	if query != nil {
		if len(query.Meta) != 0 {
			for _, name := range query.Meta {
				sel["meta."+name] = bson.M{"$exists": true}
			}
		}
	}
	q := dbDevices.Find(sel)
	var jsonSelect []string
	if query != nil {
		if query.Select != nil {
			jsonSelect = query.Select
			s := bson.M{"_id": 1}
			for _, field := range query.Select {
				s[field] = 1
				if strings.HasPrefix(field, "sensors.") {
					s["sensors"] = 1
					s["sensors.id"] = 1
				}
				if strings.HasPrefix(field, "actuators.") {
					s["actuators"] = 1
					s["actuators.id"] = 1
				}
			}
			q.Select(s)
		}
		if query.Limit != 0 {
			q.Limit(int(query.Limit))
		}
	}

	return &DeviceIterator{
		dbIter: q.Iter(),
		device: Device{
			jsonSelect: jsonSelect,
		},
	}
}

// GetDevice returns the Waziup device with that id.
func GetDevice(deviceID string) (*Device, error) {
	var device Device

	query := dbDevices.FindId(deviceID)
	if err := query.One(&device); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "Database Error: " + err.Error()}
	}
	return &device, nil
}

// GetDeviceName returns the name of that device.
func GetDeviceName(deviceID string) (string, error) {
	var device Device
	err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(
		bson.M{
			"name": 1,
		},
	).One(&device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return "", ErrNotFound
		}
		return "", CodeError{500, "database error: " + err.Error()}
	}
	return device.Name, nil
}

// GetDeviceMeta returns the metadata of that device.
func GetDeviceMeta(deviceID string) (map[string]interface{}, error) {
	var device Device
	err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(
		bson.M{
			"meta": 1,
		},
	).One(&device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}
	return device.Meta, nil
}

var noTime = time.Time{}

// PostDevice creates a new device a the database.
func PostDevice(device *Device) error {
	var err error

	if device.ID == "" {
		device.ID = bson.NewObjectId().Hex()
	}

	now := time.Now()
	device.Created = now
	device.Modified = now

	if device.Sensors != nil {
		for _, sensor := range device.Sensors {
			if sensor.ID == "" {
				sensor.ID = bson.NewObjectId().Hex()
			}
			sensor.Created = now
			sensor.Modified = now
			if sensor.Value == nil {
				sensor.Time = nil
			} else if sensor.Time == nil {
				sensor.Time = &now
			}
		}
	}
	if device.Actuators != nil {
		for _, actuator := range device.Actuators {
			if actuator.ID == "" {
				actuator.ID = bson.NewObjectId().Hex()
			}
			actuator.Created = now
			actuator.Modified = now
			if actuator.Value == nil {
				actuator.Time = nil
			} else if actuator.Time == nil {
				actuator.Time = &now
			}
		}
	}

	err = dbDevices.Insert(&device)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) != 0 {
		values := make([]interface{}, 0, len(device.Sensors))
		for _, sensor := range device.Sensors {
			if sensor.Value != nil {
				values = append(values, sValue{
					ID:       newID(*sensor.Time),
					DeviceID: device.ID,
					SensorID: sensor.ID,
					Value:    sensor.Value,
				})
			}
		}
		dbSensorValues.Insert(values...)
	}

	if len(device.Actuators) != 0 {
		values := make([]interface{}, 0, len(device.Actuators))
		for _, actuator := range device.Actuators {
			if actuator.Value != nil {
				values = append(values, aValue{
					ID:         newID(*actuator.Time),
					DeviceID:   device.ID,
					ActuatorID: actuator.ID,
					Value:      actuator.Value,
				})
			}
		}
		dbActuatorValues.Insert(values...)
	}

	return nil
}

// SetDeviceName changes a device name.
func SetDeviceName(deviceID string, name string) (Meta, error) {

	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(
		bson.M{
			"meta": 1,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"modified": time.Now(),
				"name":     name,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}
	return device.Meta, nil
}

// SetDeviceMeta changes a device metadata.
func SetDeviceMeta(deviceID string, meta Meta) error {

	var unset = bson.M{}
	var set = bson.M{
		"modified": time.Now(),
	}
	for key, value := range meta {
		if value == nil {
			unset["meta."+key] = 1
		} else {
			set["meta."+key] = value
		}
	}

	err := dbDevices.UpdateId(deviceID, bson.M{
		"$set":   set,
		"$unset": unset,
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}
	return nil
}

var errDeleteLocal = CodeError{400, "Can not delete the Gateway itself"}

// DeleteDevice removes the device and all sensor and actuator values from the database.
// This returns the removed device and the number of sensor and actuator values that were removed.
func DeleteDevice(deviceID string) (*Device, int, int, error) {

	if deviceID == LocalID() {
		return nil, 0, 0, errDeleteLocal
	}

	var device Device
	err := dbDevices.FindId(deviceID).One(&device)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, 0, 0, ErrNotFound
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
			return nil, numS, numA, ErrNotFound
		}
		return nil, numS, numA, CodeError{500, "database error: " + err.Error()}
	}

	return nil, numS, numA, nil
}
