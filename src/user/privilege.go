package user

// A Privilege is a single action a user may be able to take
type Privilege struct {
	Name  string
	roles map[*Role]bool
}

var privileges = make(map[string]*Privilege)

// init builds our giant, horrible list of hard-coded privileges
func init() {
	newPrivilege("list titles", RoleAny)
	newPrivilege("modify titles", RoleTitleManager)
	newPrivilege("manage mocs", RoleMOCManager)
	newPrivilege("modify validated lccns")
	newPrivilege("list issues", RoleIssueCurator, RoleIssueReviewer)
	newPrivilege("modify issues", RoleIssueCurator, RoleIssueReviewer)
	newPrivilege("list issue queues", RoleIssueCurator, RoleIssueReviewer)
	newPrivilege("modify review queue", RoleIssueCurator, RoleIssueReviewer)
	newPrivilege("review issues", RoleIssueReviewer)
	newPrivilege("list users", RoleUserManager)
	newPrivilege("modify users", RoleUserManager)
	newPrivilege("view title sftp", RoleTitleManager)
	newPrivilege("sftp report", RoleAny)
	newPrivilege("search workflow issues", RoleTitleManager)
	newPrivilege("queue sftp workflow", RoleWorkflowManager)
	newPrivilege("modify title sftp")
	newPrivilege("list audit logs")
}

// newPrivilege sets up a Privilege by name, adds the given roles to its list
// of roles allowed to use it, and keys the privilege lookup so it can be
// discovered by name
func newPrivilege(name string, roles ...*Role) *Privilege {
	var priv = &Privilege{Name: name, roles: make(map[*Role]bool)}
	for _, r := range roles {
		priv.roles[r] = true
	}
	privileges[name] = priv
	return priv
}

// FindPrivilege returns a Privilege by its name, or nil if none exists
func FindPrivilege(name string) *Privilege {
	return privileges[name]
}

// AllowedBy returns whether the privilege is allowed by the given role
func (p *Privilege) AllowedBy(r *Role) bool {
	if r == RoleAdmin || p.roles[RoleAny] {
		return true
	}

	return p.roles[r]
}

// AllowedByAny returns true if any of the roles can access this privilege
func (p *Privilege) AllowedByAny(roles []*Role) bool {
	// Special case: even if there are no roles, some privileges are still
	// allowed to be accessed
	if p.roles[RoleAny] {
		return true
	}

	for _, r := range roles {
		if p.AllowedBy(r) {
			return true
		}
	}

	return false
}
