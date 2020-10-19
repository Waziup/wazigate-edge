package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"time"

	// "strings"

	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/tools"

	jwt "github.com/dgrijalva/jwt-go"
	routing "github.com/julienschmidt/httprouter"
)

const tokenExpTimeMinutes = 10 // in minutes

/*---------------------*/

const (
	SameSiteDefaultMode http.SameSite = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

/*---------------------*/

// GetToken implements POST /auth/token
func GetToken(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		log.Printf("[ERR  ] GetToken: %s", err.Error())
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}

	var inputUser edge.User

	err = json.Unmarshal(body, &inputUser)
	if err != nil {
		log.Printf("[ERR  ] GetToken: %s", err.Error())
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}

	// log.Printf("Input User: %q", inputUser)

	validUser, err := edge.CheckUserCredentials(inputUser.Username, inputUser.Password)

	if err != nil {
		log.Printf("[ERR  ] GetToken: %s", err.Error())
		http.Error(resp, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	//Login success.

	tokenString, err := generateToken(validUser.ID)

	if err != nil {
		// resp.WriteHeader(http.StatusForbidden)
		// fmt.Fprint(resp, "Something went wrong!")
		log.Printf("[ERR  ] GetToken: %s", err.Error())
		http.Error(resp, "Something went wrong", http.StatusForbidden)
		return
	}

	/*---------*/

	// Set Cookie, it is just an extra feature that makes the life easier on the UI part
	expiration := time.Now().Add(time.Minute * tokenExpTimeMinutes)
	cookie := http.Cookie{
		Name:     "Token",
		Value:    string(tokenString),
		Path:     "/",
		Expires:  expiration,
		HttpOnly: true,
		MaxAge:   60 * tokenExpTimeMinutes,
		// Secure:     true,
		SameSite: SameSiteStrictMode,
	}
	http.SetCookie(resp, &cookie)

	/*---------*/

	// fmt.Fprint(resp, tokenString)
	tools.SendJSON(resp, tokenString)
}

/*---------------------*/

// GetRefereshToken implements POST /auth/retoken
// it takes a valid token and generate a new valid token
// it is used to keep the user logged in without asking for credentials every time the token gets expired
func GetRefereshToken(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	userID, err := getAuthorizedUserID(req)

	if err != nil {
		log.Printf("[ERR  ] GetRefereshToken: %s", err.Error())
		http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	tokenString, err := generateToken(userID)

	if err != nil {
		log.Printf("[ERR  ] GetRefereshToken: %s", err.Error())
		http.Error(resp, "Something went wrong", http.StatusForbidden)
		return
	}

	/*---------*/

	// Set Cookie, it is just an extra feature that makes the life easier on the UI part
	expiration := time.Now().Add(time.Minute * tokenExpTimeMinutes)
	cookie := http.Cookie{
		Name:     "Token",
		Value:    string(tokenString),
		Path:     "/",
		Expires:  expiration,
		HttpOnly: true,
		MaxAge:   60 * tokenExpTimeMinutes,
		// Secure:     true,
		SameSite: SameSiteStrictMode,
	}
	http.SetCookie(resp, &cookie)

	/*---------*/

	// fmt.Fprint(resp, tokenString)
	tools.SendJSON(resp, tokenString)
}

/*---------------------*/

func getAuthorizedUserID(req *http.Request) (string, error) {

	reqToken := ""

	if req.Header["Token"] != nil && len(req.Header["Token"][0]) > 0 {

		reqToken = req.Header["Token"][0]

	} else {

		c, err := req.Cookie("Token")
		if err != nil {
			log.Printf("[ERR  ] Auth reading cookie: %s", err.Error())
		} else {
			reqToken = c.Value
		}
	}

	/*---------*/

	if len(reqToken) == 0 {

		return "", fmt.Errorf("Not Authorized")
	}

	token, err := CheckToken(reqToken)
	if err != nil {
		return "", err
	}

	/*---------*/

	claims := token.Claims.(jwt.MapClaims)

	return claims["client"].(string), nil
}

func CheckToken(t string) (*jwt.Token, error) {
	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("There was an error")
		}
		return getSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("Invalid Token")
	}
	return nil, nil
}

/*---------------------*/

func generateToken(userID string) (string, error) {

	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)

	claims["authorized"] = true
	claims["client"] = userID
	claims["exp"] = time.Now().Add(time.Minute * tokenExpTimeMinutes).Unix()

	tokenString, err := token.SignedString(getSecret())

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

/*---------------------*/

var tokenSecret []byte

func getSecret() []byte {

	if tokenSecret != nil {
		return tokenSecret
	}

	secretMinSize := 40
	someBytes := make([]byte, secretMinSize)
	rand.Seed(time.Now().UTC().UnixNano())

	for i := 0; i < secretMinSize; i++ {
		someBytes[i] = byte(rand.Intn(255))
	}

	tokenSecret = []byte(base64.StdEncoding.EncodeToString(someBytes))
	return tokenSecret
}

/*---------------------*/

// PostUserProfile implements POST /auth/profile
func PostUserProfile(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		log.Printf("[ERR  ] PostUserProfile: %s", err.Error())
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}

	var inputProfile edge.User

	err = json.Unmarshal(body, &inputProfile)
	if err != nil {
		log.Printf("[ERR  ] PostUserProfile: %s", err.Error())
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
	}

	userID, err := getAuthorizedUserID(req)

	if err != nil {
		log.Printf("[ERR  ] PostUserProfile: %s", err.Error())
		http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	err = edge.UpdateUser(userID, &inputProfile)

	if err != nil {
		log.Printf("[ERR  ] PostUserProfile: %s", err.Error())
		http.Error(resp, err.Error(), http.StatusUnauthorized)
		return
	}

	tools.SendJSON(resp, "Profile changes saved successfully.")
}

/*---------------------*/

// GetUserProfile implements GET /auth/profile
func GetUserProfile(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	userID, err := getAuthorizedUserID(req)

	if err != nil {
		log.Printf("[ERR  ] GetUserProfile: %s", err.Error())
		http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	user, err := edge.GetUser(userID)
	if err != nil {
		log.Printf("[ERR  ] GetUserProfile: %s", err.Error())
		http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	user.Password = ""

	tools.SendJSON(resp, user)

}

/*---------------------*/

// GetPermissions implements GET /auth/permissions
func GetPermissions(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	// TODO: implement
	tools.SendJSON(resp, "GetPermissions()")
}

/*---------------------*/

// Logout implements GET /auth/logout
func Logout(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	c := http.Cookie{
		Name:     "Token",
		Path:     "/",
		HttpOnly: true,
		// Secure:     true,
		SameSite: SameSiteStrictMode,
		MaxAge:   -1}
	http.SetCookie(resp, &c)

	//TODO: Other actions that we may need to do
	tools.SendJSON(resp, "Logged out.")
}

/*---------------------*/

// IsAuthorized checks if the given request is valid for the API call
func IsAuthorized(endpoint routing.Handle, checkIPWhiteList bool) routing.Handle {

	return func(resp http.ResponseWriter, req *http.Request, params routing.Params) {

		// mqtt connections are already logged in & authorized
		// the header is set by the edge and not by the request!
		if req.Header.Get("X-Proto") == "mqtt" {
			endpoint(resp, req, params)
			return
		}

		if checkIPWhiteList {

			ip, _, err := net.SplitHostPort(req.RemoteAddr)

			// log.Printf("\n\n\tINFO: %q", req.RemoteAddr)
			// log.Printf("\n\n\tINFO: %q, %q, %q", ip, port, err)

			if err != nil {

				log.Printf("[ERR  ] White list check: %s", err.Error())

			} else {

				ok, err := IsIPAddrInCurrentDockerNetwork(string(ip))

				// log.Printf("\n\n\tINFO OK: %q, %q", ok, err)

				// The IP is authorized
				if ok {
					endpoint(resp, req, params)
					return
				}
				if err != nil {
					log.Printf("[ERR  ] White list check: %s", err.Error())
				}
			}
		}

		/*-------------*/

		reqToken := req.Header.Get("Token")

		if reqToken == "" {
			c, err := req.Cookie("Token")
			if err != nil {
				log.Printf("[ERR  ] Auth reading cookie: %s", err.Error())
			} else {
				reqToken = c.Value
				// log.Printf("Auth reading cookie: %q", reqToken)
			}
		}

		if reqToken != "" {

			token, err := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("There was an error")
				}
				return getSecret(), nil
			})

			if err != nil {
				log.Printf("[ERR  ] Auth error: %s", err.Error())
				http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			if token.Valid {
				endpoint(resp, req, params)
			}

		} else {
			log.Printf("[ERR  ] Auth: the token is empty")
			// resp.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

/*---------------------*/

var listOfWhiteIPs map[string]interface{} = map[string]interface{}{}

// IsIPAddrInCurrentDockerNetwork the name is selfdescribe ;)
func IsIPAddrInCurrentDockerNetwork(inputIP string) (bool, error) {

	// BUG: there might be some issues when a call is received bu not all containers are loaded
	// So their IP will be out of the white list!!!

	if len(listOfWhiteIPs) == 0 {

		//Get all container's IP list
		dockerJSONRaw, err := tools.SockGetReqest(dockerSocketAddress, "networks/wazigate")

		if err != nil {
			// log.Printf("[ERR  ] Check IP White list: %s", err.Error())
			return false, err
		}

		var dockerJSON struct {
			Containers map[string]interface{} `json:"Containers"`
		}

		if err := json.Unmarshal(dockerJSONRaw, &dockerJSON); err != nil {
			return false, err
		}

		for _, value := range dockerJSON.Containers {

			ipStr := value.(map[string]interface{})["IPv4Address"].(string)
			re := regexp.MustCompile(`([0-9]+\.){3}([0-9]+)`)
			match := re.FindStringSubmatch(ipStr)
			listOfWhiteIPs[match[0]] = true

			// log.Println(match[0])
		}
	}

	// log.Printf("\n\nWhite LIST: %q", listOfWhiteIPs)

	// Check if the given IP address is in the list of "wazigate" docker network
	// which we consider it a white list
	_, ok := listOfWhiteIPs[inputIP]
	return ok, nil
}

/*---------------------*/
