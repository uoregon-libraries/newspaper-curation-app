package user

import (
	"db"
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
)

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID          int    `sql:",primary"`
	Login       string `sql:",noupdate"`
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

// FindByLogin looks for a user whose login name is the given string
func FindByLogin(l string) *User {
	var users []*User
	var op = db.DB.Operation()
	op.Select("users", &User{}).Where("login = ?", l).AllObjects(&users)
	if op.Err() != nil {
		logger.Errorf("Unable to query users: %s", op.Err())
	}

	if len(users) == 0 {
		return EmptyUser
	}
	return users[0]
}

// FindByID looks up a user by the given ID
func FindByID(id int) *User {
	var user = &User{}
	var op = db.DB.Operation()
	var ok = op.Select("users", &User{}).Where("id = ?", id).First(user)
	if op.Err() != nil {
		logger.Errorf("Unable to query users: %s", op.Err())
	}

	if !ok {
		return EmptyUser
	}
	return user
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

// PermittedTo returns true if this user has priv in his privilege list
func (u *User) PermittedTo(priv *Privilege) bool {
	return priv.AllowedByAny(u.Roles())
}
