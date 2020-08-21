package edge

import (
	// "io"
	// "time"
	"log"
	"strings"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"golang.org/x/crypto/bcrypt"
)

// User represents a Wazigate user
type User struct {
	ID          string `json:"id" bson:"_id"`
	Name        string `json:"name" bson:"name"`
	Username    string `json:"username" bson:"username"`
	Password    string `json:"password" bson:"password"`
	NewPassword string `json:"newPassword"`

	// LastLogin time.Time `json:"lastlogin" bson:"lastlogin"`
}

/*--------------------------------*/

// MakeDefaultUser checks if there is no user registered in database,
// it makes a default user
// user: admin
// pass: loragateway
func MakeDefaultUser() error {

	usersCount, err := dbUsers.Find(nil).Count()

	if err != nil {
		return err
	}

	if usersCount > 0 {
		return nil
	}

	err = AddUser(&User{
		Name:     "Wazigate User",
		Username: "admin",
		Password: "loragateway",
	})

	if err != nil {
		log.Printf("[Err  ] Default user error: %s", err.Error())
	} else {
		log.Printf("[INFO ] Default user created")
	}

	return err
}

/*--------------------------------*/

// GetUser returns the Wazigate user
func GetUser(userID string) (User, error) {

	var user User
	err := dbUsers.Find(bson.M{
		"_id": userID,
	}).One(&user)

	return user, err
}

/*--------------------------------*/

// FindUserByUsername finds and returns the Wazigate user based on a given username
func FindUserByUsername(username string) (User, error) {

	var user User
	err := dbUsers.Find(bson.M{
		"username": strings.ToLower(username),
	}).One(&user)

	return user, err
}

/*--------------------------------*/

func UpdateUser(userID string, newProfileData *User) error {

	user, err := GetUser(userID)
	if err != nil {
		return CodeError{500, "error: " + err.Error()}
	}

	hashedPassword := []byte(user.Password) // default value

	//If user does not want to change the password, leave her/him alone ;)
	if len(newProfileData.NewPassword) > 0 {

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newProfileData.Password))
		if err != nil {
			// if err == bcrypt.ErrMismatchedHashAndPassword //Password does not match!
			log.Printf("[ERR  ] UpdatePassword: %s", err.Error())
			return CodeError{403, "Wrong password!"}
		}

		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(newProfileData.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[ERR  ] Password Generate: %s", err.Error())
			return CodeError{500, "Internal Error!"}
		}
	}

	_, err = dbUsers.Find(bson.M{
		"_id": userID,
	}).Apply(mgo.Change{

		Update: bson.M{
			"$set": bson.M{
				"password": string(hashedPassword),
				"name":     newProfileData.Name,
			},
		},
	}, &user)

	if err != nil {
		log.Printf("[ERR  ] Password Generate: %s", err.Error())
		return CodeError{500, "Database error"}
	}

	return nil
}

/*--------------------------------*/

// AddUser creates a new user.
func AddUser(user *User) error {

	// Check if the user already exist:
	_, err := FindUserByUsername(user.Username)
	if err == nil {
		return CodeError{409, "username already exists!"}

	} else if err != mgo.ErrNotFound {

		return CodeError{500, "error: " + err.Error()}
	}

	//TODO: hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERR  ] Password Generate: %s", err.Error())
	}

	if len(user.Username) > 0 { /*We may need to have a policy for username*/
		dbUsers.Insert(&User{
			ID:       bson.NewObjectId().Hex(),
			Name:     user.Name,
			Username: strings.ToLower(user.Username),
			Password: string(hashedPassword),
		})
	}

	return nil
}

/*--------------------------------*/

func CheckUserCredentials(username string, password string) (User, error) {

	user, err := FindUserByUsername(username)
	if err != nil {
		log.Printf("[ERR  ] login error: %s", err.Error())
		return user, CodeError{403, "Invalid login credentials!"}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		// if err == bcrypt.ErrMismatchedHashAndPassword //Password does not match!
		log.Printf("[ERR  ] login error: %s", err.Error())
		return user, CodeError{403, "Invalid login credentials!"}
	}

	// Success
	return user, nil
}

/*--------------------------------*/

// DeleteUser removes the giveb user.
func DeleteUser(userID string) error {

	_, err := dbUsers.RemoveAll(bson.M{
		"_id": userID,
	})

	return err
}
