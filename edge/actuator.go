package edge

import (
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Actuator represents a Waziup actuator
type Actuator struct {
	ID   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`

	Modified time.Time `json:"modified" bson:"modified"`
	Created  time.Time `json:"created" bson:"created"`

	Time  *time.Time  `json:"time" bson:"time"`
	Value interface{} `json:"value" bson:"value"`

	Meta Meta `json:"meta" bson:"meta"`

	jsonSelect []string
}

func (actuator *Actuator) SetJSONSelect(s []string) {
	actuator.jsonSelect = s
}

func (actuator *Actuator) MarshalJSON() ([]byte, error) {
	clone := map[string]interface{}{}
	t := reflect.ValueOf(actuator).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.CanSet() {
			key := t.Type().Field(i).Tag.Get("json")
			if key != "id" {
				if actuator.jsonSelect != nil {
					selected := false
					for _, field := range actuator.jsonSelect {
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

// GetActuator returns the Waziup actuator.
func GetActuator(deviceID string, actuatorID string) (*Actuator, error) {

	var device Device
	err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"actuators": bson.M{
			"$elemMatch": bson.M{
				"id": actuatorID,
			},
		},
	}).One(&device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Actuators) == 0 {
		return nil, ErrNotFound
	}

	return device.Actuators[0], nil
}

// PostActuator creates a new actuator for this device.
func PostActuator(deviceID string, actuator *Actuator) error {

	if actuator.ID == "" {
		actuator.ID = bson.NewObjectId().Hex()
	}

	now := time.Now()
	actuator.Modified = now
	actuator.Created = now

	if actuator.Value == nil {
		actuator.Time = nil
	} else if actuator.Time == nil {
		actuator.Time = &now
	}

	var device Device
	err := dbDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"actuators": bson.M{
			"$elemMatch": bson.M{
				"id": actuator.ID,
			},
		},
	}).One(&device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Actuators) != 0 {
		return CodeError{409, "actuator already exists"}
	}

	err = dbDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$push": bson.M{
			"actuators": &actuator,
		},
	})
	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	if actuator.Value != nil {
		dbActuatorValues.Insert(&aValue{
			ID:         newID(*actuator.Time),
			DeviceID:   deviceID,
			ActuatorID: actuator.ID,
			Value:      actuator.Value,
		})
	}

	return nil
}

// SetActuatorName changes this actuators name.
func SetActuatorName(deviceID string, actuatorID string, name string) (Meta, error) {

	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id":          deviceID,
		"actuators.id": actuatorID,
	}).Select(
		bson.M{
			"actuators.id":   1,
			"actuators.meta": 1,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"actuators.$.modified": time.Now(),
				"actuators.$.name":     name,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Actuators) == 0 {
		return nil, mgo.ErrNotFound
	}

	for _, actuator := range device.Actuators {
		if actuator.ID == actuatorID {
			return actuator.Meta, nil
		}
	}

	return nil, nil
}

// SetActuatorMeta changes this actuators metadata.
func SetActuatorMeta(deviceID string, actuatorID string, meta map[string]interface{}) error {

	var unset = bson.M{}
	var set = bson.M{
		"actuators.$.modified": time.Now(),
	}
	for key, value := range meta {
		if value == nil {
			unset["actuators.$.meta."+key] = 1
		} else {
			set["actuators.$.meta."+key] = value
		}
	}

	var update = bson.M{
		"$set": set,
	}
	if len(unset) != 0 {
		update["$unset"] = unset
	}
	err := dbDevices.Update(bson.M{
		"_id":          deviceID,
		"actuators.id": actuatorID,
	}, update)

	if err != nil {
		if err == mgo.ErrNotFound {
			return ErrNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}

// DeleteActuator removes this actuator from the device and deletes all data points.
// This returns the number of data points deleted.
func DeleteActuator(deviceID string, actuatorID string) (int, error) {

	err1 := dbDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$pull": bson.M{
			"actuators": bson.M{
				"id": actuatorID,
			},
		},
	})
	info, err2 := dbActuatorValues.RemoveAll(bson.M{
		"deviceId":   deviceID,
		"actuatorId": actuatorID,
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

type aValueIterator struct {
	dbIter *mgo.Iter
}

func (iter aValueIterator) Next() (Value, error) {
	var sval aValue
	if iter.dbIter.Next(&sval) {
		val := Value{
			Value: sval.Value,
			Time:  sval.ID.Time(),
		}
		return val, iter.dbIter.Err()
	}
	return Value{}, io.EOF
}

func (iter aValueIterator) Close() error {
	return iter.dbIter.Close()
}

// GetActuatorValues returns an iterator over all actuator values.
func GetActuatorValues(deviceID string, actuatorID string, query *ValuesQuery) ValueIterator {

	// var value ActuatorValue

	m := bson.M{
		"deviceId":   deviceID,
		"actuatorId": actuatorID,
	}
	var noTime = time.Time{}
	if query.From != noTime || query.To != noTime {
		mid := bson.M{}
		m["_id"] = mid
		if query.From != noTime {
			mid["$gt"] = bson.NewObjectIdWithTime(query.From)
		}
		if query.To != noTime {
			query.To.Add(time.Second)
			mid["$lt"] = bson.NewObjectIdWithTime(query.To)
		}
	}
	q := dbActuatorValues.Find(m)
	if query.Limit != 0 {
		q.Limit(int(query.Limit))
	}

	return aValueIterator{q.Iter()}
}

type aValue struct {
	ID         bson.ObjectId `json:"id" bson:"_id"`
	Value      interface{}   `json:"value" bson:"value"`
	DeviceID   string        `json:"deviceId" bson:"deviceId"`
	ActuatorID string        `json:"actuatorId" bson:"actuatorId"`
}

// PostActuatorValue stores a new actuator value for this actuator.
func PostActuatorValue(deviceID string, actuatorID string, val Value) (Meta, error) {

	value := aValue{
		ID:         newID(val.Time),
		Value:      val.Value,
		DeviceID:   deviceID,
		ActuatorID: actuatorID,
	}
	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id":          deviceID,
		"actuators.id": actuatorID,
	}).Select(
		bson.M{
			"actuators.id": actuatorID,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"actuators.$.value": val.Value,
				"actuators.$.time":  val.Time,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Actuators) == 0 {
		return nil, ErrNotFound
	}

	err = dbActuatorValues.Insert(&value)

	if err != nil {
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	return device.Actuators[0].Meta, nil
}

// PostActuatorValues can be used to post multiple data point for this actuator.
func PostActuatorValues(deviceID string, actuatorID string, vals []Value) (Meta, error) {

	values := make([]aValue, len(vals))
	interf := make([]interface{}, len(vals))

	for i, v := range vals {
		values[i] = aValue{
			ID:         newID(v.Time),
			DeviceID:   deviceID,
			ActuatorID: actuatorID,
			Value:      v.Value,
		}
		interf[i] = values[i]
	}

	val := vals[len(vals)-1]

	var device Device
	_, err := dbDevices.Find(bson.M{
		"_id":          deviceID,
		"actuators.id": actuatorID,
	}).Select(
		bson.M{
			"actuators.id": actuatorID,
		},
	).Apply(mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"actuators.$.value": val.Value,
				"actuators.$.time":  val.Time,
			},
		},
	}, &device)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Actuators) == 0 {
		return nil, ErrNotFound
	}

	err = dbActuatorValues.Insert(interf...)
	if err != nil {
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	return device.Actuators[0].Meta, nil
}
