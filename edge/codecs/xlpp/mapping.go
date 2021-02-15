package codec

import (
	"github.com/waziup/xlpp"
)

type def struct {
	Kind     string
	Quantity string
	Unit     string
}

var sensorMapping = map[xlpp.Type]def{
	xlpp.TypeAccelerometer: {
		Kind:     "Accelerometer",
		Quantity: "Acceleration",
		Unit:     "MetrePerSecondSquare",
	},
	xlpp.TypeAltitude: {
		Quantity: "Altitude",
		Unit:     "Metre",
	},
	xlpp.TypeAnalogInput: {
		Quantity: "AnalogeValue",
	},
	xlpp.TypeAnalogOutput: {
		Quantity: "AnalogeValue",
	},
	xlpp.TypeArray: {
		Quantity: "Array",
	},
	xlpp.TypeBarometricPressure: {
		Kind:     "PressureSensor",
		Quantity: "AtmosphericPressure",
		Unit:     "Hectopascal",
	},
	xlpp.TypeBinary: {
		Quantity: "Binary",
	},
	xlpp.TypeBool: {
		Quantity: "Boolean",
	},
	xlpp.TypeBoolFalse: {
		Quantity: "Boolean",
	},
	xlpp.TypeBoolTrue: {
		Quantity: "Boolean",
	},
	xlpp.TypeColour: {
		Quantity: "Color",
	},
	xlpp.TypeConcentration: {
		Kind:     "GaseousPollutantSensor",
		Quantity: "ChemicalAgentConcentration",
		Unit:     "PartsPerMillion",
	},
	xlpp.TypeCurrent: {
		Kind:     "ElectricalSensor",
		Quantity: "ElectricCurrent",
		Unit:     "Ampere",
	},
	xlpp.TypeDigitalInput: {
		Quantity: "DigitalValue",
	},
	xlpp.TypeDigitalOutput: {
		Quantity: "DigitalValue",
	},
	xlpp.TypeDirection: {
		Kind:     "WindDirectionSensor", // or SunPositionDirectionSensor
		Quantity: "WindDirection",
		Unit:     "DegreeAngle",
	},
	xlpp.TypeDistance: {
		Kind:     "DistanceSensor",
		Quantity: "Distance",
		Unit:     "Metre",
	},
	xlpp.TypeEnergy: {
		Kind:     "EnergyMeter",
		Quantity: "Energy",
		Unit:     "KiloWattHour",
	},
	xlpp.TypeFlags: {},
	xlpp.TypeFrequency: {
		Kind:     "FrequencySensor",
		Quantity: "Frequency",
		Unit:     "Hertz",
	},
	xlpp.TypeGPS: {
		Kind:     "GPSSensor",
		Quantity: "Position",
		Unit:     "LatLong",
	},
	xlpp.TypeGyrometer: {
		Kind:     "GyroscopeSensor",
		Quantity: "RotationalSpeed",
		Unit:     "DegreeAnglePerSecond",
	},
	xlpp.TypeInteger: {},
	xlpp.TypeLuminosity: {
		Kind:     "LightSensor",
		Quantity: "Illuminance",
		Unit:     "Lux",
	},
	xlpp.TypeNull: {},
	xlpp.TypeObject: {
		Quantity: "Object",
	},
	xlpp.TypePercentage: {
		Unit: "Percent",
	},
	xlpp.TypePower: {
		Kind:     "ElectricalSensor",
		Quantity: "ActivePower",
		Unit:     "Watt",
	},
	xlpp.TypePresence: {
		Kind:     "HumanPresenceDetector",
		Quantity: "Presence",
	},
	xlpp.TypeRelativeHumidity: {
		Kind:     "HumiditySensor",
		Quantity: "RelativeHumidity",
		Unit:     "Percent",
	},
	xlpp.TypeString: {
		Quantity: "String",
	},
	xlpp.TypeSwitch: {
		Quantity: "Boolean",
	},
	xlpp.TypeTemperature: {
		Kind:     "Thermometer",
		Quantity: "Temperature",
		Unit:     "DegreeCelsius",
	},
	xlpp.TypeUnixTime: {
		Kind:     "Clock",
		Quantity: "Timestamp",
		Unit:     "SecondTime",
	},
	xlpp.TypeVoltage: {
		Kind:     "VoltageSensor",
		Quantity: "Voltage",
		Unit:     "Volt",
	},
}

var actuatorMapping = map[xlpp.Type]def{
	xlpp.TypeAccelerometer: {
		Quantity: "Acceleration",
		Unit:     "MetrePerSecondSquare",
	},
	xlpp.TypeAltitude: {
		Quantity: "Binary",
	},
	xlpp.TypeAnalogInput: {
		Quantity: "AnalogeValue",
	},
	xlpp.TypeAnalogOutput: {
		Quantity: "AnalogeValue",
	},
	xlpp.TypeArray: {
		Quantity: "Array",
	},
	xlpp.TypeBarometricPressure: {
		Quantity: "AtmosphericPressure",
		Unit:     "Hectopascal",
	},
	xlpp.TypeBinary: {
		Quantity: "Binary",
	},
	xlpp.TypeBool: {
		Quantity: "Boolean",
	},
	xlpp.TypeBoolFalse: {
		Quantity: "Boolean",
	},
	xlpp.TypeBoolTrue: {
		Quantity: "Boolean",
	},
	xlpp.TypeColour: {
		Quantity: "Color",
	},
	xlpp.TypeConcentration: {
		Quantity: "ChemicalAgentConcentration",
		Unit:     "PartsPerMillion",
	},
	xlpp.TypeCurrent: {
		Quantity: "ElectricCurrent",
		Unit:     "Ampere",
	},
	xlpp.TypeDigitalInput: {
		Quantity: "DigitalValue",
	},
	xlpp.TypeDigitalOutput: {
		Quantity: "DigitalValue",
	},
	xlpp.TypeDirection: {
		Quantity: "WindDirection",
		Unit:     "DegreeAngle",
	},
	xlpp.TypeDistance: {
		Quantity: "Distance",
		Unit:     "Metre",
	},
	xlpp.TypeEnergy: {
		Quantity: "Energy",
		Unit:     "KiloWattHour",
	},
	// xlpp.TypeFlags: {},
	xlpp.TypeFrequency: {
		Quantity: "Frequency",
		Unit:     "Hertz",
	},
	xlpp.TypeGPS: {
		Quantity: "Position",
		Unit:     "LatLong",
	},
	xlpp.TypeGyrometer: {
		Quantity: "RotationalSpeed",
		Unit:     "DegreeAnglePerSecond",
	},
	xlpp.TypeInteger: {},
	xlpp.TypeLuminosity: {
		Quantity: "Illuminance",
		Unit:     "Lux",
	},
	xlpp.TypeNull: {},
	xlpp.TypeObject: {
		Quantity: "Object",
	},
	xlpp.TypePercentage: {
		Unit: "Percent",
	},
	xlpp.TypePower: {
		Quantity: "ActivePower",
		Unit:     "Watt",
	},
	xlpp.TypePresence: {
		Quantity: "Presence",
	},
	xlpp.TypeRelativeHumidity: {
		Quantity: "RelativeHumidity",
		Unit:     "Percent",
	},
	xlpp.TypeString: {
		Quantity: "String",
	},
	xlpp.TypeSwitch: {
		Quantity: "Boolean",
	},
	xlpp.TypeTemperature: {
		Quantity: "Temperature",
		Unit:     "DegreeCelsius",
	},
	xlpp.TypeUnixTime: {
		Quantity: "Timestamp",
		Unit:     "SecondTime",
	},
	xlpp.TypeVoltage: {
		Quantity: "Voltage",
		Unit:     "Volt",
	},
}

func typeFromDef(quantity string, unit string) xlpp.Type {
	for t, a := range actuatorMapping {
		if a.Quantity == quantity && (a.Unit == unit || a.Unit == "") {
			return t
		}
	}
	return 255
}
