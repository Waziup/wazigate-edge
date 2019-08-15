package clouds

import (
	"fmt"
	"log"
	"net/http"
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

	auth := func() {
		for !cloud.pausing {
			status := cloud.authenticate()
			if status == http.StatusForbidden || status == http.StatusUnauthorized {
				cloud.SetPaused(true)
				break
			}
			if !isOk(status) {
				retry()
				continue
			}
			break
		}
	}

	auth()

INITIAL_SYNC:
	for !cloud.pausing {

		cloud.setStatus(0, "Beginning initial sync ...")

		cloud.remote = make(map[entity]*remote)
		cloud.sigDirty = make(chan struct{}, 1)

		status := cloud.initialSync()
		if status == http.StatusForbidden || status == http.StatusUnauthorized {
			auth()
			continue
		}

		if !isOk(status) {
			retry()
			continue
		}

		log.Printf("[UP   ] Initial sync completed with %d dirty.", len(cloud.remote))
		nretry = 0
		break
	}

	for !cloud.pausing {

		status := cloud.persistentSync()
		if status == http.StatusForbidden || status == http.StatusUnauthorized {
			auth()
			continue
		}

		if status <= 0 { // Network Error
			retry()
			continue
		}

		if !isOk(status) {
			retry()
			goto INITIAL_SYNC
		}

		nretry = 0
	}

	cloud.pausing = false
}
