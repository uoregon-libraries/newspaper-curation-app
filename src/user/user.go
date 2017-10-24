package user

import (
	"github.com/Nerdmaster/magicsql"
	"logger"
	"strings"
)

// DB holds the persistent magicsql.DB object, and must be set externally in
// order for this package's database operations to succeed
var DB *magicsql.DB

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID          int `sql:",primary"`
	Login       string
	RolesString string `sql:"roles"`
	Guest       bool   `sql:"-"`
	IP          string `sql:"-"`
	roles       []*Role
	privileges  []*Privilege
}

// EmptyUser gives us a way to avoid returning a nil *User while still being
// able to detect a user not being found.  Also lets us use any User functions
// without risking a panic.
var EmptyUser = &User{Login: "N/A", Guest: true}

// New returns an empty user with no roles or ID
func New(login string) *User {
	return &User{Login: login}
}

// FindByLogin looks for a user whose login name is the given string.  The
// package variable DB must be set before this is called.
func FindByLogin(l string) *User {
	var users []*User
	var op = DB.Operation()
	op.Select("users", &User{}).Where("login = ?", l).AllObjects(&users)
	if op.Err() != nil {
		logger.Errorf("Unable to query users: %s", op.Err())
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
			logger.Errorf("User %s has an invalid role: %s", u.Login, role)
			continue
		}
		u.roles = append(u.roles, role)
	}
}

// PermittedTo returns true if this user has pName in his privilege list
func (u *User) PermittedTo(pName string) bool {
	var priv = FindPrivilege(pName)
	if priv == nil {
		logger.Warnf("Invalid privilege checked: %s", pName)
		return false
	}

	return priv.AllowedByAny(u.Roles())
}
