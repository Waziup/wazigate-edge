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
	xlpp.TypeAltitude:     {},
	xlpp.TypeAnalogInput:  {},
	xlpp.TypeAnalogOutput: {},
	xlpp.TypeArray:        {},
	xlpp.TypeBarometricPressure: {
		Kind:     "PressureSensor",
		Quantity: "AtmosphericPressure",
		Unit:     "Hectopascal",
	},
	xlpp.TypeBinary:    {},
	xlpp.TypeBool:      {},
	xlpp.TypeBoolFalse: {},
	xlpp.TypeBoolTrue:  {},
	xlpp.TypeColour:    {},
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
	xlpp.TypeDigitalInput:  {},
	xlpp.TypeDigitalOutput: {},
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
	xlpp.TypeNull:   {},
	xlpp.TypeObject: {},
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
	xlpp.TypeString: {},
	xlpp.TypeSwitch: {},
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
	// xlpp.TypeAltitude:     {},
	// xlpp.TypeAnalogInput:  {},
	// xlpp.TypeAnalogOutput: {},
	// xlpp.TypeArray:        {},
	xlpp.TypeBarometricPressure: {
		Quantity: "AtmosphericPressure",
		Unit:     "Hectopascal",
	},
	// xlpp.TypeBinary:    {},
	// xlpp.TypeBool:      {},
	// xlpp.TypeBoolFalse: {},
	// xlpp.TypeBoolTrue:  {},
	// xlpp.TypeColour:    {},
	xlpp.TypeConcentration: {
		Quantity: "ChemicalAgentConcentration",
		Unit:     "PartsPerMillion",
	},
	xlpp.TypeCurrent: {
		Quantity: "ElectricCurrent",
		Unit:     "Ampere",
	},
	// xlpp.TypeDigitalInput:  {},
	// xlpp.TypeDigitalOutput: {},
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
	// xlpp.TypeNull:   {},
	// xlpp.TypeObject: {},
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
	// xlpp.TypeString: {},
	// xlpp.TypeSwitch: {},
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
		if a.Quantity == quantity && a.Unit == unit {
			return t
		}
	}
	return 255
}
