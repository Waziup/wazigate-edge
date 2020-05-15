package edge

import (
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

// Meta holds entity metadata.
type Meta map[string]interface{}

// DefaultInterval for sync.
var DefaultInterval = time.Second * 5

func init() {
	defaultDelay := os.Getenv("WAZIGATE_EDGE_DELAY")
	if defaultDelay != "" {
		duration, err := parseDuration(defaultDelay)
		if err != nil {
			log.Panicf("WAZIGATE_EDGE_DELAY is not a valid duration.\nSee https://pkg.go.dev/time?tab=doc#ParseDuration")
		}
		DefaultInterval = duration
	}
}

// SyncInterval = min time between syncs
func (meta Meta) SyncInterval() time.Duration {
	if meta != nil {
		if m := meta["syncInterval"]; m != nil {
			switch i := m.(type) {
			case string:
				j, err := parseDuration(i)
				if err != nil {
					log.Printf("[ERR  ] Meta 'syncInterval': %v", err)
					return DefaultInterval
				}
				return j
			}
		}
	}
	return DefaultInterval
}

// DoNotSync = do not sync with clouds
func (meta Meta) DoNotSync() bool {
	if meta != nil {
		if m := meta["doNotSync"]; m != nil {
			switch i := m.(type) {
			case bool:
				return i
			case int:
				return i != 0
			}
		}
	}
	return false
}

var durationRegex = regexp.MustCompile(`^\s*([\d\.]+Y)?\s*([\d\.]+M)?\s*([\d\.]+D)?T?\s*([\d\.]+h)?\s*([\d\.]+m)?\s*([\d\.]+?s)?\s*$`)

var errFormat = errors.New("invalid time duration format")

func parseDuration(str string) (time.Duration, error) {
	matches := durationRegex.FindStringSubmatch(str)
	if matches == nil {
		return 0, errFormat
	}
	Y := parseDurationFrag(matches[1], time.Hour*24*365)
	M := parseDurationFrag(matches[2], time.Hour*24*30)
	d := parseDurationFrag(matches[3], time.Hour*24)
	h := parseDurationFrag(matches[4], time.Hour)
	m := parseDurationFrag(matches[5], time.Second*60)
	s := parseDurationFrag(matches[6], time.Second)
	return time.Duration(Y + M + d + h + m + s), nil
}

func parseDurationFrag(f string, unit time.Duration) time.Duration {
	if len(f) != 0 {
		if parsed, err := strconv.ParseFloat(f[:len(f)-1], 64); err == nil {
			return time.Duration(float64(unit) * parsed)
		}
	}
	return 0
}
