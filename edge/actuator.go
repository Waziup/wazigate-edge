package edge

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Actuator represents a Waziup actuator
type Actuator struct {
	ID       string      `json:"id" bson:"id"`
	Name     string      `json:"name" bson:"name"`
	Modified time.Time   `json:"modified" bson:"modified"`
	Created  time.Time   `json:"created" bson:"created"`
	Time     time.Time   `json:"time" bson:"time"`
	Value    interface{} `json:"value" bson:"value"`
}

// GetActuatorValues returns an iterator over all actuator values.
func GetActuatorValues(deviceID string, actuatorID string, query *Query) *ValueIterator {

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

	return &ValueIterator{
		dbIter: q.Iter(),
	}
}

type aValue struct {
	ID         bson.ObjectId `json:"id" bson:"_id"`
	Value      interface{}   `json:"value" bson:"value"`
	DeviceID   string        `json:"deviceId" bson:"deviceId"`
	ActuatorID string        `json:"actuatorId" bson:"actuatorId"`
}

// PostActuatorValue stores a new actuator value for this actuator.
func PostActuatorValue(deviceID string, actuatorID string, val *Value) error {

	value := aValue{
		ID:         newID(val.Time),
		Value:      val.Value,
		DeviceID:   deviceID,
		ActuatorID: actuatorID,
	}

	err := dbDevices.Update(bson.M{
		"_id":          deviceID,
		"actuators.id": actuatorID,
	}, bson.M{
		"$set": bson.M{
			"actuators.$.value": val.Value,
			"actuators.$.time":  val.Time,
		},
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			return errNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	err = dbActuatorValues.Insert(&value)

	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}

// PostActuatorValues can be used to post multiple data point for this actuator.
func PostActuatorValues(deviceID string, actuatorID string, vals []Value) error {

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

	err := dbDevices.Update(bson.M{
		"_id":          deviceID,
		"actuators.id": actuatorID,
	}, bson.M{
		"$set": bson.M{
			"actuators.$.value": val.Value,
			"actuators.$.time":  val.Time,
		},
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			return errNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	err = dbActuatorValues.Insert(interf...)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}
