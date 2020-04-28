package ontology

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type Unit int

func (u Unit) String() string {
	if u >= 0 && int(u) < len(unitStr) {
		return unitStr[u]
	}
	return ""
}

func (u Unit) MarshalJSON() ([]byte, error) {
	if u >= 0 && int(u) < len(unitStr) {
		return json.Marshal(unitStr[u])
	}
	return null, nil
}

func (u *Unit) UnmarshalJSON(data []byte) error {
	if bytes.Compare(data, null) == 0 {
		*u = 0
		return nil
	}
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	i := sort.SearchStrings(unitStr, str)
	if i == -1 {
		return fmt.Errorf("unknown ontology kind: %q", unitStr)
	}
	*u = Unit(i)
	return nil
}

var unitStr = []string{
	"",
	"Ampere",
	"Bar",
	"BeatPerMinute",
	"Candela",
	"Centibar",
	"Centimetre",
	"Coulomb",
	"Day",
	"Decibel",
	"DecibelA",
	"DecibelMilliwatt",
	"DegreeAngle",
	"DegreeAnglePerSecond",
	"DegreeCelsius",
	"DegreeFahrenheit",
	"Dimensionless",
	"EAQI",
	"Farad",
	"Gauss",
	"Gram",
	"GramPerCubicMetre",
	"GramPerLitre",
	"Hertz",
	"Hour",
	"Joule",
	"Kelvin",
	"KilobitsPerSecond",
	"Kilogram",
	"KilogramPerCubicMetre",
	"Kilometre",
	"KilometrePerHour",
	"KiloPascal",
	"KiloWattHour",
	"Litre",
	"LitrePer100Kilometres",
	"Lumen",
	"Lux",
	"Metre",
	"MetrePerSecond",
	"MetrePerSecondSquare",
	"Microampere",
	"Microgram",
	"MicrogramPerCubicMetre",
	"MicroSiemens",
	"Microvolt",
	"Microwatt",
	"MicrowattPerSquareCentimetre",
	"Milliampere",
	"Millibar",
	"Milligram",
	"MilligramPerCubicMetre",
	"MilligramPerLitre",
	"MilligramPerDecilitre",
	"MilligramPerSquareMetre",
	"Millilitre",
	"Millimetre",
	"MillimetreMercure",
	"MillimetrePerHour",
	"MilliMolPerLitre",
	"Millisecond",
	"Millivolt",
	"MillivoltPerMetre",
	"Milliwatt",
	"MinuteAngle",
	"MinuteTime",
	"Ohm",
	"Okta",
	"Other",
	"PartsPerBillion",
	"PartsPerMillion",
	"Pascal",
	"Percent",
	"Radian",
	"RadianPerSecond",
	"RadiationParticlesPerMinute",
	"RevolutionsPerMinute",
	"SecondAngle",
	"SecondTime",
	"Siemens",
	"SiemensPerMetre",
	"Tesla",
	"Tonne",
	"Volt",
	"VoltAmpereReactive",
	"VoltPerMetre",
	"Watt",
	"WattPerSquareMetre",
	"Weber",
	"Year",
}
