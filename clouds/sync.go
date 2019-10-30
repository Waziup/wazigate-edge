package clouds

import (
	"errors"
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

var errLoginFailed = errors.New("login failed")

func (cloud *Cloud) SetPaused(paused bool) (int, error) {

	if cloud.Pausing || cloud.PausingMQTT {
		return http.StatusLocked, errCloudNoPause
	}

	cloud.remoteMutex.Lock()
	defer cloud.remoteMutex.Unlock()

	if cloud.Paused == paused {
		return 200, nil
	}

	if paused {
		cloud.Paused = true
		cloud.Pausing = true
		cloud.PausingMQTT = true

		cloud.mqttMutex.Lock()
		if cloud.client != nil {
			cloud.client.Disconnect()
		}
		cloud.mqttMutex.Unlock()

		select {
		case cloud.sigDirty <- struct{}{}:
		default: // channel full
		}
		return 200, nil
	}

	status := cloud.authenticate()
	if status == 0 {
		status = http.StatusAccepted
	}

	if !isOk(status) {
		return status, errLoginFailed
	}

	cloud.Paused = false
	go cloud.sync()
	return status, nil
}

func (cloud *Cloud) sync() {

	nretry := 0

	retry := func() {

		if cloud.Pausing {
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

	////

	auth := func() {
		for !cloud.Pausing {
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

	////

	activeMQTT := false

	////

INITIAL_SYNC:
	for !cloud.Pausing {

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

	if !activeMQTT && !cloud.Pausing {
		activeMQTT = true
		go cloud.mqttSync()
	}

	for !cloud.Pausing {

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

	cloud.Pausing = false
	log.Println("[UP   ] REST sync is now paused.")
	if !activeMQTT {
		cloud.PausingMQTT = false
		log.Println("[UP   ] MQTT sync is now paused.")
	}
}
