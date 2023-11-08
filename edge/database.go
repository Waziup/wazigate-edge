package edge

import (
	"embed"
	"log"
	"strings"
	"time"

	"github.com/globalsign/mgo"
)

// dbSensorValues is the database holding sensor values.
var dbSensorValues *mgo.Collection

// dbActuatorValues is the database holding actuator values.
var dbActuatorValues *mgo.Collection

// dbDevices is the database holding devices' information
var dbDevices *mgo.Collection

// dbCodecs is the database holding codecs & scripts
var dbCodecs *mgo.Collection

// dbMessages is the database holding wazigate messages
var dbMessages *mgo.Collection

// dbUsers is the database holding users' information
var dbUsers *mgo.Collection

// dbConfig is the database holding the configurations in a key-value form
var dbConfig *mgo.Collection

// ConnectWithInfo initializes the edge core by connecting to the database.
func ConnectWithInfo(info *mgo.DialInfo) error {
	i := 0
	for true {
		db, err := mgo.DialWithInfo(info)
		if err != nil {
			i++
			if i == 100 {
				return err
			}
			time.Sleep(time.Second * 2)
			continue
		}

		db.SetSafe(&mgo.Safe{})
		dbSensorValues = db.DB("waziup").C("sensor_values")
		dbActuatorValues = db.DB("waziup").C("actuator_values")
		dbDevices = db.DB("waziup").C("devices")
		dbMessages = db.DB("waziup").C("messages")
		dbCodecs = db.DB("waziup").C("codecs")
		dbUsers = db.DB("waziup").C("users")
		dbConfig = db.DB("waziup").C("config")

		err = CheckCustomJSCodecsAvailable()
		if err != nil {
			return err
		}

		return nil
	}
	return nil // unreachable
}

// Go embed loads the files at the COMPILE TIME and puts them into the binary exectable.
//
//go:embed codecs/custom/*.js
var codecs embed.FS

// TODO: use proper logging and error handling
func CheckCustomJSCodecsAvailable() error {
	count, err := dbCodecs.Count()
	if err != nil {
		return err
	}

	if count == 0 {
		files, err := codecs.ReadDir("codecs/custom")
		if err != nil {
			return err
		}

		for _, file := range files {
			data, err := codecs.ReadFile("codecs/custom/" + file.Name())
			if err != nil {
				return err
			}

			// Extract name without file extension
			codecName := strings.TrimSuffix(file.Name(), ".js")

			// Create codec instance
			codec := ScriptCodec{
				Name:      codecName,
				Mime:      "application/javascript",
				ServeMime: "application/waziup." + strings.ReplaceAll(codecName, " ", ""),
				Script:    string(data),
			}

			// Post codec
			err = PostCodec(&codec)
			if err != nil {
				return err
			}

			count++
		}
	}

	log.Printf("[     ] There are %d custom JavaScript codecs installed.", count)

	return nil
}

func Connect(addr string) error {
	info, err := mgo.ParseURL(addr)
	if err != nil {
		return err
	}
	info.Timeout = 10 * time.Second
	return ConnectWithInfo(info)
}
