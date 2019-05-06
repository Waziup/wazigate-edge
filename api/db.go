package api

import "github.com/globalsign/mgo"

// DBSensorValues is the database holding sensor values.
var DBSensorValues *mgo.Collection

// DBActuatorValues is the database holding actuator values.
var DBActuatorValues *mgo.Collection

// DBDevices is the database holding device informations
var DBDevices *mgo.Collection
