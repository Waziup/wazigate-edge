package edge

import (
	"log"
	"os"
	"time"
)

// Meta holds entity metadata.
type Meta map[string]interface{}

// DefaultInterval for sync.
var DefaultInterval = time.Second * 5

func init() {
	defaultDelay := os.Getenv("WAZIGATE_EDGE_DELAY")
	if defaultDelay != "" {
		duration, err := time.ParseDuration(defaultDelay)
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
				if j, err := time.ParseDuration(i); err == nil {
					return j
				}
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
