package clouds

import (
	"strings"
	"time"
)

type v2SensorValue struct {
	Value        interface{} `json:"value"`
	Time         time.Time   `json:"timestamp"`
	TimeReceived time.Time   `json:"date_received"`
}

type v2ActuatorValue interface{}

type v2Sensor struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Value *v2SensorValue `json:"value,omitempty"`
}

type v2Actuator struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Value v2ActuatorValue `json:"value,omitempty"`
}

type v2Device struct {
	Name      string       `json:"name"`
	ID        string       `json:"id"`
	Sensors   []v2Sensor   `json:"sensors"`
	Actuators []v2Actuator `json:"actuators"`
	Gateway   string       `json:"gateway_id"`
}

type v2Gateway struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

func v2IdCompat(id string) string {
	return strings.ReplaceAll(id, " ", "_")
}
