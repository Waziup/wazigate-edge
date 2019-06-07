package main

import (
	"github.com/Waziup/wazigateway-edge/api"
	routing "github.com/julienschmidt/httprouter"
)

var router = routing.New()

func init() {

	api.Downstream = mqttServer

	// Device Endpoints

	router.GET("/devices", api.GetDevices)
	router.GET("/devices/:device_id", api.GetDevice)
	router.POST("/devices", api.PostDevice)
	router.DELETE("/devices/:device_id", api.DeleteDevice)
	router.POST("/devices/:device_id/name", api.PostDeviceName)

	// Sensor Endpoints

	router.GET("/devices/:device_id/sensors", api.GetDeviceSensors)
	router.GET("/devices/:device_id/sensors/:sensor_id", api.GetDeviceSensor)

	router.POST("/devices/:device_id/sensors", api.PostDeviceSensor)
	router.DELETE("/devices/:device_id/sensors/:sensor_id", api.DeleteDeviceSensor)
	router.POST("/devices/:device_id/sensors/:sensor_id/name", api.PostDeviceSensorName)

	router.GET("/devices/:device_id/sensors/:sensor_id/value", api.GetDeviceSensorValue)
	router.GET("/devices/:device_id/sensors/:sensor_id/values", api.GetDeviceSensorValues)

	router.POST("/devices/:device_id/sensors/:sensor_id/value", api.PostDeviceSensorValue)
	router.POST("/devices/:device_id/sensors/:sensor_id/values", api.PostDeviceSensorValues)

	// Actuator Endpoints

	router.GET("/devices/:device_id/actuators", api.GetDeviceActuators)
	router.GET("/devices/:device_id/actuators/:actuator_id", api.GetDeviceActuator)

	router.POST("/devices/:device_id/actuators", api.PostDeviceActuator)
	router.DELETE("/devices/:device_id/actuators/:actuator_id", api.DeleteDeviceActuator)
	router.POST("/devices/:device_id/actuators/:actuator_id/name", api.PostDeviceActuatorName)

	router.GET("/devices/:device_id/actuators/:actuator_id/value", api.GetDeviceActuatorValue)
	router.GET("/devices/:device_id/actuators/:actuator_id/values", api.GetDeviceActuatorValues)

	router.POST("/devices/:device_id/actuators/:actuator_id/value", api.PostDeviceActuatorValue)
	router.POST("/devices/:device_id/actuators/:actuator_id/values", api.PostDeviceActuatorValues)

	// Shortcut Endpoints (equals device_id = current device ID)

	router.GET("/device", api.GetCurrentDevice)
	router.POST("/device/name", api.PostCurrentDeviceName)

	router.GET("/sensors", api.GetSensors)
	router.GET("/sensors/:sensor_id", api.GetSensor)
	router.DELETE("/sensors/:sensor_id", api.DeleteSensor)
	router.GET("/sensors/:sensor_id/value", api.GetSensorValue)
	router.GET("/sensors/:sensor_id/values", api.GetSensorValues)

	router.GET("/actuators", api.GetActuators)
	router.GET("/actuators/:actuator_id", api.GetActuator)
	router.DELETE("/actuators/:actuator_id", api.DeleteActuator)
	router.GET("/actuators/:actuator_id/value", api.GetActuatorValue)
	router.GET("/actuators/:actuator_id/values", api.GetActuatorValues)

	router.POST("/sensors", api.PostSensor)

	router.POST("/sensors/:sensor_id/value", api.PostSensorValue)
	router.POST("/sensors/:sensor_id/values", api.PostSensorValues)

	router.POST("/actuators/:actuator_id/value", api.PostSensorValue)
	router.POST("/actuators/:actuator_id/values", api.PostSensorValues)

	// Clouds configuration

	router.GET("/clouds", api.GetClouds)
	router.POST("/clouds", api.PostClouds)
	router.GET("/clouds/:cloud_id", api.GetCloud)
	router.DELETE("/clouds/:cloud_id", api.DeleteCloud)
	router.POST("/clouds/:cloud_id/paused", api.PostCloudPaused)
	router.POST("/clouds/:cloud_id/credentials", api.PostCloudCredentials)
	router.POST("/clouds/:cloud_id/url", api.PostCloudURL)

}