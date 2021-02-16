package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"
	routing "github.com/julienschmidt/httprouter"
)

// GetCodecs implements GET /codecs
func GetCodecs(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	codecs := edge.GetCodecs()
	encoder := json.NewEncoder(resp)

	codec, err := codecs.Next()
	if err != nil && err.Error() != "EOF" {
		serveError(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.Write([]byte{'['})
	for codec != nil {
		encoder.Encode(codec)
		codec, _ = codecs.Next()
		if codec != nil {
			resp.Write([]byte{','})
		}
	}
	resp.Write([]byte{']'})
}

// PostCodec implements POST /codecs/{id}
func PostCodec(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	codecID := params.ByName("codec_id")
	var codec edge.ScriptCodec
	if err := unmarshalRequestBody(req, &codec); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	codec.ID = codecID

	if err := edge.PostCodec(&codec); err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("Codec upsert: %s", codec.ID)

	tools.SetRequestBody(req, &codec)

	resp.Write([]byte(codec.ID))
}

// DeleteCodec implements DELETE /codecs/{id}
func DeleteCodec(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	codecID := params.ByName("codec_id")
	if err := edge.DeleteCodec(codecID); err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("Codec deleted: %s", codecID)
}

// PostCodecs implements POST /codecs
func PostCodecs(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	var codec edge.ScriptCodec
	if err := unmarshalRequestBody(req, &codec); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := edge.PostCodec(&codec); err != nil {
		serveError(resp, err)
		return
	}

	log.Printf("Codec upsert: %s", codec.ID)

	tools.SetRequestBody(req, &codec)

	resp.Write([]byte(codec.ID))
}
