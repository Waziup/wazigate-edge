package edge

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
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

// Query is used to range or limit query results.
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

/*--------------------------*/

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

	localID, err := GetConfig("gatewayID")
	if err != nil {
		log.Printf("[WARN ] LocalID: %s", err.Error())
		if err == mgo.ErrNotFound {
			log.Printf("[INFO ] Creating a new gateway ID...")
			localID = GenerateNewGatewayID()
			err := SetConfig("gatewayID", localID)
			if err != nil {
				log.Printf("[Err  ] Store New LocalID: %s", err.Error())
			}
		}
	}

	return localID
}

/*--------------------------*/

// GenerateNewGatewayID generated a new ID for the gateway that will be Unique and Random
func GenerateNewGatewayID() string {

	localIDPrefix := make([]byte, 12)

	//Get the mac address from the host
	mac, err := ioutil.ReadFile("/sys/class/net/eth0/address")

	if err != nil {
		log.Printf("[ERR  ] Get new GWID: %s", err.Error())
	} else {

		localIDPrefix, err = hex.DecodeString(strings.Replace(string(mac), ":", "", -1))
		if err != nil {
			log.Printf("[ERR  ] Get new GWID: %s", err.Error())
		}
	}

	const localIDLength = 8 // larger would have been better, but Chirpstack accepts exactly 8 byte in Hex format
	someBytes := make([]byte, localIDLength)

	for i := 0; i < len(localIDPrefix) && i < localIDLength; i++ {
		someBytes[i] = localIDPrefix[i]
	}

	rand.Seed(time.Now().UTC().UnixNano())
	for i := len(localIDPrefix); i < localIDLength; i++ {
		someBytes[i] = byte(rand.Intn(255))
	}

	return hex.EncodeToString(someBytes)

	/*newID := base64.StdEncoding.EncodeToString(someBytes)

	// Cleaning the ID from all unwanted chars

	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Printf("[ERR  ] Get new GWID: %s", err.Error())
		return ""
	}
	newID = reg.ReplaceAllString(newID, "")

	return newID[:16] /* Let's keep it 16 bytes */
}

/*--------------------------*/

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

// PostDevices creates a new device a the database.
func PostDevices(device *Device) error {
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

////////////////////////////////////////////////////////////////////////////////

// var errNoCodec = errors.New("Err Device has no codec set. Can not process 'application/octet-stream'.")
// var errBadCodec = errors.New("Err Device meta 'codec' is not a string. Can not process 'application/octet-stream'.")

// UnmarshalDevice writes complex data to the device.
// This might be JSON data, LoRaWAN XLPP payload or something else.
func UnmarshalDevice(deviceID string, headers http.Header, r io.Reader) error {

	_, codec, err := FindCodec(deviceID, headers.Get("Content-Type"))
	if err != nil {
		return err
	}
	return codec.UnmarshalDevice(deviceID, headers, r)
}

func FindCodec(deviceID string, contentType string) (name string, codec Codec, err error) {
	var ok bool

	warnNoDefaultCodec := false
	warnDefaultCodecUnavailable := false

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	contentTypes := strings.Split(contentType, ",")
	for _, name = range contentTypes {
		i := strings.IndexByte(name, ';')
		if i != -1 {
			name = name[:i]
		}
		if name == "*/*" {
			name = "application/json"
			codec = Codecs[name]
			break
		} else if name == "application/octet-stream" {
			meta, err := GetDeviceMeta(deviceID)
			if err != nil {
				return "", nil, err
			}
			defaultCodecName := meta["codec"]
			if defaultCodecName == nil {
				warnNoDefaultCodec = true
				continue
			}
			name, ok = defaultCodecName.(string)
			if !ok {
				warnNoDefaultCodec = true
				continue
			}
			if strings.ContainsRune(name, '/') {
				if codec, ok = Codecs[name]; !ok {
					warnDefaultCodecUnavailable = true
					continue
				}
			} else {
				var script ScriptCodec
				err := dbCodecs.FindId(name).One(&script)
				if err != nil {
					if err == mgo.ErrNotFound {
						warnDefaultCodecUnavailable = true
						continue
					}
					return "", nil, err
				}
				codec = &script
			}
			break
		} else {
			if codec = Codecs[name]; codec == nil {
				continue
			}
			break
		}
	}

	if codec == nil {
		var errStr = "The 'Content-Type' or 'Accept' header did not match any known codec."
		if warnNoDefaultCodec {
			errStr += "\nThe device has no codec set."
		}
		if warnDefaultCodecUnavailable {
			errStr += "\nThe device codec is unavailable or was deleted."
		}
		return "", nil, NewError(400, errStr)
	}

	return
}

// MarshalDevice writes complex data to the device.
// This might be JSON data, LoRaWAN XLPP payload or something else.
func MarshalDevice(deviceID string, headers http.Header, w io.Writer) (string, error) {

	name, codec, err := FindCodec(deviceID, headers.Get("Accept"))
	if err != nil {
		return "", err
	}
	return name, codec.MarshalDevice(deviceID, headers, w)
}

////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////

// SetDeviceID changes the gateway ID.
func SetDeviceID(newID string) error {

	currentGateway, err := GetDevice(LocalID())
	if err != nil {
		return err
	}

	newID = strings.ToLower(newID)

	if currentGateway.ID == newID {
		return nil // Nothing to change
	}

	if len(newID) != 16 { // 8 bytes accepted
		return fmt.Errorf("The length of the ID must be exactly 16 characters ( 8 bytes)")
	}

	_, err = hex.DecodeString(newID)
	if err != nil {
		return err
	}

	currentGateway.ID = newID
	currentGateway.Name = "(NEW) " + currentGateway.Name

	err = SetConfig("gatewayID", newID)
	if err != nil {
		return err
	}

	// Add the gateway as a new device and keep the old one as it has link to the old data.
	err = PostDevices(currentGateway)

	//...

	return err
}

/*------------------------------*/

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

	m := bson.M{
		"$set": set,
	}

	if len(unset) != 0 {
		m["$unset"] = unset
	}

	err := dbDevices.UpdateId(deviceID, m)

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
