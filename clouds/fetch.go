package clouds

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

type fetchInit struct {
	method  string
	headers map[string]string
	body    io.Reader
}

type fetchResponse struct {
	ok         bool
	status     int
	statusText string
	body       io.ReadCloser
}

func isOk(status int) bool {
	return status >= 200 && status < 300
}

func fetch(resource string, init fetchInit) fetchResponse {

	req, err := http.NewRequest(init.method, resource, init.body)
	if init.headers != nil {
		for header, value := range init.headers {
			req.Header.Set(header, value)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fetchResponse{
			ok:         false,
			status:     0,
			statusText: err.Error(),
		}
	}
	return fetchResponse{
		ok:         resp.StatusCode >= 200 && resp.StatusCode < 300,
		status:     resp.StatusCode,
		statusText: resp.Status,
		body:       resp.Body,
	}
}

func (resp *fetchResponse) json(data interface{}) error {
	if resp.body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(resp.body)
	err := decoder.Decode(data)
	resp.body.Close()
	return err
}

func (resp *fetchResponse) text() string {
	if resp.body == nil {
		return ""
	}
	data, err := ioutil.ReadAll(resp.body)
	resp.body.Close()
	if err != nil {
		return ""
	}
	return string(data)
}
