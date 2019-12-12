package clouds

import (
	"errors"
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

var errPausing = errors.New("cloud is pausing")

var errLoginFailed = errors.New("login failed")

// SetPaused stops or resumes the sync manager.
func (cloud *Cloud) SetPaused(paused bool) (int, error) {

	if cloud.Pausing || cloud.PausingMQTT {
		return http.StatusLocked, errCloudNoPause
	}

	cloud.StatusMutex.Lock()
	defer cloud.StatusMutex.Unlock()

	if cloud.Paused == paused {
		return 200, nil
	}

	if paused {
		cloud.Paused = true
		cloud.Pausing = true
		cloud.PausingMQTT = true
		cloud.auth = ""

		cloud.mqttMutex.Lock()
		if cloud.client != nil {
			cloud.client.Disconnect()
		}
		cloud.mqttMutex.Unlock()

		select {
		case cloud.sigDirty <- Entity{}:
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

func (cloud *Cloud) reconnect() {
	nretry := 0
	for !cloud.Pausing {
		status := cloud.authenticate()
		if status == http.StatusForbidden || status == http.StatusUnauthorized {
			cloud.Printf("Synchronization has been paused because of a login error.", http.StatusUnauthorized)
			cloud.SetPaused(true)
			break
		}
		if !isOk(status) && !cloud.Pausing {
			duration := retries[nretry]
			time.Sleep(duration)
			nretry++
			if nretry == len(retries) {
				nretry = len(retries) - 1
			}
			continue
		}
		break
	}
}

func (cloud *Cloud) sync() {

	nretry := 0

	retry := func() {

		if cloud.Pausing {
			return
		}

		duration := retries[nretry]
		log.Printf("[UP   ] Waiting %ds with REST before retry after error.", duration/time.Second)
		time.Sleep(duration)

		nretry++
		if nretry == len(retries) {
			nretry = len(retries) - 1
		}
	}

	////

	if cloud.auth == "" {
		cloud.reconnect()
	}

	////

	activeMQTT := false

	////

INITIAL_SYNC:
	for !cloud.Pausing {

		// cloud.setStatus(0, "Beginning initial sync ...")

		cloud.ResetStatus()
		cloud.sigDirty = make(chan Entity, 1)

		status := cloud.initialSync()
		if status == http.StatusForbidden || status == http.StatusUnauthorized {
			cloud.reconnect()
			continue
		}

		if !isOk(status) {
			retry()
			continue
		}

		// log.Printf("[UP   ] Initial sync completed with %d dirty.", len(cloud.Status))
		nretry = 0
		break
	}

	if !activeMQTT && !cloud.Pausing {
		activeMQTT = true
		go cloud.mqttSync()
	}

	for !cloud.Pausing {

		code, _ := cloud.persistentSync()

		if code <= 0 { // Network Error
			retry()
			continue
		}

		if !isOk(code) {
			retry()
			goto INITIAL_SYNC
		}

		nretry = 0
	}

	cloud.Pausing = false
	cloud.Printf("Synchronization paused.", 200)

	// log.Println("[UP   ] REST sync is now paused.")
	if !activeMQTT {
		cloud.PausingMQTT = false
		// log.Println("[UP   ] MQTT sync is now paused.")
	}
}
