package ontology

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type Quantity int

func (q Quantity) String() string {
	if q >= 0 && int(q) < len(quantityStr) {
		return quantityStr[q]
	}
	return ""
}

func (q Quantity) MarshalJSON() ([]byte, error) {
	if q >= 0 && int(q) < len(quantityStr) {
		return json.Marshal(quantityStr[q])
	}
	return null, nil
}

func (q *Quantity) UnmarshalJSON(data []byte) error {
	if bytes.Compare(data, null) == 0 {
		*q = 0
		return nil
	}
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	i := sort.SearchStrings(quantityStr, str)
	if i == -1 {
		return fmt.Errorf("unknown ontology kind: %q", quantityStr)
	}
	*q = Quantity(i)
	return nil
}

var quantityStr = []string{
	"",
	"Acceleration",
	"AccelerationInstantaneous",
	"ActivePower",
	"AirPollution",
	"AirQuality",
	"AirTemperature",
	"AirHumidity",
	"AlcoholLevel",
	"AngularSpeed",
	"AtmosphericPressure",
	"BatteryLevel",
	"BloodGlucose",
	"BloodPressure",
	"BoardTemperature",
	"BodyTemperature",
	"BuildingTemperature",
	"Capacitance",
	"ChemicalAgentAtmosphericConcentration",
	"ChemicalAgentAtmosphericConcentrationAirParticles",
	"ChemicalAgentAtmosphericConcentrationCO",
	"ChemicalAgentAtmosphericConcentrationDust",
	"ChemicalAgentAtmosphericConcentrationNH3",
	"ChemicalAgentAtmosphericConcentrationNO",
	"ChemicalAgentAtmosphericConcentrationNO2",
	"ChemicalAgentAtmosphericConcentrationO3",
	"ChemicalAgentAtmosphericConcentrationSO2",
	"ChemicalAgentAtmosphericConcentrationVOC",
	"ChemicalAgentConcentration",
	"ChemicalAgentWaterConcentration",
	"ChemicalAgentWaterConcentrationNH4Ion",
	"ChemicalAgentWaterConcentrationNO3Ion",
	"ChemicalAgentWaterConcentrationO2",
	"Cholesterol",
	"CloudCover",
	"CO2",
	"Conductivity",
	"Count",
	"CountAvailableVehicles",
	"CountEmptyDockingPoints",
	"CountPeople",
	"DeltaDewPoint",
	"DeviceUptime",
	"DewPoint",
	"DewPointTemperature",
	"DirectionOfArrival",
	"Distance",
	"DistanceNextVehicle",
	"DoorStatus",
	"ElectricalResistance",
	"ElectricCharge",
	"ElectricCurrent",
	"ElectricField",
	"ElectricPotential",
	"Energy",
	"FillLevel",
	"FillLevelGasTank",
	"FillLevelWasteContainer",
	"FoodTemperature",
	"Frequency",
	"FuelConsumption",
	"FuelConsumptionInstantaneous",
	"FuelConsumptionTotal",
	"HeartBeat",
	"HouseholdApplianceTemperature",
	"Humidity",
	"Illuminance",
	"IonisingRadiation",
	"LeafWetness",
	"LuminousFlux",
	"LuminousIntensity",
	"MagneticField",
	"MagneticFluxDensity",
	"Mass",
	"Mileage",
	"MileageDistanceToService",
	"MileageTotal",
	"Motion",
	"MotionState",
	"MotionStateVehicle",
	"Orientation",
	"Other",
	"PH",
	"Position",
	"Power",
	"Precipitation",
	"Presence",
	"PresenceStateParking",
	"PresenceStatePeople",
	"Pressure",
	"Proximity",
	"Rainfall",
	"ReactivePower",
	"RelativeHumidity",
	"RoadTemperature",
	"RoomTemperature",
	"RotationalSpeed",
	"Salinity",
	"SkinConductance",
	"SoilHumidity",
	"SoilMoistureTension",
	"SoilTemperature",
	"SolarRadiation",
	"Sound",
	"SoundPressureLevel",
	"SoundPressureLevelAmbient",
	"Speed",
	"SpeedAverage",
	"SpeedInstantaneous",
	"SPO2",
	"SunPositionDirection",
	"SunPositionElevation",
	"Temperature",
	"TemperatureEngine",
	"TemperatureWasteContainer",
	"TimeOfArrival",
	"TimeOfArrivalNextVehicle",
	"Timestamp",
	"TrafficIntensity",
	"Visibility",
	"VoiceCommand",
	"Voltage",
	"WaterLevel",
	"WaterTemperature",
	"WeatherLuminosity",
	"WeatherPrecipitation",
	"Weight",
	"WindChill",
	"WindDirection",
	"WindSpeed",
	"WorkingState",
}
