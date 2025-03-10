package models

import (
	"errors"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// User identifies a person who has logged in via Apache's auth
type User struct {
	ID          int64  `sql:",primary"`
	Login       string `sql:",noupdate"`
	RolesString string `sql:"roles"`
	Guest       bool   `sql:"-"`
	IP          string `sql:"-"`
	Deactivated bool

	// realRoles is the actual list of roles a user has based on RolesString
	realRoles *privilege.RoleSet

	// implicitRoles are roles that this user's realRoles grant implicitly, such
	// as SysOps being given all roles
	implicitRoles *privilege.RoleSet
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
	return &User{Login: login, realRoles: privilege.NewRoleSet(), implicitRoles: privilege.NewRoleSet()}
}

// deserialize gets business-logic-friendly values from raw database data:
//
// - The comma-separated roles are looked up and turned into a usable [privilege.RoleSet]
// - Sysops and site managers get "implicit" roles set so they don't need to be manually granted all roles
func (u *User) deserialize() {
	u.realRoles = privilege.NewRoleSet()
	u.implicitRoles = privilege.NewRoleSet()

	// Figure out real roles based on the database string
	var roleStrings = strings.Split(u.RolesString, ",")
	for _, rs := range roleStrings {
		if rs == "" {
			continue
		}
		var role = privilege.FindRole(rs)
		if role == nil {
			logger.Errorf("User %s has an invalid role: %s", u.Login, role)
			continue
		}
		u.realRoles.Insert(role)
	}

	u.cleanRoles()
}

// cleanRoles removes unnecessary roles from the user. If a role is granted
// implicitly, it shouldn't be in the user's list separately.
func (u *User) cleanRoles() {
	// If you're a SysOp, your real roles list should just be SysOp, and your
	// implicit roles should get everything *but* SysOp.
	if u.realRoles.Contains(privilege.RoleSysOp) {
		u.realRoles = privilege.NewRoleSet(privilege.RoleSysOp)

		u.implicitRoles = privilege.AssignableRoles()
		u.implicitRoles.Remove(privilege.RoleSysOp)
	}

	// Similarly, if you aren't a SysOp but have a site manager role, we assign
	// that as your only real role and add all non-sysop and non-site-manager
	// roles to the implicit role list.
	//
	// NOTE: Order is important: the above SysOp check must happen first in case
	// somebody is assigned SysOp *and* Site Manager. The above code will "reset"
	// the user to just be SysOp, preventing this check from passing.
	if u.realRoles.Contains(privilege.RoleSiteManager) {
		u.realRoles = privilege.NewRoleSet(privilege.RoleSiteManager)

		u.implicitRoles = privilege.AssignableRoles()
		u.implicitRoles.Remove(privilege.RoleSysOp)
		u.implicitRoles.Remove(privilege.RoleSiteManager)
	}
}

// serialize turns business-logic-friendly values into raw DB-friendly data:
//
// - All roles (privilege.Role) have their names put into a deduped, sorted, comma-separated string
func (u *User) serialize() {
	u.cleanRoles()
	u.RolesString = strings.Join(u.realRoles.Names(), ",")
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

// InactiveUsers returns the list of all inactive users in the system. This
// should generally be used only for very niche tasks since we rarely want to
// display deactive users.
func InactiveUsers() ([]*User, error) {
	var users []*User
	var op = dbi.DB.Operation()
	op.Select("users", &User{}).Where("deactivated = ?", true).AllObjects(&users)

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
func FindUserByID(id int64) *User {
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

// GrantedRoles returns just the roles this user has explictitly been granted
func (u *User) GrantedRoles() *privilege.RoleSet {
	return u.realRoles.Clone()
}

// EffectiveRoles returns the calculated list of all roles this user has,
// whether explicitly granted, temporarily granted, or implicitly given.
func (u *User) EffectiveRoles() *privilege.RoleSet {
	return u.realRoles.Union(u.implicitRoles)
}

// PermittedTo returns true if this user has priv in his privilege list
func (u *User) PermittedTo(priv *privilege.Privilege) bool {
	// For extra safety, in case FindActiveUserWithLogin gets used incorrectly,
	// we make sure deactivated users aren't permitted to do *anything*.
	if u.Deactivated {
		return false
	}
	return priv.AllowedByAny(u.EffectiveRoles())
}

// isSysOp is true if this user has the SysOp role
func (u *User) isSysOp() bool {
	return u.EffectiveRoles().Contains(privilege.RoleSysOp)
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

// Grant adds the given role to this user's list of roles if it hasn't already
// been set
func (u *User) Grant(role *privilege.Role) {
	u.realRoles.Insert(role)
}

// Deny removes the given role from this user's roles list
func (u *User) Deny(role *privilege.Role) {
	u.realRoles.Remove(role)
}

// CanGrant returns true if the user can assign a given role to others.
// Assignment is allowed if a user has a role explicitly, or one of their roles
// grants the role implicitly.
func (u *User) CanGrant(role *privilege.Role) bool {
	// If this person can't modify users, they cannot grant anything
	if !u.PermittedTo(privilege.ModifyUsers) {
		return false
	}

	// We are very deliberate about the roles we allow here, only checking the
	// real roles and implicit roles, rather than using EffectiveRoles. This
	// ensures that if we ever add some form of temporary roles, a user can't
	// then reassign them permanently.
	var assignable = u.realRoles.Union(u.implicitRoles)
	return assignable.Contains(role)
}

// CanModifyUser tells us if u can modify the passed-in user
func (u *User) CanModifyUser(target *User) bool {
	// First and foremost, let's never let somebody modify themselves - too easy
	// to accidentally ruin things
	if u.ID == target.ID {
		return false
	}

	// Otherwise, SysOps can do anything to anybody
	if u.isSysOp() {
		return true
	}

	// Nobody can modify a SysOp but another SysOp
	if target.isSysOp() {
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
