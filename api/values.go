package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"
	"github.com/globalsign/mgo/bson"
)

var noTime = time.Time{}

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
	Size  int64
}

////////////////////

func getReqValue(req *http.Request) (edge.Value, error) {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		return edge.Value{}, err
	}
	val := edge.Value{
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

func getReqValues(req *http.Request) ([]edge.Value, error) {
	body, err := tools.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var values []edge.Value
	err = json.Unmarshal(body, &values)
	if err != nil {
		var plains []interface{}
		err := json.Unmarshal(body, &values)
		if err != nil {
			return nil, err
		}
		values = make([]edge.Value, len(plains))
		now := time.Now()
		for i, plain := range plains {
			values[i].Time = now
			values[i].Value = plain
		}
	} else {
		now := time.Now()
		var noTime time.Time
		for i, val := range values {
			if val.Time == noTime {
				values[i].Time = now
			}
		}
	}
	return values, nil
}

////////////////////

var sizeRegex = regexp.MustCompile(`^\d+[kKmMgG]?[bB]?`)
var sizeUnitRegex = regexp.MustCompile(`[kKmMgG]?[bB]?$`)

func (query *Query) from(req *http.Request) string {

	var param string
	var err error

	q := req.URL.Query()

	if param = q.Get("from"); param != "" {
		err = query.From.UnmarshalText([]byte(param))
		if err != nil {
			return "Query ?from=.. is mal formatted."
		}
	}

	if param = q.Get("to"); param != "" {
		err = query.To.UnmarshalText([]byte(param))
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

	if param = q.Get("size"); param != "" {
		query.Size = parseSize(param)
		if query.Size == -1 {
			return "Query ?size=.. is mal formatted."
		}
	}

	return ""
}

func parseSize(str string) (size int64) {
	for len(str) != 0 {
		match := sizeRegex.FindString(str)
		if match == "" {
			return -1
		}
		unit := sizeUnitRegex.FindString(str)
		var fact int64 = 1
		if len(unit) > 0 {
			if unit[0] == 'k' || unit[0] == 'K' {
				fact = 1e3
			} else if unit[0] == 'm' || unit[0] == 'M' {
				fact = 1e6
			} else if unit[0] == 'g' || unit[0] == 'G' {
				fact = 1e9
			}
		}
		n, _ := strconv.ParseInt(match[0:len(match)-len(unit)], 10, 64)
		size += n * fact
		str = str[len(match):]
	}
	return
}

////////////////////

func newID(t time.Time) bson.ObjectId {
	id := []byte(bson.NewObjectId())
	timeID := []byte(bson.NewObjectIdWithTime(t))
	copy(id[:4], timeID[:4])
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
