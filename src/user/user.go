package user

import (
	"github.com/Nerdmaster/magicsql"
	"log"
)

var DB *magicsql.DB

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID          int `sql:",primary"`
	Login       string
	RolesString string `sql:"roles"`
}

// New returns an empty user with no roles or ID
func New(login string) *User {
	return &User{Login: login}
}

func FindByLogin(l string) *User {
	var users []*User
	var op = DB.Operation()
	op.Select("users", &User{}).Where("login = ?", l).AllObjects(&users)
	if op.Err() != nil {
		log.Printf("ERROR: Unable to query users: %s", op.Err())
	}

	if len(users) == 0 {
		return nil
	}
	return users[0]
}
