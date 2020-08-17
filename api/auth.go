package api

import(
	"net/http"
    "fmt"
    "log"
    "encoding/json"
    "time"
    // "strings"
    
    "github.com/Waziup/wazigate-edge/edge"
    "github.com/Waziup/wazigate-edge/tools"

	routing "github.com/julienschmidt/httprouter"
	jwt "github.com/dgrijalva/jwt-go"
)

const tokenExpTimeMinutes = 10 // in minutes
var tokenSecret = []byte("Goooz") // Later we need to implement it to use some system vars + a random value

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
        log.Printf("[Err   ] GetToken: %s", err.Error())
        http.Error(resp, "bad request", http.StatusBadRequest)
		return
    }
    
    var inputUser edge.User

	err = json.Unmarshal(body, &inputUser)
	if err != nil {
        log.Printf("[Err   ] GetToken: %s", err.Error())
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
    }
    
    // log.Printf("Input User: %q", inputUser)
    
    validUser, err := edge.CheckUserCredentials( inputUser.Username, inputUser.Password);

    if err != nil{
        log.Printf("[Err   ] GetToken: %s", err.Error())
        http.Error(resp, "Invalid credentials", http.StatusUnauthorized)
		return
    }

    //Login success.

    tokenString, err := generateToken( validUser.ID);

    if err != nil {
        // resp.WriteHeader(http.StatusForbidden)
        // fmt.Fprint(resp, "Something went wrong!")
        log.Printf("[Err   ] GetToken: %s", err.Error())
        http.Error(resp, "Something went wrong", http.StatusForbidden)
        return
    }

    /*---------*/

    // Set Cookie, it is just an extra feature that makes the life easier on the UI part
    expiration  :=  time.Now().Add(time.Minute * tokenExpTimeMinutes)
    cookie      :=  http.Cookie{ 
            Name:       "Token",
            Value:      string( tokenString) ,
            Path:       "/",
            Expires:    expiration, 
            HttpOnly:   true,
            MaxAge:     60 * tokenExpTimeMinutes,
            // Secure:     true,
            SameSite:   SameSiteStrictMode,
        }
    http.SetCookie( resp, &cookie)

    /*---------*/

    // fmt.Fprint(resp, tokenString)
    tools.SendJSON(resp, tokenString)
}

/*---------------------*/

// GetRefereshToken implements POST /auth/retoken
// it takes a valid token and generate a new valid token
// it is used to keep the user logged in without asking for credentials every time the token gets expired
func GetRefereshToken(resp http.ResponseWriter, req *http.Request, params routing.Params) {

    userID, err := getAuthorizedUserID( req);

    if err != nil{
        log.Printf("[Err   ] GetRefereshToken: %s", err.Error())
        http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
        return
    }

    tokenString, err := generateToken( userID);

    if err != nil {
        log.Printf("[Err   ] GetRefereshToken: %s", err.Error())
        http.Error(resp, "Something went wrong", http.StatusForbidden)
        return
    }

    /*---------*/

    // Set Cookie, it is just an extra feature that makes the life easier on the UI part
    expiration  :=  time.Now().Add(time.Minute * tokenExpTimeMinutes)
    cookie      :=  http.Cookie{ 
            Name:       "Token",
            Value:      string( tokenString) ,
            Path:       "/",
            Expires:    expiration, 
            HttpOnly:   true,
            MaxAge:     60 * tokenExpTimeMinutes,
            // Secure:     true,
            SameSite:   SameSiteStrictMode,
        }
    http.SetCookie( resp, &cookie)

    /*---------*/

    // fmt.Fprint(resp, tokenString)
    tools.SendJSON(resp, tokenString)
}

/*---------------------*/

func getAuthorizedUserID( req *http.Request) (string, error){
    
    reqToken := ""

    if req.Header["Token"] != nil && len( req.Header["Token"][0]) > 0  {
        
        reqToken = req.Header["Token"][0]
    
    }else{

        c, err := req.Cookie("Token")
        if err != nil {
            log.Printf("[Err   ] Auth reading cookie: %s", err.Error())
        } else {
            reqToken = c.Value
        }
    }
    
    /*---------*/

    if len( reqToken) == 0 {
        
        return "", fmt.Errorf( "Not Authorized" )
    }

    token, err := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf( "There was an error" )
        }
        return tokenSecret, nil
    })

    if err != nil {
        return "", err
    }

    if !token.Valid {
        return "", fmt.Errorf( "Invalid Token" )
    }

    /*---------*/

    claims := token.Claims.(jwt.MapClaims)

    return claims["client"].(string), nil
}

/*---------------------*/

func generateToken( userID string) (string, error){

    token := jwt.New(jwt.SigningMethodHS256)

    claims := token.Claims.(jwt.MapClaims)

    claims["authorized"] = true
    claims["client"] = userID
    claims["exp"] = time.Now().Add(time.Minute * tokenExpTimeMinutes).Unix()

    tokenString, err := token.SignedString(tokenSecret)

    if err != nil {
        return "", err
    }

    return tokenString, nil
}

/*---------------------*/

// PostUserProfile implements POST /auth/profile
func PostUserProfile(resp http.ResponseWriter, req *http.Request, params routing.Params) {
	
    body, err := tools.ReadAll(req.Body)
	if err != nil {
        log.Printf("[Err   ] PostUserProfile: %s", err.Error())
        http.Error(resp, "bad request", http.StatusBadRequest)
		return
    }

    var inputProfile edge.User

	err = json.Unmarshal(body, &inputProfile)
	if err != nil {
        log.Printf("[Err   ] PostUserProfile: %s", err.Error())
		http.Error(resp, "bad request", http.StatusBadRequest)
		return
    }

    userID, err := getAuthorizedUserID( req);

    if err != nil{
        log.Printf("[Err   ] PostUserProfile: %s", err.Error())
        http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
        return
    }
    
    err = edge.UpdateUser( userID, &inputProfile)

    if err != nil{
        log.Printf("[Err   ] PostUserProfile: %s", err.Error())
        http.Error(resp, err.Error(), http.StatusUnauthorized)
        return
    }

   
    tools.SendJSON(resp, "Profile changes saved successfully.")
}

/*---------------------*/

func GetUserProfile(resp http.ResponseWriter, req *http.Request, params routing.Params) {

    userID, err := getAuthorizedUserID( req);

    if err != nil{
        log.Printf("[Err   ] GetUserProfile: %s", err.Error())
        http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
        return
    }

    user, err := edge.GetUser( userID)
	if err != nil {
        log.Printf("[Err   ] GetUserProfile: %s", err.Error())
        http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
    }

    user.Password = ""
    
    tools.SendJSON(resp, user);

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
            Name:   "Token",
            Path:       "/",
            HttpOnly:   true,
            // Secure:     true,
            SameSite:   SameSiteStrictMode,            
            MaxAge: -1}
    http.SetCookie(resp, &c)

    //TODO: Other actions that we may need to do
    tools.SendJSON(resp, "Logged out.")
}

/*---------------------*/

func IsAuthorized(endpoint routing.Handle) routing.Handle {
    return func(resp http.ResponseWriter, req *http.Request, params routing.Params) {

        reqToken := ""

        if req.Header["Token"] != nil && len( req.Header["Token"][0]) > 0  {
            reqToken = req.Header["Token"][0]
        
        }else{

            c, err := req.Cookie("Token")
            if err != nil {
                log.Printf("[Err   ] Auth reading cookie: %s", err.Error())
            } else {
                reqToken = c.Value
                log.Printf("Auth reading cookie: %q", reqToken)
            }
        }

        if len( reqToken) > 0 {

            token, err := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf( "There was an error" )
                }
                return tokenSecret, nil
            })

            if err != nil {
                log.Printf("[Err   ] Auth error: %s", err.Error())
                http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
                return
            }

            if token.Valid {
                endpoint(resp, req, params)
            }

        } else {
            log.Printf("[Err   ] Auth: the token is empty")
            // resp.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
            http.Error(resp, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
        }
    }
}

/*---------------------*/