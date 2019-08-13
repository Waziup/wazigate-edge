package clouds

import (
	"fmt"
	"log"
	"time"
)

var retries = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	20 * time.Second,
	60 * time.Second,
}

func (cloud *Cloud) SetPaused(paused bool) {

	if cloud.pausing || paused == cloud.Paused {
		return
	}

	if paused {
		cloud.Paused = true
		cloud.pausing = true
	} else {
		cloud.Paused = false
		go cloud.sync()
	}
}

func (cloud *Cloud) sync() {

	nretry := 0

	retry := func() {

		if cloud.pausing {
			return
		}

		duration := retries[nretry]
		cloud.setStatus(cloud.StatusCode, fmt.Sprintf("Waiting %ds before retry after error.\n%s", duration/time.Second, cloud.StatusText))
		time.Sleep(duration)

		nretry++
		if nretry == len(retries) {
			nretry = len(retries) - 1
		}
	}

	for !cloud.pausing {

		cloud.setStatus(0, "Beginning initial sync ...")

		cloud.remote = make(map[entity]*remote)
		cloud.sigDirty = make(chan struct{})

		if !cloud.initialSync() {
			log.Println("[UP   ] Initial sync ended.")
			retry()
			continue
		}

		log.Printf("[UP   ] Initial sync completed with %d dirty. Now persistent sync ... ", len(cloud.remote))

		if !cloud.persistentSync() {
			log.Println("[UP   ] Persistent sync ended.")
			retry()
			continue
		}

		break
	}

	cloud.pausing = false
}
