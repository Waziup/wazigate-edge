package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/globalsign/mgo/bson"
)

const timeLayout = "2006-01-02T15:04:05-0700"

// SensorValue represents a Waziup sensor data value
type SensorValue struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Value    interface{}   `json:"value" bson:"value"`
	DeviceID string        `json:"deviceId" bson:"deviceId"`
	SensorID string        `json:"sensorId" bson:"sensorId"`
}

// SensorValue represents a Waziup actuator data value
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

func getReqValue(req *http.Request) (interface{}, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var value interface{}
	return value, json.Unmarshal(body, &value)
}

func getReqValues(req *http.Request) ([]interface{}, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var values []interface{}
	return values, json.Unmarshal(body, &values)
}

////////////////////

func (query *Query) from(req *http.Request) string {

	var param string
	var err error

	q := req.URL.Query()

	if param = q.Get("from"); param != "" {
		query.From, err = time.Parse(timeLayout, param)
		if err != nil {
			return "Query ?from=.. is mal formatted."
		}
	}

	if param = q.Get("to"); param != "" {
		query.To, err = time.Parse(timeLayout, param)
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
