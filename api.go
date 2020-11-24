package main

import (
	"github.com/Waziup/wazigate-edge/api"
	routing "github.com/julienschmidt/httprouter"
)

var router = routing.New()

func init() {

	// api.Downstream = mqttServer

	// Auth

	router.POST("/auth/token", api.GetToken)
	router.POST("/auth/retoken", api.IsAuthorized(api.GetRefereshToken, true /* true: check for IP based white list*/))
	router.GET("/auth/logout", api.Logout)
	router.POST("/auth/logout", api.Logout)
	router.GET("/auth/permissions", api.GetPermissions)

	router.GET("/auth/profile", api.IsAuthorized(api.GetUserProfile, true /* true: check for IP based white list*/))
	router.POST("/auth/profile", api.IsAuthorized(api.PostUserProfile, true /* true: check for IP based white list*/))

	//Apps

	router.GET("/apps", api.IsAuthorized(api.GetApps, true /* true: check for IP based white list*/))
	router.GET("/apps/:app_id", api.IsAuthorized(api.GetApp, true /* true: check for IP based white list*/))

	router.POST("/apps/:app_id", api.IsAuthorized(api.PostApp, true /* true: check for IP based white list*/))
	router.DELETE("/apps/:app_id", api.IsAuthorized(api.DeleteApp, true /* true: check for IP based white list*/))

	router.POST("/apps", api.IsAuthorized(api.PostApps, true /* true: check for IP based white list*/)) // install a new app

	router.GET("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true /* true: check for IP based white list*/))
	router.POST("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true /* true: check for IP based white list*/))
	router.PUT("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true /* true: check for IP based white list*/))
	router.DELETE("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true /* true: check for IP based white list*/))

	//Update Appps

	// router.GET("/update", api.GetUpdateApps, false /* true: check for IP based white list*/))
	router.GET("/update/:app_id", api.IsAuthorized(api.GetUpdateApp, true /* true: check for IP based white list*/))

	// router.POST("/update", api.PostUpdateApps, false /* true: check for IP based white list*/))
	router.POST("/update/:app_id", api.IsAuthorized(api.PostUpdateApp, true /* true: check for IP based white list*/))

	// Device Endpoints

	router.GET("/devices", api.IsAuthorized(api.GetDevices, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id", api.IsAuthorized(api.GetDevice, true /* true: check for IP based white list*/))
	router.POST("/devices", api.IsAuthorized(api.PostDevice, true /* true: check for IP based white list*/))
	router.DELETE("/devices/:device_id", api.IsAuthorized(api.DeleteDevice, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id/name", api.IsAuthorized(api.GetDeviceName, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/name", api.IsAuthorized(api.PostDeviceName, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id/meta", api.IsAuthorized(api.GetDeviceMeta, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/meta", api.IsAuthorized(api.PostDeviceMeta, true /* true: check for IP based white list*/))

	// Sensor Endpoints

	router.GET("/devices/:device_id/sensors", api.IsAuthorized(api.GetDeviceSensors, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id/sensors/:sensor_id", api.IsAuthorized(api.GetDeviceSensor, true /* true: check for IP based white list*/))

	router.POST("/devices/:device_id/sensors", api.IsAuthorized(api.PostDeviceSensor, true /* true: check for IP based white list*/))
	router.DELETE("/devices/:device_id/sensors/:sensor_id", api.IsAuthorized(api.DeleteDeviceSensor, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/sensors/:sensor_id/name", api.IsAuthorized(api.PostDeviceSensorName, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/sensors/:sensor_id/meta", api.IsAuthorized(api.PostDeviceSensorMeta, true /* true: check for IP based white list*/))

	router.GET("/devices/:device_id/sensors/:sensor_id/value", api.IsAuthorized(api.GetDeviceSensorValue, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id/sensors/:sensor_id/values", api.IsAuthorized(api.GetDeviceSensorValues, true /* true: check for IP based white list*/))

	router.POST("/devices/:device_id/sensors/:sensor_id/value", api.IsAuthorized(api.PostDeviceSensorValue, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/sensors/:sensor_id/values", api.IsAuthorized(api.PostDeviceSensorValues, true /* true: check for IP based white list*/))

	// Actuator Endpoints

	router.GET("/devices/:device_id/actuators", api.IsAuthorized(api.GetDeviceActuators, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id/actuators/:actuator_id", api.IsAuthorized(api.GetDeviceActuator, true /* true: check for IP based white list*/))

	router.POST("/devices/:device_id/actuators", api.IsAuthorized(api.PostDeviceActuator, true /* true: check for IP based white list*/))
	router.DELETE("/devices/:device_id/actuators/:actuator_id", api.IsAuthorized(api.DeleteDeviceActuator, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/actuators/:actuator_id/name", api.IsAuthorized(api.PostDeviceActuatorName, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/actuators/:actuator_id/meta", api.IsAuthorized(api.PostDeviceActuatorMeta, true /* true: check for IP based white list*/))

	router.GET("/devices/:device_id/actuators/:actuator_id/value", api.IsAuthorized(api.GetDeviceActuatorValue, true /* true: check for IP based white list*/))
	router.GET("/devices/:device_id/actuators/:actuator_id/values", api.IsAuthorized(api.GetDeviceActuatorValues, true /* true: check for IP based white list*/))

	router.POST("/devices/:device_id/actuators/:actuator_id/value", api.IsAuthorized(api.PostDeviceActuatorValue, true /* true: check for IP based white list*/))
	router.POST("/devices/:device_id/actuators/:actuator_id/values", api.IsAuthorized(api.PostDeviceActuatorValues, true /* true: check for IP based white list*/))

	// Shortcut Endpoints (equals device_id = current device ID, true /* true: check for IP based white list*/))

	router.GET("/device", api.IsAuthorized(api.GetCurrentDevice, true /* true: check for IP based white list*/))
	router.GET("/device/id", api.GetCurrentDeviceID)
	router.GET("/device/name", api.IsAuthorized(api.GetCurrentDeviceName, true /* true: check for IP based white list*/))
	router.GET("/device/meta", api.IsAuthorized(api.GetCurrentDeviceMeta, true /* true: check for IP based white list*/))
	router.POST("/device/id", api.IsAuthorized(api.PostCurrentDeviceID, false /* true: check for IP based white list*/))
	router.POST("/device/name", api.IsAuthorized(api.PostCurrentDeviceName, false /* true: check for IP based white list*/))
	router.POST("/device/meta", api.IsAuthorized(api.PostCurrentDeviceMeta, false /* true: check for IP based white list*/))

	router.GET("/sensors", api.IsAuthorized(api.GetSensors, true /* true: check for IP based white list*/))
	router.POST("/sensors", api.IsAuthorized(api.PostSensor, true /* true: check for IP based white list*/))
	router.GET("/sensors/:sensor_id", api.IsAuthorized(api.GetSensor, true /* true: check for IP based white list*/))
	router.DELETE("/sensors/:sensor_id", api.IsAuthorized(api.DeleteSensor, true /* true: check for IP based white list*/))
	router.GET("/sensors/:sensor_id/value", api.IsAuthorized(api.GetSensorValue, true /* true: check for IP based white list*/))
	router.GET("/sensors/:sensor_id/values", api.IsAuthorized(api.GetSensorValues, true /* true: check for IP based white list*/))
	router.POST("/sensors/:sensor_id/name", api.IsAuthorized(api.PostSensorName, true /* true: check for IP based white list*/))
	router.POST("/sensors/:sensor_id/meta", api.IsAuthorized(api.PostSensorMeta, true /* true: check for IP based white list*/))

	router.GET("/actuators", api.IsAuthorized(api.GetActuators, true /* true: check for IP based white list*/))
	router.POST("/actuators", api.IsAuthorized(api.PostActuator, true /* true: check for IP based white list*/))
	router.GET("/actuators/:actuator_id", api.IsAuthorized(api.GetActuator, true /* true: check for IP based white list*/))
	router.DELETE("/actuators/:actuator_id", api.IsAuthorized(api.DeleteActuator, true /* true: check for IP based white list*/))
	router.GET("/actuators/:actuator_id/value", api.IsAuthorized(api.GetActuatorValue, true /* true: check for IP based white list*/))
	router.GET("/actuators/:actuator_id/values", api.IsAuthorized(api.GetActuatorValues, true /* true: check for IP based white list*/))
	router.POST("/actuators/:actuator_id/name", api.IsAuthorized(api.PostActuatorName, true /* true: check for IP based white list*/))
	router.POST("/actuators/:actuator_id/meta", api.IsAuthorized(api.PostActuatorMeta, true /* true: check for IP based white list*/))

	router.POST("/sensors/:sensor_id/value", api.IsAuthorized(api.PostSensorValue, true /* true: check for IP based white list*/))
	router.POST("/sensors/:sensor_id/values", api.IsAuthorized(api.PostSensorValues, true /* true: check for IP based white list*/))

	router.POST("/actuators/:actuator_id/value", api.IsAuthorized(api.PostSensorValue, true /* true: check for IP based white list*/))
	router.POST("/actuators/:actuator_id/values", api.IsAuthorized(api.PostSensorValues, true /* true: check for IP based white list*/))

	// Clouds configuration

	router.GET("/clouds", api.IsAuthorized(api.GetClouds, true /* true: check for IP based white list*/))
	router.POST("/clouds", api.IsAuthorized(api.PostClouds, false /* true: check for IP based white list*/))
	router.GET("/clouds/:cloud_id", api.IsAuthorized(api.GetCloud, true /* true: check for IP based white list*/))
	router.DELETE("/clouds/:cloud_id", api.IsAuthorized(api.DeleteCloud, false /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/name", api.IsAuthorized(api.PostCloudName, false /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/paused", api.IsAuthorized(api.PostCloudPaused, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/username", api.IsAuthorized(api.PostCloudUsername, false /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/token", api.IsAuthorized(api.PostCloudToken, false /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/rest", api.IsAuthorized(api.PostCloudRESTAddr, false /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/mqtt", api.IsAuthorized(api.PostCloudMQTTAddr, false /* true: check for IP based white list*/))
	router.GET("/clouds/:cloud_id/status", api.IsAuthorized(api.GetCloudStatus, true /* true: check for IP based white list*/))
	router.GET("/clouds/:cloud_id/events", api.IsAuthorized(api.GetCloudEvents, true /* true: check for IP based white list*/))

	router.GET("/sys/uptime", api.IsAuthorized(api.SysGetUptime, true /* true: check for IP based white list*/))
	router.PUT("/sys/clear_all", api.IsAuthorized(api.SysClearAll, false /* true: check for IP based white list*/))
	router.GET("/sys/logs", api.IsAuthorized(api.SysGetLogs, true /* true: check for IP based white list*/))
	router.GET("/sys/log/:log_id", api.IsAuthorized(api.SysGetLog, true /* true: check for IP based white list*/))
	router.DELETE("/sys/log/:log_id", api.IsAuthorized(api.SysDeleteLog, false /* true: check for IP based white list*/))
}
