package user

import (
	"github.com/Nerdmaster/magicsql"
	"log"
	"strings"
)

var DB *magicsql.DB

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID          int `sql:",primary"`
	Login       string
	RolesString string `sql:"roles"`
	Guest       bool `sql:"-"`
	roles       []*Role
	privileges  []*Privilege
}

var EmptyUser = &User{Login: "N/A", Guest: true}

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
		return EmptyUser
	}
	return users[0]
}

// Roles returns the split list of roles assigned to a user
func (u *User) Roles() []*Role {
	if len(u.roles) == 0 {
		u.buildRoles()
	}
	return u.roles
}

func (u *User) buildRoles() {
	var roleStrings = strings.Split(u.RolesString, ",")
	u.roles = make([]*Role, 0)
	for _, rs := range roleStrings {
		if rs == "" {
			continue
		}
		var role = FindRole(rs)
		if role == nil {
			log.Printf("ERROR: User %s has an invalid role: %s", u.Login, role)
			continue
		}
		u.roles = append(u.roles, role)
	}
}

// PermittedTo returns true if this user has pName in his privilege list
func (u *User) PermittedTo(pName string) bool {
	var priv = FindPrivilege(pName)
	if priv == nil {
		log.Printf("WARNING: Invalid privilege checked: %s", pName)
		return false
	}

	return priv.AllowedByAny(u.Roles())
}
