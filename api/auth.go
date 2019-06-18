package api

import(
	"net/http"
	"fmt"

	routing "github.com/julienschmidt/httprouter"
)


// GetToken implements GET /auth/token
func GetToken(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// TODO: implement
	fmt.Fprint(resp, "GetToken()")
}

// GetPermissions implements GET /auth/permissions
func GetPermissions(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// TODO: implement
	fmt.Fprint(resp, "GetPermissions()")
}
