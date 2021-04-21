package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
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
	// tools.SendJSON(resp, tokenString)

	log.Printf("[ GHOLI ]: [%s]", tokenString)

	resp.Write([]byte(tokenString))
}

/*---------------------*/

func getAuthorizedUserID(req *http.Request) (string, error) {

	reqToken := ""

	if req.Header["Authorization"] != nil && len(req.Header["Authorization"][0]) > 0 {

		bearToken := req.Header["Authorization"][0]
		strArr := strings.Split(bearToken, " ")
		if len(strArr) == 2 {
			reqToken = strArr[1]
		}

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
	return token, nil
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

			reqIPStr, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				log.Printf("[ERR  ] Whitelist check failed: Invalid req.RemoteAddr: %q", req.RemoteAddr)
				return
			}
			reqIP := net.ParseIP(reqIPStr)
			if reqIP == nil {
				log.Printf("[ERR  ] Whitelist check failed: Invalid req.RemoteAddr: %q", req.RemoteAddr)
				return
			}
			ok, err := IsWazigateIP(reqIP)
			if err != nil {
				log.Printf("[ERR  ] Whitelist check failed for %q", req.RemoteAddr)
			}
			if ok {
				endpoint(resp, req, params)
				return
			}
		}

		/*-------------*/

		reqToken := ""
		bearToken := req.Header.Get("Authorization")

		if bearToken != "" && len(bearToken) > 0 {

			strArr := strings.Split(bearToken, " ")
			if len(strArr) == 2 {
				reqToken = strArr[1]
			}
		}

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

var wazigateSubnet *net.IPNet

// IsWazigateIP calls `docker network inspect wazigate` to get the wazigate subnet from docker.
// It checks if the ip is in the subnet, because all containers from docker are whitelisted for the API.
func IsWazigateIP(ip net.IP) (bool, error) {

	if wazigateSubnet == nil {

		//Get all container's IP list
		dockerJSONRaw, err := tools.SockGetReqest(dockerSocketAddress, "networks/wazigate")

		if err != nil {
			log.Printf("[ERR  ] Can not \"docker network inspect wazigate\": %v", err.Error())
			return false, err
		}

		var dockerNetwork struct {
			IPAM struct {
				Config []struct {
					Subnet  string `json:"Subnet"`
					Gateway string `json:"Gateway"`
				} `json:"Config"`
			} `json:"IPAM"`
		}

		err = json.Unmarshal(dockerJSONRaw, &dockerNetwork)
		if err != nil {
			log.Printf("[ERR  ] Can not unmarshal \"docker network inspect\" response (see below): %s", err.Error())
			log.Printf("$ docker network inspect\n%s", dockerJSONRaw)
			return false, err
		}

		if len(dockerNetwork.IPAM.Config) < 1 {
			log.Printf("[ERR  ] \"docker network inspect\" response (see below) has not .IPAM.Config: %s", err.Error())
			log.Printf("$ docker network inspect\n%s", dockerJSONRaw)
			return false, err
		}

		_, wazigateSubnet, err = net.ParseCIDR(dockerNetwork.IPAM.Config[0].Subnet)
		if err != nil {
			log.Printf("[ERR  ] Can not parse wazigate subnet from \"docker network inspect\" response (see below): %s", err.Error())
			log.Printf("$ docker network inspect\n%s", dockerJSONRaw)
			return false, err
		}
	}

	return wazigateSubnet.Contains(ip), nil
}

/*---------------------*/
