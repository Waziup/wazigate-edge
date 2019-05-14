package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/globalsign/mgo/bson"
)

const TimeFormat = time.RFC3339 // "2006-01-02T15:04:05-0700"

// Value is one datapoint
type Value struct {
	Value interface{} `json:"value" bson:"value"`
	Time  time.Time   `json:"time" bson:"time"`
}

// SensorValue represents a Waziup sensor data value
type SensorValue struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Value    interface{}   `json:"value" bson:"value"`
	DeviceID string        `json:"deviceId" bson:"deviceId"`
	SensorID string        `json:"sensorId" bson:"sensorId"`
}

// ActuatorValue represents a Waziup actuator data value
type ActuatorValue struct {
	ID         bson.ObjectId `json:"id" bson:"_id"`
	Value      interface{}   `json:"value" bson:"value"`
	DeviceID   string        `json:"deviceId" bson:"deviceId"`
	ActuatorID string        `json:"actuatorId" bson:"actuatorId"`
}

// Query is used to range or limit query results
type Query struct {
	Limit int64
	From  time.Time
	To    time.Time
}

////////////////////

func getReqValue(req *http.Request) (Value, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return Value{}, err
	}
	val := Value{
		Value: nil,
		Time:  time.Now(),
	}
	err = json.Unmarshal(body, &val)
	if err != nil {
		err := json.Unmarshal(body, &val.Value)
		if err != nil {
			return val, err
		}
		return val, nil
	}
	return val, nil
}

func getReqValues(req *http.Request) ([]Value, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var values []Value
	err = json.Unmarshal(body, &values)
	if err != nil {
		var plains []interface{}
		err := json.Unmarshal(body, &values)
		if err != nil {
			return nil, err
		}
		values = make([]Value, len(plains))
		now := time.Now()
		for i, plain := range plains {
			values[i].Time = now
			values[i].Value = plain
		}
	}
	return values, nil
}

////////////////////

func (query *Query) from(req *http.Request) string {

	var param string
	var err error

	q := req.URL.Query()

	if param = q.Get("from"); param != "" {
		query.From, err = time.Parse(TimeFormat, param)
		if err != nil {
			return "Query ?from=.. is mal formatted."
		}
	}

	if param = q.Get("to"); param != "" {
		query.To, err = time.Parse(TimeFormat, param)
		if err != nil {
			return "Query ?to=.. is mal formatted."
		}
	}

	if param = q.Get("limit"); param != "" {
		query.Limit, err = strconv.ParseInt(param, 10, 64)
		if err != nil {
			return "Query ?limit=.. is mal formatted."
		}
	}

	return ""
}

////////////////////

func newID(t time.Time) bson.ObjectId {
	id := []byte(bson.NewObjectId())
	timeId := []byte(bson.NewObjectIdWithTime(t))
	copy(id[:4], timeId[:4])
	return bson.ObjectId(id)
}

////////////////////

// MarshalJSON provides a custom json serialization for values, transforming
//   {_id: ..., value: ...}
// to
//   {value: ..., time: ...}
func (v *SensorValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Value interface{} `json:"value"`
		Time  interface{} `json:"time"`
	}{
		Value: v.Value,
		Time:  v.ID.Time(),
	})
}

// ActuatorValue, see SensorValue
func (v *ActuatorValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Value interface{} `json:"value"`
		Time  interface{} `json:"time"`
	}{
		Value: v.Value,
		Time:  v.ID.Time(),
	})
}
