package edge

import (
	"io"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Sensor represents a Waziup sensor
type Sensor struct {
	ID       string      `json:"id" bson:"id"`
	Name     string      `json:"name" bson:"name"`
	Modified time.Time   `json:"modified" bson:"modified"`
	Created  time.Time   `json:"created" bson:"created"`
	Time     time.Time   `json:"time" bson:"time"`
	Value    interface{} `json:"value" bson:"value"`
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
			return nil, errNotFound
		}
		return nil, CodeError{500, "database error: " + err.Error()}
	}

	if len(device.Sensors) == 0 {
		return nil, errNotFound
	}

	return device.Sensors[0], nil
}

// PostSensor creates a new sensor for this device.
func PostSensor(deviceID string, sensor *Sensor) error {

	err := dbDevices.Update(bson.M{
		"_id": deviceID,
	}, bson.M{
		"$push": bson.M{
			"sensors": &sensor,
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

// SetSensorName changes this sensors name.
func SetSensorName(deviceID string, sensorID string, name string) error {

	err := dbDevices.Update(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}, bson.M{
		"$set": bson.M{
			"sensors.$.modified": time.Now(),
			"sensors.$.name":     name,
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
			return 0, errNotFound
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
func GetSensorValues(deviceID string, sensorID string, query *Query) ValueIterator {

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
	q := dbSensorValues.Find(m)
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
func PostSensorValue(deviceID string, sensorID string, val Value) error {

	value := sValue{
		ID:       newID(val.Time),
		Value:    val.Value,
		DeviceID: deviceID,
		SensorID: sensorID,
	}

	err := dbDevices.Update(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}, bson.M{
		"$set": bson.M{
			"sensors.$.value": val.Value,
			"sensors.$.time":  val.Time,
		},
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			return errNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	err = dbSensorValues.Insert(&value)

	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}

// PostSensorValues can be used to post multiple data point for this sensor.
func PostSensorValues(deviceID string, sensorID string, vals []Value) error {

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

	err := dbDevices.Update(bson.M{
		"_id":        deviceID,
		"sensors.id": sensorID,
	}, bson.M{
		"$set": bson.M{
			"sensors.$.value": val.Value,
			"sensors.$.time":  val.Time,
		},
	})

	if err != nil {
		if err == mgo.ErrNotFound {
			return errNotFound
		}
		return CodeError{500, "database error: " + err.Error()}
	}

	err = dbSensorValues.Insert(interf...)
	if err != nil {
		return CodeError{500, "database error: " + err.Error()}
	}

	return nil
}
