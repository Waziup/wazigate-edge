package edge

import "time"

// Meta holds entity metadata.
type Meta map[string]interface{}

// DefaultInterval for sync.
const DefaultInterval = time.Second * 5

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
