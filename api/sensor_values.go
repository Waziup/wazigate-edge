package api

import (
	"encoding/json"
	"net/http"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	routing "github.com/julienschmidt/httprouter"
)

// GetDeviceSensorValue implements GET /devices/{deviceID}/sensors/{sensorID}/value
func GetDeviceSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getLastSensorValue(resp, params.ByName("device_id"), params.ByName("sensor_id"))
}

// GetSensorValue implements GET /sensors/{sensorID}/value
func GetSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	getLastSensorValue(resp, GetLocalID(), params.ByName("sensor_id"))
}

// GetDeviceSensorValues implements GET /devices/{deviceID}/sensors/{sensorID}/values
func GetDeviceSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query Query
	if err := query.from(req); err != "" {
		http.Error(resp, "Bad Request - "+err, http.StatusBadRequest)
		return
	}
	getSensorValues(resp, params.ByName("device_id"), params.ByName("sensor_id"), &query)
}

// GetSensorValues implements GET /sensors/{sensorID}/values
func GetSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query Query
	if err := query.from(req); err != "" {
		http.Error(resp, "Bad Request - "+err, http.StatusBadRequest)
		return
	}
	getSensorValues(resp, GetLocalID(), params.ByName("sensor_id"), &query)
}

// PostDeviceSensorValue implements POST /devices/{deviceID}/sensors/{sensorID}/value
func PostDeviceSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValue(resp, req, params.ByName("device_id"), params.ByName("sensor_id"))
}

// PostDeviceSensorValues implements POST /devices/{deviceID}/sensors/{sensorID}/values
func PostDeviceSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValues(resp, req, params.ByName("device_id"), params.ByName("sensor_id"))
}

// PostSensorValue implements POST /sensors/{sensorID}/value
func PostSensorValue(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValue(resp, req, GetLocalID(), params.ByName("sensor_id"))
}

// PostSensorValues implements POST /sensors/{sensorID}/values
func PostSensorValues(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	postSensorValues(resp, req, GetLocalID(), params.ByName("sensor_id"))
}

////////////////////

func getLastSensorValue(resp http.ResponseWriter, deviceID string, sensorID string) {

	if DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var device Device
	err := DBDevices.Find(bson.M{
		"_id": deviceID,
	}).Select(bson.M{
		"sensors": bson.M{
			"$elemMatch": bson.M{
				"id": sensorID,
			},
		},
	}).One(&device)

	if err != nil || len(device.Sensors) == 0 {
		http.Error(resp, "null", http.StatusNotFound)
		return
	}

	value := device.Sensors[0].Value
	data, _ := json.Marshal(value)
	resp.Write(data)
}

func getSensorValues(resp http.ResponseWriter, deviceID string, sensorID string, query *Query) {

	if DBSensorValues == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	var value SensorValue
	iter := DBSensorValues.Find(bson.M{"deviceId": deviceID, "sensorId": sensorID}).Iter()
	serveIter(resp, iter, &value)
}

////////////////////

func postSensorValue(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	val, err := getReqValue(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBSensorValues == nil || DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	value := SensorValue{
		ID:       newID(val.Time),
		Value:    val.Value,
		DeviceID: deviceID,
		SensorID: sensorID,
	}

	err = DBDevices.Update(bson.M{
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
			http.Error(resp, "Device or sensor not found.", http.StatusNotFound)
			return
		}
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = DBSensorValues.Insert(&value)

	if err != nil {
		http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Write([]byte{'"'})
	resp.Write([]byte(value.ID.Hex()))
	resp.Write([]byte{'"'})
}

func postSensorValues(resp http.ResponseWriter, req *http.Request, deviceID string, sensorID string) {

	vals, err := getReqValues(req)
	if err != nil {
		http.Error(resp, "Bad Request - "+err.Error(), http.StatusBadRequest)
		return
	}

	if DBSensorValues == nil || DBDevices == nil {
		http.Error(resp, "Database unavailable.", http.StatusServiceUnavailable)
		return
	}

	if len(vals) != 0 {
		values := make([]SensorValue, len(vals))
		interf := make([]interface{}, len(vals))

		for i, v := range vals {
			values[i] = SensorValue{
				ID:       newID(v.Time),
				DeviceID: deviceID,
				SensorID: sensorID,
				Value:    v.Value,
			}
			interf[i] = values[i]
		}

		val := vals[len(vals)-1]

		err := DBDevices.Update(bson.M{
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
				http.Error(resp, "Device or sensor not found.", http.StatusNotFound)
				return
			}
			http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = DBSensorValues.Insert(interf...)
		if err != nil {
			http.Error(resp, "Database Error - "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp.Write([]byte("true"))
}
