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
	router.POST("/auth/retoken", api.IsAuthorized( api.GetRefereshToken))
	router.GET("/auth/logout", api.Logout)
	router.POST("/auth/logout", api.Logout)
	router.GET("/auth/permissions", api.GetPermissions)
	
	router.GET("/auth/profile", api.IsAuthorized( api.GetUserProfile))
	router.POST("/auth/profile", api.IsAuthorized( api.PostUserProfile))

	//Apps

	router.GET("/apps", api.IsAuthorized( api.GetApps))
	router.GET("/apps/:app_id", api.IsAuthorized( api.GetApp))

	router.POST("/apps/:app_id", api.IsAuthorized( api.PostApp))
	router.DELETE("/apps/:app_id", api.IsAuthorized( api.DeleteApp))

	router.POST("/apps", api.IsAuthorized( api.PostApps)) // install a new app

	router.GET("/apps/:app_id/*file_path", api.IsAuthorized( api.HandleAppProxyRequest))
	router.POST("/apps/:app_id/*file_path", api.IsAuthorized( api.HandleAppProxyRequest))
	router.PUT("/apps/:app_id/*file_path", api.IsAuthorized( api.HandleAppProxyRequest))
	router.DELETE("/apps/:app_id/*file_path", api.IsAuthorized( api.HandleAppProxyRequest))

	//Update Appps

	// router.GET("/update", api.GetUpdateApps))
	router.GET("/update/:app_id", api.IsAuthorized( api.GetUpdateApp))

	// router.POST("/update", api.PostUpdateApps))
	router.POST("/update/:app_id", api.IsAuthorized( api.PostUpdateApp))

	// Device Endpoints

	router.GET("/devices", api.IsAuthorized( api.GetDevices))
	router.GET("/devices/:device_id", api.IsAuthorized( api.GetDevice))
	router.POST("/devices", api.IsAuthorized( api.PostDevice))
	router.DELETE("/devices/:device_id", api.IsAuthorized( api.DeleteDevice))
	router.GET("/devices/:device_id/name", api.IsAuthorized( api.GetDeviceName))
	router.POST("/devices/:device_id/name", api.IsAuthorized( api.PostDeviceName))
	router.GET("/devices/:device_id/meta", api.IsAuthorized( api.GetDeviceMeta))
	router.POST("/devices/:device_id/meta", api.IsAuthorized( api.PostDeviceMeta))

	// Sensor Endpoints

	router.GET("/devices/:device_id/sensors", api.IsAuthorized( api.GetDeviceSensors))
	router.GET("/devices/:device_id/sensors/:sensor_id", api.IsAuthorized( api.GetDeviceSensor))

	router.POST("/devices/:device_id/sensors", api.IsAuthorized( api.PostDeviceSensor))
	router.DELETE("/devices/:device_id/sensors/:sensor_id", api.IsAuthorized( api.DeleteDeviceSensor))
	router.POST("/devices/:device_id/sensors/:sensor_id/name", api.IsAuthorized( api.PostDeviceSensorName))
	router.POST("/devices/:device_id/sensors/:sensor_id/meta", api.IsAuthorized( api.PostDeviceSensorMeta))

	router.GET("/devices/:device_id/sensors/:sensor_id/value", api.IsAuthorized( api.GetDeviceSensorValue))
	router.GET("/devices/:device_id/sensors/:sensor_id/values", api.IsAuthorized( api.GetDeviceSensorValues))

	router.POST("/devices/:device_id/sensors/:sensor_id/value", api.IsAuthorized( api.PostDeviceSensorValue))
	router.POST("/devices/:device_id/sensors/:sensor_id/values", api.IsAuthorized( api.PostDeviceSensorValues))

	// Actuator Endpoints

	router.GET("/devices/:device_id/actuators", api.IsAuthorized( api.GetDeviceActuators))
	router.GET("/devices/:device_id/actuators/:actuator_id", api.IsAuthorized( api.GetDeviceActuator))

	router.POST("/devices/:device_id/actuators", api.IsAuthorized( api.PostDeviceActuator))
	router.DELETE("/devices/:device_id/actuators/:actuator_id", api.IsAuthorized( api.DeleteDeviceActuator))
	router.POST("/devices/:device_id/actuators/:actuator_id/name", api.IsAuthorized( api.PostDeviceActuatorName))
	router.POST("/devices/:device_id/actuators/:actuator_id/meta", api.IsAuthorized( api.PostDeviceActuatorMeta))

	router.GET("/devices/:device_id/actuators/:actuator_id/value", api.IsAuthorized( api.GetDeviceActuatorValue))
	router.GET("/devices/:device_id/actuators/:actuator_id/values", api.IsAuthorized( api.GetDeviceActuatorValues))

	router.POST("/devices/:device_id/actuators/:actuator_id/value", api.IsAuthorized( api.PostDeviceActuatorValue))
	router.POST("/devices/:device_id/actuators/:actuator_id/values", api.IsAuthorized( api.PostDeviceActuatorValues))

	// Shortcut Endpoints (equals device_id = current device ID))

	router.GET("/device", api.IsAuthorized( api.GetCurrentDevice))
	router.GET("/device/id", api.IsAuthorized( api.GetCurrentDeviceID))
	router.GET("/device/name", api.IsAuthorized( api.GetCurrentDeviceName))
	router.GET("/device/meta", api.IsAuthorized( api.GetCurrentDeviceMeta))
	router.POST("/device/name", api.IsAuthorized( api.PostCurrentDeviceName))
	router.POST("/device/meta", api.IsAuthorized( api.PostCurrentDeviceMeta))

	router.GET("/sensors", api.IsAuthorized( api.GetSensors))
	router.POST("/sensors", api.IsAuthorized( api.PostSensor))
	router.GET("/sensors/:sensor_id", api.IsAuthorized( api.GetSensor))
	router.DELETE("/sensors/:sensor_id", api.IsAuthorized( api.DeleteSensor))
	router.GET("/sensors/:sensor_id/value", api.IsAuthorized( api.GetSensorValue))
	router.GET("/sensors/:sensor_id/values", api.IsAuthorized( api.GetSensorValues))
	router.POST("/sensors/:sensor_id/name", api.IsAuthorized( api.PostSensorName))
	router.POST("/sensors/:sensor_id/meta", api.IsAuthorized( api.PostSensorMeta))

	router.GET("/actuators", api.IsAuthorized( api.GetActuators))
	router.POST("/actuators", api.IsAuthorized( api.PostActuator))
	router.GET("/actuators/:actuator_id", api.IsAuthorized( api.GetActuator))
	router.DELETE("/actuators/:actuator_id", api.IsAuthorized( api.DeleteActuator))
	router.GET("/actuators/:actuator_id/value", api.IsAuthorized( api.GetActuatorValue))
	router.GET("/actuators/:actuator_id/values", api.IsAuthorized( api.GetActuatorValues))
	router.POST("/actuators/:actuator_id/name", api.IsAuthorized( api.PostActuatorName))
	router.POST("/actuators/:actuator_id/meta", api.IsAuthorized( api.PostActuatorMeta))

	router.POST("/sensors/:sensor_id/value", api.IsAuthorized( api.PostSensorValue))
	router.POST("/sensors/:sensor_id/values", api.IsAuthorized( api.PostSensorValues))

	router.POST("/actuators/:actuator_id/value", api.IsAuthorized( api.PostSensorValue))
	router.POST("/actuators/:actuator_id/values", api.IsAuthorized( api.PostSensorValues))

	// Clouds configuration

	router.GET("/clouds", api.IsAuthorized( api.GetClouds))
	router.POST("/clouds", api.IsAuthorized( api.PostClouds))
	router.GET("/clouds/:cloud_id", api.IsAuthorized( api.GetCloud))
	router.DELETE("/clouds/:cloud_id", api.IsAuthorized( api.DeleteCloud))
	router.POST("/clouds/:cloud_id/name", api.IsAuthorized( api.PostCloudName))
	router.POST("/clouds/:cloud_id/paused", api.IsAuthorized( api.PostCloudPaused))
	router.POST("/clouds/:cloud_id/username", api.IsAuthorized( api.PostCloudUsername))
	router.POST("/clouds/:cloud_id/token", api.IsAuthorized( api.PostCloudToken))
	router.POST("/clouds/:cloud_id/rest", api.IsAuthorized( api.PostCloudRESTAddr))
	router.POST("/clouds/:cloud_id/mqtt", api.IsAuthorized( api.PostCloudMQTTAddr))
	router.GET("/clouds/:cloud_id/status", api.IsAuthorized( api.GetCloudStatus))
	router.GET("/clouds/:cloud_id/events", api.IsAuthorized( api.GetCloudEvents))

	router.GET("/sys/uptime", api.IsAuthorized( api.SysGetUptime))
	router.PUT("/sys/clear_all", api.IsAuthorized( api.SysClearAll))
	router.GET("/sys/logs", api.IsAuthorized( api.SysGetLogs))
	router.GET("/sys/log/:log_id", api.IsAuthorized( api.SysGetLog))
	router.DELETE("/sys/log/:log_id", api.IsAuthorized( api.SysDeleteLog))
}
