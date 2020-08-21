package edge

import (
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/Waziup/wazigate-edge/edge/ontology"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Sensor represents a Waziup sensor
type Sensor struct {
	ID   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`

	Modified time.Time `json:"modified" bson:"modified"`
	Created  time.Time `json:"created" bson:"created"`

	Kind     ontology.SensingKind `json:"kind" bson:"kind"`
	Quantity ontology.Quantity    `json:"quantity" bson:"quantity"`
	Unit     ontology.Unit        `json:"unit" bson:"unit"`

	Time  *time.Time  `json:"time" bson:"time"`
	Value interface{} `json:"value" bson:"value"`

	Meta Meta `json:"meta" bson:"meta"`

	jsonSelect []string
}

func (sensor *Sensor) SetJSONSelect(s []string) {
	sensor.jsonSelect = s
}

func (sensor *Sensor) MarshalJSON() ([]byte, error) {
	clone := map[string]interface{}{}
	t := reflect.ValueOf(sensor).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.CanSet() {
			key := t.Type().Field(i).Tag.Get("json")
			if key != "id" {
				if sensor.jsonSelect != nil {
					selected := false
					for _, field := range sensor.jsonSelect {
						if field == key || strings.HasPrefix(field, key+".") {
							selected = true
							break
						}
					}
					if !selected {
						continue
					}
				}
			}
			val := f.Interface()
			clone[key] = val
		}
	}
	return json.Marshal(clone)
}

// GetSensor returns the Waziup sensor.
func GetSensor(deviceID string, sensorID string) (*Sensor, error) {

	var device Device
	err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"sensors": bson.M{
			"$elemMatch": bson.M{
				"id": sensorID,
			},
		},
	}).One(&device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) == 0 {
		return nil, ErrNotFound
	}

	return device.Sensors[0], nil
}

// PostSensor creates a new sensor for this device.
func PostSensor(deviceID string, sensor *Sensor) error {

	if sensor.ID == "" {
		sensor.ID = bson.NewObjectId().Hex()
	}

	now := time.Now()
	sensor.Modified = now
	sensor.Created = now

	if sensor.Value == nil {
		sensor.Time = nil
	} else if sensor.Time == nil {
		sensor.Time = &now
	}

	var device Device
	err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"sensors": bson.M{
			"$elemMatch": bson.M{
				"id": sensor.ID,
			},
		},
	}).One(&device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) != 0 {
		return CodeError{409, "sensor already exists"}
	}

	err = dbDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$push": bson.M{
			"sensors": &sensor,
		},
	})
	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	if sensor.Value != nil {
		dbSensorValues.Insert(&sValue{
			ID:       newID(*sensor.Time),
			DeviceID: deviceID,
			SensorID: sensor.ID,
			Value:    sensor.Value,
		})
	}

	return nil
}

// SetSensorName changes this sensors name.
func SetSensorName(deviceID string, sensorID string, name string) (Meta, error) {

	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}).Select(
		bson.M{
			"sensors.id": sensorID,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"sensors.$.modified": time.Now(),
				"sensors.$.name":     name,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) == 0 {
		return nil, mgo.ErrNotFound
	}

	return device.Sensors[0].Meta, nil
}

// SetSensorMeta changes this sensors metadata.
func SetSensorMeta(deviceID string, sensorID string, meta Meta) error {

	var unset = bson.M{}
	var set = bson.M{
		"sensors.$.modified": time.Now(),
	}
	for key, value := range meta {
		if value == nil {
			unset["sensors.$.meta."+key] = 1
		} else {
			set["sensors.$.meta."+key] = value
		}
	}

	var update = bson.M{
		"$set": set,
	}
	if len(unset) != 0 {
		update["$unset"] = unset
	}
	err := dbDevices.Update(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}, update)

	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}

// DeleteSensor removes this sensor from the device and deletes all data points.
// This returns the number of data points deleted.
func DeleteSensor(deviceID string, sensorID string) (int, error) {

	err1 := dbDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$pull": bson.M{
			"sensors": bson.M{
				"id": sensorID,
			},
		},
	})
	info, err2 := dbSensorValues.RemoveAll(bson.M{
		"deviceId": deviceID,
		"sensorId": sensorID,
	})

	if err1 != nil || err2 != nil {
		err := err1
		if err == nil {
			err = err2
		}
		if err == mgo.ErrNotFound {
			return 0, ErrNotFound
		}
		return 0, CodeError{500, "database error: " + err.Error()}
	}

	return info.Removed, nil
}

////////////////////

type sValueIterator struct {
	dbIter *mgo.Iter
}

func (iter sValueIterator) Next() (Value, error) {
	var sval sValue
	if iter.dbIter.Next(&sval) {
		val := Value{
			Value: sval.Value,
			Time:  sval.ID.Time(),
		}
		return val, iter.dbIter.Err()
	}
	return Value{}, io.EOF
}

func (iter sValueIterator) Close() error {
	return iter.dbIter.Close()
}

// GetSensorValues returns an iterator over all sensor values.
func GetSensorValues(deviceID string, sensorID string, query *ValuesQuery) ValueIterator {

	// var value SensorValue

	m := bson.M{
		"deviceId": deviceID,
		"sensorId": sensorID,
	}
	var noTime = time.Time{}
	if query.From != noTime || query.To != noTime {
		mid := bson.M{}
		m["_id"] = mid
		if query.From != noTime {
			mid["$gte"] = bson.NewObjectIdWithTime(query.From)
		}
		if query.To != noTime {
			query.To.Add(time.Second)
			mid["$lt"] = bson.NewObjectIdWithTime(query.To)
		}
	}
	q := dbSensorValues.Find(m).Sort("_id")
	if query.Limit != 0 {
		q.Limit(int(query.Limit))
	}

	return sValueIterator{q.Iter()}
}

type sValue struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Value    interface{}   `json:"value" bson:"value"`
	DeviceID string        `json:"deviceId" bson:"deviceId"`
	SensorID string        `json:"sensorId" bson:"sensorId"`
}

// PostSensorValue stores a new sensor value for this sensor.
func PostSensorValue(deviceID string, sensorID string, val Value) (Meta, error) {

	value := sValue{
		ID:       newID(val.Time),
		Value:    val.Value,
		DeviceID: deviceID,
		SensorID: sensorID,
	}

	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}).Select(
		bson.M{
			"sensors.id":   1,
			"sensors.meta": 1,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"sensors.$.value": val.Value,
				"sensors.$.time":  val.Time,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) == 0 {
		return nil, ErrNotFound
	}

	err = dbSensorValues.Insert(&value)

	if err != nil {
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	for _, sensor := range device.Sensors {
		if sensor.ID == sensorID {
			return sensor.Meta, nil
		}
	}

	return nil, nil
}

// PostSensorValues can be used to post multiple data point for this sensor.
func PostSensorValues(deviceID string, sensorID string, vals []Value) (Meta, error) {

	values := make([]sValue, len(vals))
	interf := make([]interface{}, len(vals))

	for i, v := range vals {
		values[i] = sValue{
			ID:       newID(v.Time),
			DeviceID: deviceID,
			SensorID: sensorID,
			Value:    v.Value,
		}
		interf[i] = values[i]
	}

	val := vals[len(vals)-1]

	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}).Select(
		bson.M{
			"sensors.id": sensorID,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"sensors.$.value": val.Value,
				"sensors.$.time":  val.Time,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) == 0 {
		return nil, ErrNotFound
	}

	err = dbSensorValues.Insert(interf...)
	if err != nil {
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	return device.Sensors[0].Meta, nil
}
