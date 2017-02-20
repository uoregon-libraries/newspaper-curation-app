package user

// A Role defines a grouping of privileges
type Role struct {
	Name string
}

// Hard-coded list of roles
var (
	RoleAny           = newRole("-any-")
	RoleAdmin         = newRole("admin")
	RoleTitleManager  = newRole("title manager")
	RoleIssueCurator  = newRole("issue curator")
	RoleIssueReviewer = newRole("issue reviewer")
	RoleUserManager   = newRole("user manager")
	RoleMOCManager    = newRole("marc org code manager")
)

// roles is our internal map of string to Role object
var roles = make(map[string]*Role)

// newRole is internal as the list of roles shouldn't be modified by anything external
func newRole(name string) *Role {
	var r = &Role{name}
	roles[name] = r
	return r
}

// FindRole returns a role looked up by its name, or nil if no such role exists
func FindRole(name string) *Role {
	return roles[name]
}

// Privileges returns which privileges this role has based on our hard-coded lists
func (r *Role) Privileges() []*Privilege {
	var privs []*Privilege
	for _, priv := range privileges {
		if priv.AllowedBy(r) {
			privs = append(privs, priv)
		}
	}
	return privs
}
