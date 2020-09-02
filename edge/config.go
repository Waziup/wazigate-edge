package edge

import (
	// "io"
	// "time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Config represents a Wazigate edge Config
type Config struct {
	Key   string `json:"key" bson:"key"`
	Value string `json:"value" bson:"value"`
}

/*--------------------------------*/

// GetConfig returns the Wazigate user
func GetConfig(key string) (string, error) {

	var config Config
	err := dbConfig.Find(bson.M{
		"key": key,
	}).One(&config)

	return config.Value, err
}

/*--------------------------------*/

// SetConfig saved a new value for a config key and creates if it does not exist
func SetConfig(key string, value string) error {

	var config Config

	_, err := dbConfig.Find(bson.M{
		"key": key,
	}).Apply(mgo.Change{

		Update: bson.M{
			"$set": bson.M{
				"value": value,
			},
		},
	}, &config)

	if err == mgo.ErrNotFound {
		err = dbConfig.Insert(&Config{
			Key:   key,
			Value: value,
		})
	}

	return err
}

/*--------------------------------*/
