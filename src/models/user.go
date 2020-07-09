package models

import (
	"errors"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID          int    `sql:",primary"`
	Login       string `sql:",noupdate"`
	RolesString string `sql:"roles"`
	Guest       bool   `sql:"-"`
	IP          string `sql:"-"`
	Deactivated bool
	roles       []*privilege.Role
}

// EmptyUser gives us a way to avoid returning a nil *User while still being
// able to detect a user not being found.  Also lets us use any User functions
// without risking a panic.
var EmptyUser = &User{Login: "N/A", Guest: true}

// SystemUser is an "internal" object we can use to represent actions the
// system takes, comments from the processing as opposed to people, etc.
var SystemUser = &User{ID: -1, Login: "System Process"}

// NewUser returns an empty user with no roles or ID
func NewUser(login string) *User {
	return &User{Login: login}
}

func (u *User) deserialize() {
	u.buildRoles()
}

func (u *User) serialize() {
	u.RolesString = u.makeRoleString()
}

// ActiveUsers returns all users in the database who have the "active" status
func ActiveUsers() ([]*User, error) {
	var users []*User
	var op = dbi.DB.Operation()
	op.Select("users", &User{}).Where("deactivated = ?", false).AllObjects(&users)

	for _, u := range users {
		u.deserialize()
	}
	return users, op.Err()
}

// FindActiveUserWithLogin looks for a user whose login name is the given string.
// Deactivated users need not apply.
func FindActiveUserWithLogin(l string) *User {
	var users []*User
	var op = dbi.DB.Operation()
	op.Select("users", &User{}).Where("deactivated = ? AND login = ?", false, l).AllObjects(&users)
	if op.Err() != nil {
		logger.Errorf("Unable to query users: %s", op.Err())
	}

	if len(users) == 0 {
		return EmptyUser
	}

	users[0].deserialize()
	return users[0]
}

// FindUserByID looks up a user by the given ID - this can return inactive users
// since it's just using a database ID, so there's no possible ambiguity
func FindUserByID(id int) *User {
	// Hack to ensure we can just call standard finders to get the system user
	if id == SystemUser.ID {
		return SystemUser
	}

	var user = &User{}
	var op = dbi.DB.Operation()
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
func (u *User) Roles() []*privilege.Role {
	if len(u.roles) == 0 {
		u.buildRoles()
	}
	return u.roles
}

// HasRole returns true if the user has role in their list of roles
func (u *User) HasRole(role *privilege.Role) bool {
	for _, r := range u.Roles() {
		if r == role {
			return true
		}
	}

	return false
}

func (u *User) buildRoles() {
	var roleStrings = strings.Split(u.RolesString, ",")
	u.roles = make([]*privilege.Role, 0)
	for _, rs := range roleStrings {
		if rs == "" {
			continue
		}
		var role = privilege.FindRole(rs)
		if role == nil {
			logger.Errorf("User %s has an invalid role: %s", u.Login, role)
			continue
		}
		u.roles = append(u.roles, role)
	}
}

// PermittedTo returns true if this user has priv in his privilege list
func (u *User) PermittedTo(priv *privilege.Privilege) bool {
	// For extra safety, in case FindActiveUserWithLogin gets used incorrectly,
	// we make sure deactivated users aren't permitted to do *anything*.
	if u.Deactivated {
		return false
	}
	return priv.AllowedByAny(u.Roles())
}

// IsAdmin is true if this user has the admin role
func (u *User) IsAdmin() bool {
	return u.HasRole(privilege.RoleAdmin)
}

// Save stores the user's data to the database, rewriting the roles list
func (u *User) Save() error {
	if u.ID < 0 {
		return errors.New("cannot save system users")
	}
	if u.Guest {
		return errors.New("cannot save guest users")
	}

	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	u.serialize()
	op.Save("users", u)
	return op.Err()
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
func (u *User) Grant(role *privilege.Role) {
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
func (u *User) Deny(role *privilege.Role) {
	for i, r := range u.roles {
		if r == role {
			u.roles = append(u.roles[:i], u.roles[i+1:]...)
			u.serialize()
			return
		}
	}
}

// CanGrant returns true if this user can grant the given role to other users
func (u *User) CanGrant(role *privilege.Role) bool {
	// If this person can't modify users, they cannot grant anything
	if !u.PermittedTo(privilege.ModifyUsers) {
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

	return u.PermittedTo(privilege.ModifyUsers)
}

// Deactivate performs a soft-delete in order to remove a user from the visible
// users list without causing problems if the user is tied to metadata we need
// to reference later
func (u *User) Deactivate() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Exec("UPDATE users SET deactivated = ? WHERE id = ?", true, u.ID)
	return op.Err()
}
