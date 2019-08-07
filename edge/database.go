package edge

import (
	"time"

	"github.com/globalsign/mgo"
)

// dbSensorValues is the database holding sensor values.
var dbSensorValues *mgo.Collection

// dbActuatorValues is the database holding actuator values.
var dbActuatorValues *mgo.Collection

// dbDevices is the database holding device informations
var dbDevices *mgo.Collection

// Connect initializes the edge core by connecting to the database.
func Connect(addr string) error {

	i := 0
	for true {
		db, err := mgo.Dial(addr)
		if err != nil {
			i++
			if i == 5 {
				return err
			}
			time.Sleep(time.Second * 2)
			continue
		}

		db.SetSafe(&mgo.Safe{})
		dbSensorValues = db.DB("waziup").C("sensor_values")
		dbActuatorValues = db.DB("waziup").C("actuator_values")
		dbDevices = db.DB("waziup").C("devices")
		return nil
	}
	return nil // unreachable
}
