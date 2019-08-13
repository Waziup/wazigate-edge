package clouds

import "time"

type v2Value struct {
	Value        interface{} `json:"value"`
	Time         time.Time   `json:"timestamp"`
	TimeReceived time.Time   `json:"date_received"`
}

type v2Sensor struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Value *v2Value `json:"value,omitempty"`
}

type v2Actuator struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Value *v2Value `json:"value,omitempty"`
}

type v2Device struct {
	Name      string       `json:"name"`
	ID        string       `json:"id"`
	Sensors   []v2Sensor   `json:"sensors"`
	Actuators []v2Actuator `json:"actuators"`
}

type v2Gateway struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}
