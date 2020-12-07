package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"
	routing "github.com/julienschmidt/httprouter"
)

func PostMessage(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var msg edge.Message
	if err := unmarshalRequestBody(req, &msg); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := edge.PostMessage(&msg); err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("[MSG  ] %s", msg.Title)
	for _, line := range strings.Split(msg.Text, "\n") {
		log.Printf("[MSG  ] > %s", line)
	}

	tools.SetRequestBody(req, &msg)

	resp.Write([]byte(msg.ID))
}

// GetMessages implements GET /devices
func GetMessages(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var query edge.MessagesQuery
	if err := query.Parse(req); err != "" {
		http.Error(resp, "bad request: "+err, http.StatusBadRequest)
		return
	}

	messages := edge.GetMessages(&query)
	encoder := json.NewEncoder(resp)

	msg, err := messages.Next()
	if err != nil && err.Error() != "EOF" {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'['})
	for msg != nil {
		encoder.Encode(msg)
		msg, _ = messages.Next()
		if msg != nil {
			resp.Write([]byte{','})
		}
	}
	resp.Write([]byte{']'})
}
