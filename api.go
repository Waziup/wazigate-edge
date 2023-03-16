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
	router.POST("/auth/retoken", api.IsAuthorized(api.GetRefereshToken, true))
	router.GET("/auth/logout", api.Logout)
	router.POST("/auth/logout", api.Logout)
	router.GET("/auth/permissions", api.GetPermissions)

	router.GET("/auth/profile", api.IsAuthorized(api.GetUserProfile, true))
	router.POST("/auth/profile", api.IsAuthorized(api.PostUserProfile, true))

	// COdecs

	router.GET("/codecs", api.IsAuthorized(api.GetCodecs, true))
	router.POST("/codecs", api.IsAuthorized(api.PostCodecs, true))
	router.POST("/codecs/:codec_id", api.IsAuthorized(api.PostCodec, true))
	router.DELETE("/codecs/:codec_id", api.IsAuthorized(api.DeleteCodec, true))

	//Apps

	router.GET("/apps", api.IsAuthorized(api.GetApps, true))
	router.GET("/apps/:app_id", api.IsAuthorized(api.GetApp, true))

	router.POST("/apps/:app_id", api.IsAuthorized(api.PostApp, true))
	router.DELETE("/apps/:app_id", api.IsAuthorized(api.DeleteApp, true))

	router.POST("/apps", api.IsAuthorized(api.PostApps, true)) // install a new app

	router.GET("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true))
	router.POST("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true))
	router.PUT("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true))
	router.DELETE("/apps/:app_id/*file_path", api.IsAuthorized(api.HandleAppProxyRequest, true))

	//Update Appps

	// router.GET("/update", api.GetUpdateApps, false))
	router.GET("/update/:app_id", api.IsAuthorized(api.GetUpdateApp, true))

	// router.POST("/update", api.PostUpdateApps, false))
	router.POST("/update/:app_id", api.IsAuthorized(api.PostUpdateApp, true))

	// Device Endpoints

	router.GET("/devices", api.IsAuthorized(api.GetDevices, true))
	router.POST("/devices", api.IsAuthorized(api.PostDevices, true))
	router.GET("/devices/:device_id", api.IsAuthorized(api.GetDevice, true))
	router.POST("/devices/:device_id", api.IsAuthorized(api.PostDevice, true))
	router.DELETE("/devices/:device_id", api.IsAuthorized(api.DeleteDevice, true))
	router.GET("/devices/:device_id/name", api.IsAuthorized(api.GetDeviceName, true))
	router.POST("/devices/:device_id/name", api.IsAuthorized(api.PostDeviceName, true))
	router.GET("/devices/:device_id/meta", api.IsAuthorized(api.GetDeviceMeta, true))
	router.POST("/devices/:device_id/meta", api.IsAuthorized(api.PostDeviceMeta, true))

	// Sensor Endpoints

	router.GET("/devices/:device_id/sensors", api.IsAuthorized(api.GetDeviceSensors, true))
	router.GET("/devices/:device_id/sensors/:sensor_id", api.IsAuthorized(api.GetDeviceSensor, true))

	router.POST("/devices/:device_id/sensors", api.IsAuthorized(api.PostDeviceSensor, true))
	router.DELETE("/devices/:device_id/sensors/:sensor_id", api.IsAuthorized(api.DeleteDeviceSensor, true))
	router.POST("/devices/:device_id/sensors/:sensor_id/name", api.IsAuthorized(api.PostDeviceSensorName, true))
	router.POST("/devices/:device_id/sensors/:sensor_id/meta", api.IsAuthorized(api.PostDeviceSensorMeta, true))
	router.POST("/devices/:device_id/sensors/:sensor_id/meta/:meta", api.IsAuthorized(api.PostDeviceSensorMeta, true))

	router.GET("/devices/:device_id/sensors/:sensor_id/value", api.IsAuthorized(api.GetDeviceSensorValue, true))
	router.GET("/devices/:device_id/sensors/:sensor_id/values", api.IsAuthorized(api.GetDeviceSensorValues, true))

	router.POST("/devices/:device_id/sensors/:sensor_id/value", api.IsAuthorized(api.PostDeviceSensorValue, true))
	router.POST("/devices/:device_id/sensors/:sensor_id/values", api.IsAuthorized(api.PostDeviceSensorValues, true))

	// Actuator Endpoints

	router.GET("/devices/:device_id/actuators", api.IsAuthorized(api.GetDeviceActuators, true))
	router.GET("/devices/:device_id/actuators/:actuator_id", api.IsAuthorized(api.GetDeviceActuator, true))

	router.POST("/devices/:device_id/actuators", api.IsAuthorized(api.PostDeviceActuator, true))
	router.DELETE("/devices/:device_id/actuators/:actuator_id", api.IsAuthorized(api.DeleteDeviceActuator, true))
	router.POST("/devices/:device_id/actuators/:actuator_id/name", api.IsAuthorized(api.PostDeviceActuatorName, true))
	router.POST("/devices/:device_id/actuators/:actuator_id/meta", api.IsAuthorized(api.PostDeviceActuatorMeta, true))

	router.GET("/devices/:device_id/actuators/:actuator_id/value", api.IsAuthorized(api.GetDeviceActuatorValue, true))
	router.GET("/devices/:device_id/actuators/:actuator_id/values", api.IsAuthorized(api.GetDeviceActuatorValues, true))

	router.POST("/devices/:device_id/actuators/:actuator_id/value", api.IsAuthorized(api.PostDeviceActuatorValue, true))
	router.POST("/devices/:device_id/actuators/:actuator_id/values", api.IsAuthorized(api.PostDeviceActuatorValues, true))

	// Shortcut Endpoints (equals device_id = current device ID, true))

	router.GET("/device", api.IsAuthorized(api.GetCurrentDevice, true))
	router.GET("/device/id", api.GetCurrentDeviceID)
	router.GET("/device/name", api.IsAuthorized(api.GetCurrentDeviceName, true))
	router.GET("/device/meta", api.IsAuthorized(api.GetCurrentDeviceMeta, true))
	router.POST("/device/id", api.IsAuthorized(api.PostCurrentDeviceID, true))
	router.POST("/device/name", api.IsAuthorized(api.PostCurrentDeviceName, true))
	router.POST("/device/meta", api.IsAuthorized(api.PostCurrentDeviceMeta, true))

	router.GET("/sensors", api.IsAuthorized(api.GetSensors, true))
	router.POST("/sensors", api.IsAuthorized(api.PostSensor, true))
	router.GET("/sensors/:sensor_id", api.IsAuthorized(api.GetSensor, true))
	router.DELETE("/sensors/:sensor_id", api.IsAuthorized(api.DeleteSensor, true))
	router.GET("/sensors/:sensor_id/value", api.IsAuthorized(api.GetSensorValue, true))
	router.GET("/sensors/:sensor_id/values", api.IsAuthorized(api.GetSensorValues, true))
	router.POST("/sensors/:sensor_id/name", api.IsAuthorized(api.PostSensorName, true))
	router.POST("/sensors/:sensor_id/meta", api.IsAuthorized(api.PostSensorMeta, true))

	router.GET("/actuators", api.IsAuthorized(api.GetActuators, true))
	router.POST("/actuators", api.IsAuthorized(api.PostActuator, true))
	router.GET("/actuators/:actuator_id", api.IsAuthorized(api.GetActuator, true))
	router.DELETE("/actuators/:actuator_id", api.IsAuthorized(api.DeleteActuator, true))
	router.GET("/actuators/:actuator_id/value", api.IsAuthorized(api.GetActuatorValue, true))
	router.GET("/actuators/:actuator_id/values", api.IsAuthorized(api.GetActuatorValues, true))
	router.POST("/actuators/:actuator_id/name", api.IsAuthorized(api.PostActuatorName, true))
	router.POST("/actuators/:actuator_id/meta", api.IsAuthorized(api.PostActuatorMeta, true))

	router.POST("/sensors/:sensor_id/value", api.IsAuthorized(api.PostSensorValue, true))
	router.POST("/sensors/:sensor_id/values", api.IsAuthorized(api.PostSensorValues, true))

	router.POST("/actuators/:actuator_id/value", api.IsAuthorized(api.PostSensorValue, true))
	router.POST("/actuators/:actuator_id/values", api.IsAuthorized(api.PostSensorValues, true))

	// Messages

	router.POST("/messages", api.IsAuthorized(api.PostMessage, true /* true: check for IP based white list*/))
	router.GET("/messages", api.IsAuthorized(api.GetMessages, true /* true: check for IP based white list*/))

	// Clouds configuration

	router.GET("/clouds", api.IsAuthorized(api.GetClouds, true /* true: check for IP based white list*/))
	router.POST("/clouds", api.IsAuthorized(api.PostClouds, true /* true: check for IP based white list*/))
	router.GET("/clouds/:cloud_id", api.IsAuthorized(api.GetCloud, true /* true: check for IP based white list*/))
	router.DELETE("/clouds/:cloud_id", api.IsAuthorized(api.DeleteCloud, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/name", api.IsAuthorized(api.PostCloudName, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/paused", api.IsAuthorized(api.PostCloudPaused, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/username", api.IsAuthorized(api.PostCloudUsername, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/token", api.IsAuthorized(api.PostCloudToken, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/rest", api.IsAuthorized(api.PostCloudRESTAddr, true /* true: check for IP based white list*/))
	router.POST("/clouds/:cloud_id/mqtt", api.IsAuthorized(api.PostCloudMQTTAddr, true /* true: check for IP based white list*/))
	router.GET("/clouds/:cloud_id/status", api.IsAuthorized(api.GetCloudStatus, true /* true: check for IP based white list*/))
	router.GET("/clouds/:cloud_id/events", api.IsAuthorized(api.GetCloudEvents, true /* true: check for IP based white list*/))

	// Export, Backup and Import

	router.GET("/exportall", api.IsAuthorized(api.GetExportAllInOne, true /* true: check for IP based white list*/))
	router.GET("/exporttree", api.IsAuthorized(api.GetExportTree, true /* true: check for IP based white list*/))
	router.GET("/exportforml", api.IsAuthorized(api.GetExportMlBins, true /* true: check for IP based white list*/))

	// Sys

	router.GET("/sys/uptime", api.IsAuthorized(api.SysGetUptime, true /* true: check for IP based white list*/))
	router.PUT("/sys/clear_all", api.IsAuthorized(api.SysClearAll, true /* true: check for IP based white list*/))
	router.GET("/sys/logs", api.IsAuthorized(api.SysGetLogs, true /* true: check for IP based white list*/))
	router.GET("/sys/log/:log_id", api.IsAuthorized(api.SysGetLog, true /* true: check for IP based white list*/))
	router.DELETE("/sys/log/:log_id", api.IsAuthorized(api.SysDeleteLog, true /* true: check for IP based white list*/))

	router.GET("/version", api.IsAuthorized(api.SysGetVersion, true /* true: check for IP based white list*/))
	router.GET("/buildnr", api.IsAuthorized(api.SysGetBuildNr, true /* true: check for IP based white list*/))

	router.GET("/info", api.IsAuthorized(api.SysGetInfo, true /* true: check for IP based white list*/))
}
