package ontology

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type SensingKind int

var null = []byte("null")

func (k SensingKind) String() string {
	if k >= 0 && int(k) < len(sensingKindStr) {
		return sensingKindStr[k]
	}
	return ""
}

func (k SensingKind) MarshalJSON() ([]byte, error) {
	if k >= 0 && int(k) < len(sensingKindStr) {
		return json.Marshal(sensingKindStr[k])
	}
	return null, nil
}

func (k *SensingKind) UnmarshalJSON(data []byte) error {
	if bytes.Compare(data, null) == 0 {
		*k = 0
		return nil
	}
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	i := sort.SearchStrings(sensingKindStr, str)
	if i == -1 {
		return fmt.Errorf("unknown ontology kind: %q", sensingKindStr)
	}
	*k = SensingKind(i)
	return nil
}

var sensingKindStr = []string{
	"",
	"Accelerometer",
	"AirHumiditySensor",
	"AirPollutantSensor",
	"AirThermometer",
	"AlcoholLevelSensor",
	"AtmosphericPressureSensor",
	"BloodPressureSensor",
	"BoardThermometer",
	"BoardVoltageSensor",
	"BodyThermometer",
	"CholesterolSensor",
	"Clock",
	"CloudCoverSensor",
	"CO2Sensor",
	"ConductivitySensor",
	"COSensor",
	"Counter",
	"CurrentSensor",
	"DeltaDewPointSensor",
	"DeviceUptimeClock",
	"DewPointSensor",
	"DirectionOfArrivalSensor",
	"DistanceNextVehicleSensor",
	"DistanceSensor",
	"DoorStateSensor",
	"DustSensor",
	"ElectricalSensor",
	"ElectricFieldSensor",
	"EnergyMeter",
	"FallDetector",
	"FrequencySensor",
	"FuelLevel",
	"FuelConsumptionSensor",
	"GasDetector",
	"GaseousPollutantSensor",
	"Glucometer",
	"GPSSensor",
	"GyroscopeSensor",
	"HeartBeatSensor",
	"HumanPresenceDetector",
	"HumiditySensor",
	"Hydrophone",
	"ImageSensor",
	"LeafWetnessSensor",
	"LightSensor",
	"LoRaInterfaceEnergyMeter",
	"Magnetometer",
	"MotionSensor",
	"NH3Sensor",
	"NO2Sensor",
	"NOSensor",
	"O3Sensor",
	"Odometer",
	"OpticalDustSensor",
	"OxidationReductionPotentialSensor",
	"OxygenSensor",
	"OtherSensor",
	"Pedometer",
	"PeopleCountSensor",
	"PHSensor",
	"PrecipitationSensor",
	"PresenceDetector",
	"PressureSensor",
	"ProximitySensor",
	"PulseOxymeter",
	"RadiationParticleDetector",
	"RainFallSensor",
	"RoadSurfaceThermometer",
	"SaltMeter",
	"Seismometer",
	"SkinConductanceSensor",
	"SmokeDetector",
	"SO2Sensor",
	"SoilHumiditySensor",
	"SoilThermometer",
	"SolarRadiationSensor",
	"SoundSensor",
	"SpeedSensor",
	"SunPositionDirectionSensor",
	"SunPositionElevationSensor",
	"Thermometer",
	"TimeOfArrivalNextVehicleSensor",
	"TimeOfArrivalSensor",
	"TouchSensor",
	"UltrasonicSensor",
	"VehicleCountSensor",
	"VehiclePresenceDetector",
	"VisibilitySensor",
	"VOCSensor",
	"VoiceCommandSensor",
	"VoltageSensor",
	"WasteLevelSensor",
	"WaterLevel",
	"WaterConductivitySensor",
	"WaterNH4IonSensor",
	"WaterNO3IonSensor",
	"WaterO2IonSensor",
	"WaterPHSensor",
	"WaterPollutantSensor",
	"WaterThermometer",
	"WeightSensor",
	"WiFiInterfaceEnergyMeter",
	"WindChillSensor",
	"WindDirectionSensor",
	"WindSpeedSensor",
}
