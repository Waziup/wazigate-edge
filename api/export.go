package api

//TODO: capsulate actuator and sensor in more generic function, check time correct (ISO)

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	routing "github.com/julienschmidt/httprouter"
)

// Only use host API calls for export
var Urls = []string{"http://localhost/" /*, "http://192.168.188.86/"*/}

// Meta holds entity metadata.
type Meta map[string]interface{}

// // For values of a sensor probe
// type Value struct {
// 	Value interface{} `json:"value" bson:"value"`
// 	Time  time.Time   `json:"time" bson:"time"`
// }

// // Sensor represents a Waziup sensor
// type Sensor struct {
// 	ID       string      `json:"id" bson:"id"`
// 	Name     string      `json:"name" bson:"name"`
// 	Modified time.Time   `json:"modified" bson:"modified"`
// 	Created  time.Time   `json:"created" bson:"created"`
// 	Time     time.Time   `json:"time" bson:"time"`
// 	Value    interface{} `json:"value" bson:"value"`
// 	Meta     Meta        `json:"meta" bson:"meta"`
// }

// // Actuator represents a Waziup actuator
// type Actuator struct {
// 	ID       string      `json:"id" bson:"id"`
// 	Name     string      `json:"name" bson:"name"`
// 	Modified time.Time   `json:"modified" bson:"modified"`
// 	Created  time.Time   `json:"created" bson:"created"`
// 	Time     time.Time   `json:"time" bson:"time"`
// 	Value    interface{} `json:"value" bson:"value"`
// 	Meta     Meta        `json:"meta" bson:"meta"`
// }

// // Device represents a Waziup Device
// type Device struct {
// 	Name      string      `json:"name" bson:"name"`
// 	ID        string      `json:"id" bson:"_id"`
// 	Sensors   []*Sensor   `json:"sensors" bson:"sensors"`
// 	Actuators []*Actuator `json:"actuators" bson:"actuators"`
// 	Modified  time.Time   `json:"modified" bson:"modified"`
// 	Created   time.Time   `json:"created" bson:"created"`
// 	Meta      Meta        `json:"meta" bson:"meta"`
// }

func execCurlCmd(url string, token string) []byte {
	cmd := exec.Command("curl", "--header", "Authorization: Bearer "+token, url)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error invoking curl cmd", err)
	}
	return output
}

func createCsv(path string) *csv.Writer {
	file, err := os.Create(filepath.Join(path))
	if err != nil {
		fmt.Println(err)
	}
	//defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer
}

func transpose(a [][]string) [][]string {
	l := 0
	for _, r := range a {
		if len(r) > l {
			l = len(r)
		}
	}

	b := make([][]string, l)
	for i := 0; i < l; i++ {
		b[i] = make([]string, len(a))
		for j := 0; j < len(a); j++ {
			if i < len(a[j]) {
				b[i][j] = a[j][i]
			}
		}
	}
	return b
}

func exportTree() {
	// Create static folders and files
	parentFolder := "exportTree"
	err := os.Mkdir(parentFolder, os.FileMode(0755))
	if err != nil {
		fmt.Println("Error creating folder: ", err)
	}
	deviceWriter := createCsv(filepath.Join(parentFolder, "devices.csv"))

	deviceRecord := make([][]string, 0)

	for _, url := range Urls {
		// Get token, important for non localhost devices
		cmd := exec.Command("curl", "-X", "POST", url+"auth/token", "-H", "accept: application/json", "-d", "{\"username\": \"admin\", \"password\": \"loragateway\"}")
		token, err := cmd.Output()
		if err != nil {
			fmt.Println("Error invoking curl cmd", err)
		}
		fmt.Println("Token: ", string(token))

		// Add devices to url for convenience
		url = url + "devices"

		// API call to Gateway
		outputCmd := execCurlCmd(url, string(token))

		// Create an empty map to hold the parsed JSON devices
		devices := make([]*Device, 0)

		// Write json response to map
		err = json.Unmarshal(outputCmd, &devices)
		if err != nil {
			fmt.Println("Error parsing JSON []byte:", err)
			return
		}

		// Iterate through map
		// Devices:
		for device := range devices {
			currentId := devices[device].ID
			fmt.Println("Device Id of current device: ", currentId)
			// Prepare device array to write to csv
			deviceSlice := make([]string, 5)
			deviceSlice[0] = currentId
			deviceSlice[1] = devices[device].Name
			deviceSlice[2] = devices[device].Created.String()
			deviceSlice[3] = devices[device].Modified.String()
			metaDeviceData, err := json.Marshal(devices[device].Meta)
			if err != nil {
				fmt.Println("Error marshal meta device data to JSON:", err)
				return
			}
			deviceSlice[4] = string(metaDeviceData)
			deviceRecord = append(deviceRecord[:device], deviceSlice)
			err = os.Mkdir(filepath.Join(parentFolder, currentId), os.FileMode(0755))
			if err != nil {
				fmt.Println("Error creating folder: ", err)
			}

			// Sensors
			// array to hold all sensors attached to one device
			sensorsRecord := make([][]string, 0)
			// Create CSV to hold values
			sensorsWriter := createCsv(filepath.Join(parentFolder, currentId, "sensors.csv"))

			for sensor := range devices[device].Sensors {
				currentSensorId := devices[device].Sensors[sensor].ID
				fmt.Println("Sensor Id of current Sensor: ", currentSensorId, "Parent device: ", currentId)

				// Sensors containing metadata
				// Create sensors.csv
				sensorsRecordSlice := make([]string, 7)
				sensorsRecordSlice[0] = currentSensorId
				sensorsRecordSlice[1] = devices[device].Sensors[sensor].Name
				sensorsRecordSlice[2] = devices[device].Sensors[sensor].Created.String()
				sensorsRecordSlice[3] = devices[device].Sensors[sensor].Modified.String()
				metaSensorsData, err := json.Marshal(devices[device].Sensors[sensor].Meta)
				if err != nil {
					fmt.Println("Error marshal meta sensor data to JSON:", err)
					return
				}
				sensorsRecordSlice[4] = string(metaSensorsData)
				sensorsRecord = append(sensorsRecord[:sensor], sensorsRecordSlice)

				// Values of probes
				// Folder for sensordata
				err = os.Mkdir(filepath.Join(parentFolder, currentId, currentSensorId), os.FileMode(0755))
				if err != nil {
					fmt.Println("Error creating folder: ", err)
				}

				// Create CSV to hold values
				sensorWriter := createCsv(filepath.Join(parentFolder, currentId, currentSensorId, "values.csv"))

				// Create sensor probe request
				requestUrl := url + "/" + currentId + "/sensors/" + currentSensorId + "/values"
				response := execCurlCmd(requestUrl, string(token))

				// Create an empty map to hold the parsed JSON devices
				values := make([]*Value, 0)

				// Write json response to map
				err = json.Unmarshal(response, &values)
				if err != nil {
					fmt.Println("Error parsing Value JSON []byte:", err)
					return
				}

				// Array to hold values and timestamps of one specific sensor probe
				sensorRecord := make([][]string, len(values))
				// Iterate over values map and create record
				for messurement := range values {
					sensorRecord[messurement] = make([]string, 2)
					sensorRecord[messurement][0] = values[messurement].Time.String()
					valueData, err := json.Marshal(values[messurement].Value)
					if err != nil {
						fmt.Println("Error marshal value data to JSON:", err)
						return
					}
					sensorRecord[messurement][1] = string(valueData)
				}

				// Write record of one sensor to value CSV
				err = sensorWriter.WriteAll(sensorRecord)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
			// Actuators
			// array to hold all actuators attached to one device
			actuatorsRecord := make([][]string, 0)
			// Create CSV to hold values
			actuatorsWriter := createCsv(filepath.Join(parentFolder, currentId, "actuators.csv"))

			for actuator := range devices[device].Actuators {
				currentActuatorId := devices[device].Actuators[actuator].ID
				fmt.Println("Actuator Id of current Actuator: ", currentActuatorId, "Parent device: ", currentId)

				// Actuators containing metadata
				// Create actuators.csv
				actuatorsRecordSlice := make([]string, 7)
				actuatorsRecordSlice[0] = currentActuatorId
				actuatorsRecordSlice[1] = devices[device].Actuators[actuator].Name
				actuatorsRecordSlice[2] = devices[device].Actuators[actuator].Created.String()
				actuatorsRecordSlice[3] = devices[device].Actuators[actuator].Modified.String()
				metaActuatorsData, err := json.Marshal(devices[device].Actuators[actuator].Meta)
				if err != nil {
					fmt.Println("Error marshal meta actuator data to JSON:", err)
					return
				}
				actuatorsRecordSlice[4] = string(metaActuatorsData)
				actuatorsRecord = append(actuatorsRecord[:actuator], actuatorsRecordSlice)

				err = os.Mkdir(filepath.Join(parentFolder, currentId, currentActuatorId), os.FileMode(0755))
				if err != nil {
					fmt.Println("Error creating folder: ", err)
				}

				// Values of probes
				// Folder for actuatordata
				err = os.Mkdir(filepath.Join(parentFolder, currentId, currentActuatorId), os.FileMode(0755))
				if err != nil {
					fmt.Println("Error creating folder: ", err)
				}

				// Create CSV to hold values
				actuatorWriter := createCsv(filepath.Join(parentFolder, currentId, currentActuatorId, "values.csv"))

				// Create actuator probe request
				requestUrl := url + "/" + currentId + "/actuators/" + currentActuatorId + "/values"
				response := execCurlCmd(requestUrl, string(token))

				// Create an empty map to hold the parsed JSON values
				values := make([]*Value, 0)

				// Write json response to map
				err = json.Unmarshal(response, &values)
				if err != nil {
					fmt.Println("Error parsing Value JSON []byte:", err)
					return
				}

				// Array to hold values and timestamps of one specific actuator probe
				actuatorRecord := make([][]string, len(values))
				// Iterate over values map and create record
				for messurement := range values {
					actuatorRecord[messurement] = make([]string, 2)
					actuatorRecord[messurement][0] = values[messurement].Time.String()
					valueData, err := json.Marshal(values[messurement].Value)
					if err != nil {
						fmt.Println("Error marshal value data to JSON:", err)
						return
					}
					actuatorRecord[messurement][1] = string(valueData)
				}

				// Write record of one actuator to value CSV
				err = actuatorWriter.WriteAll(actuatorRecord)
				if err != nil {
					fmt.Println(err)
					continue
				}

			}

			// Write the sensor/actuator data to sensors.csv/actuators.csv
			err = sensorsWriter.WriteAll(sensorsRecord)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = actuatorsWriter.WriteAll(actuatorsRecord)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}

		// Write to device data CSV
		err = deviceWriter.WriteAll(deviceRecord)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
	// Create ZIP file containing all the data
	cmd := exec.Command("zip", "-r", "exportTree.zip", "exportTree")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error invoking curl cmd", err)
	}
	fmt.Println("Creating zip file for csv export: \n", string(output))
}

// Exports all probes into one file
func exportAllInOne() ([][]string, error) {
	// Array to hold values and timestamps of all specific probes
	record := make([][]string, 0)

	for _, url := range Urls {
		// Get token, important for non localhost devices
		cmd := exec.Command("curl", "-X", "POST", url+"auth/token", "-H", "accept: application/json", "-d", "{\"username\": \"admin\", \"password\": \"loragateway\"}")
		token, err := cmd.Output()
		if err != nil {
			fmt.Println("Error invoking curl cmd", err)
		}
		fmt.Println("Token: ", string(token))

		// Add devices to url for convenience
		url += "devices"

		// API call to Gateway
		outputCmd := execCurlCmd(url, string(token))

		// Create an empty map to hold the parsed JSON devices
		devices := make([]*Device, 0)

		// Write json response to map
		err = json.Unmarshal(outputCmd, &devices)
		if err != nil {
			fmt.Println("Error parsing JSON []byte:", err)
			return nil, err
		}

		// Iterate through map:
		// Devices:
		for device := range devices {
			currentId := devices[device].ID
			// Sensors
			for sensor := range devices[device].Sensors {
				currentSensorId := devices[device].Sensors[sensor].ID
				fmt.Println("Sensor Id of current Sensor: ", currentSensorId, "Parent device: ", currentId)

				// Create sensor probe request
				requestUrl := url + "/" + currentId + "/sensors/" + currentSensorId + "/values"
				response := execCurlCmd(requestUrl, string(token))

				// Create an empty map to hold the parsed JSON devices
				values := make([]*Value, 0)

				// Write json response to map
				err = json.Unmarshal(response, &values)
				if err != nil {
					fmt.Println("Error parsing Value JSON []byte:", err)
					return nil, err
				}

				// Slices to hold values
				recordTimes := make([]string, len(values)+1)
				recordValues := make([]string, len(values)+1)

				// Add id and name on top
				recordTimes[0] = devices[device].ID + ", " + currentSensorId
				recordValues[0] = devices[device].Name + ", " + devices[device].Sensors[sensor].Name

				// Iterate over values map and create record
				for messurement := range values {
					recordTimes[messurement+1] = values[messurement].Time.String()
					valueData, err := json.Marshal(values[messurement].Value)
					if err != nil {
						fmt.Println("Error marshal value data to JSON:", err)
						return nil, err
					}
					recordValues[messurement+1] = string(valueData)

				}

				// Append times and values to arry
				record = append(record, recordTimes, recordValues)
			}

			// Actuators
			for actuator := range devices[device].Actuators {
				currentActuatorId := devices[device].Actuators[actuator].ID
				fmt.Println("Actuator Id of current Actuator: ", currentActuatorId, "Parent device: ", currentId)

				// Create actuator probe request
				requestUrl := url + "/" + currentId + "/actuators/" + currentActuatorId + "/values"
				response := execCurlCmd(requestUrl, string(token))

				// Create an empty map to hold the parsed JSON devices
				values := make([]*Value, 0)

				// Write json response to map
				err = json.Unmarshal(response, &values)
				if err != nil {
					fmt.Println("Error parsing Value JSON []byte:", err)
					return nil, err
				}

				// Slices to hold values
				recordTimes := make([]string, len(values)+1)
				recordValues := make([]string, len(values)+1)

				// Add id and name on top
				recordTimes[0] = devices[device].ID + ", " + currentActuatorId
				recordValues[0] = devices[device].Name + ", " + devices[device].Actuators[actuator].Name

				// Iterate over values map and create record
				for messurement := range values {
					recordTimes[messurement+1] = values[messurement].Time.String()
					valueData, err := json.Marshal(values[messurement].Value)
					if err != nil {
						fmt.Println("Error marshal value data to JSON:", err)
						return nil, err
					}
					recordValues[messurement+1] = string(valueData)

				}

				// Append times and values to arry
				record = append(record, recordTimes, recordValues)
			}

		}
	}
	// Transpose array
	tRecord := transpose(record)

	return tRecord, nil
}

func GetExportTree(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	// Function to create tree
	exportTree()

	// Read zip in []byte
	buf, err := os.ReadFile("exportTree.zip")

	if err != nil {
		serveError(resp, err)
		return
	}

	// Create response
	resp.Header().Set("Content-Type", "application/zip")
	resp.Write(buf)

	// Delete resources afterwards
	err = os.RemoveAll("./exportTree")
	if err != nil {
		serveError(resp, err)
		return
	}
	err = os.Remove("./exportTree.zip")
	if err != nil {
		serveError(resp, err)
		return
	}
}

func GetExportAllInOne(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	// Create array of all probes
	record, err := exportAllInOne()

	if err != nil {
		serveError(resp, err)
		return
	}

	// Write to CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	writer.WriteAll(record)

	// Create response
	resp.Header().Set("Content-Type", "text/csv")
	resp.Write(buf.Bytes())
}
