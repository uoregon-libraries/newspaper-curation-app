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
}

// EmptyUser gives us a way to avoid returning a nil *User while still being
// able to detect a user not being found.  Also lets us use any User functions
// without risking a panic.
var EmptyUser = &User{Login: "N/A", Guest: true}

// New returns an empty user with no roles or ID
func New(login string) *User {
	return &User{Login: login}
}

func (u *User) deserialize() {
	u.buildRoles()
}

func (u *User) serialize() {
	u.RolesString = u.makeRoleString()
}

// All returns all users in the database
func All() ([]*User, error) {
	var users []*User
	var op = db.DB.Operation()
	op.Select("users", &User{}).AllObjects(&users)

	for _, u := range users {
		u.deserialize()
	}
	return users, op.Err()
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

	users[0].deserialize()
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
	user.deserialize()
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

// IsAdmin is true if this user has the admin role
func (u *User) IsAdmin() bool {
	for _, r := range u.Roles() {
		if r == RoleAdmin {
			return true
		}
	}

	return false
}

func (u *User) makeRoleString() string {
	var roleNames = make([]string, len(u.roles))
	for i, r := range u.roles {
		roleNames[i] = r.Name
	}

	return strings.Join(roleNames, ",")
}

// Grant adds the given role to this user's list of roles if it hasn't already
// been set
func (u *User) Grant(role *Role) {
	u.deserialize()
	for _, r := range u.roles {
		if r == role {
			return
		}
	}

	u.roles = append(u.roles, role)
	u.serialize()
}

// Deny removes the given role from this user's roles list
func (u *User) Deny(role *Role) {
	for i, r := range u.roles {
		if r == role {
			u.roles = append(u.roles[:i], u.roles[i+1:]...)
			u.serialize()
			return
		}
	}
}

// CanGrant returns true if this user can grant the given role to other users
func (u *User) CanGrant(role *Role) bool {
	// If this person can't modify users, they cannot grant anything
	if !u.PermittedTo(ModifyUsers) {
		return false
	}

	// Admins can grant anything
	if u.IsAdmin() {
		return true
	}

	// Users who aren't admins cannot grant roles they don't have
	for _, r := range u.Roles() {
		if role == r {
			return true
		}
	}

	return false
}

// CanModifyUser tells us if u can modify the passed-in user
func (u *User) CanModifyUser(user *User) bool {
	// First and foremost, let's never let somebody modify themselves - too easy
	// to accidentally ruin things
	if u.ID == user.ID {
		return false
	}

	// Otherwise, admins can do anything to anybody
	if u.IsAdmin() {
		return true
	}

	// Nobody can modify an admin but another admin
	if user.IsAdmin() {
		return false
	}

	return u.PermittedTo(ModifyUsers)
}
