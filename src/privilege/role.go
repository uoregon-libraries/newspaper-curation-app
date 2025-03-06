package privilege

import (
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// A Role defines a grouping of privileges
type Role struct {
	Name string
	Desc string
}

var oneLineRegexp = regexp.MustCompile(`\s*\n\s*`)

// oneline turns a multi-line string into a single line by collapsing newlines
// surrounded by any amount of whitespace (space, tab, etc.) into a single
// ASCII space.
func oneline(s string) string {
	return oneLineRegexp.ReplaceAllString(s, " ")
}

// Hard-coded list of roles.
//
// NOTE: due to past-self's failings, you *must never* change a role's name
// (e.g., "issue curator" or "user manager") here! These names are stored in
// the database exactly as they're written here. Smart approach? Nope. Stuck
// with it? Yup.
var (
	RoleAny   = newRole("-any-", "N/A")
	RoleSysOp = newRole("sysop",
		`No restrictions. SysOps can do basically anything NCA allows. Users with this role can mistakenly break data. Only give this role to users who have access to run SQL directly against NCA's database.`)
	RoleSiteManager  = newRole("site manager", `Site managers can do nearly everything in the system, with very few restrictions. They are a combination of every other basic and management role.`)
	RoleTitleManager = newRole("title manager",
		`Has access to add and change newspaper titles, including the ability to
		view the sftp authorization information`)
	RoleIssueCurator = newRole("issue curator",
		`Can modify issue metadata and push issues to the review queue`)
	RoleIssueReviewer = newRole("issue reviewer", `Can review issues, rejecting or accepting a curator's metadata`)
	RoleIssueManager  = newRole("issue manager", `Privileged curator/review who can curate, review, approve
		their own issues' metadata, and process issues that are in the "unfixable error" state`)
	RoleUserManager = newRole("user manager",
		`Can add, edit, and deactivate users. User managers can assign any rights to
		others which have been assigned to them.`)
	RoleMOCManager      = newRole("marc org code manager", "Has access to add new MARC Org Codes")
	RoleWorkflowManager = newRole("workflow manager", "Can queue SFTP and scanned issues for processing")
	RoleBatchBuilder    = newRole("batch builder", "Can generate new batches on demand")
	RoleBatchReviewer   = newRole("batch reviewer",
		"Can view, reject, and approve batches which NCA has built but which are not yet in production.")
	RoleBatchLoader = newRole("batch loader", "Can load and purge batches on staging and production. This role states the user has these abilities, but in NCA this really just means they can view and flag batches as being loaded / ready for QC.")
)

// roles is our internal map of string to Role object
var roles = make(map[string]*Role)

// newRole is internal as the list of roles shouldn't be modified by anything external
func newRole(name, desc string) *Role {
	var r = &Role{Name: name, Desc: oneline(desc)}
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
	for _, priv := range Privileges {
		if priv.AllowedBy(r) {
			privs = append(privs, priv)
		}
	}
	return privs
}

// Title returns a slightly nicer string for display
func (r *Role) Title() string {
	// Uppercase all words, and also ensure "MARC" is fully capitalized
	var s = r.Name
	s = cases.Title(language.AmericanEnglish).String(s)
	return strings.Replace(s, "Marc", "MARC", -1)
}

type nothing struct{}

var sentinel = nothing{}

// RoleSet groups a bunch of roles in order to treat them as a single set-like
// entity: check for a role's existence, convert it to a string list, append or
// remove a role, etc.
type RoleSet struct {
	items map[*Role]nothing
}

// NewRoleSet returns a set containing the given roles
func NewRoleSet(roles ...*Role) *RoleSet {
	var rs = &RoleSet{items: make(map[*Role]nothing)}
	for _, r := range roles {
		rs.Insert(r)
	}

	return rs
}

// AssignableRoles returns a RoleSet containing all roles which can be assigned
// to a user
func AssignableRoles() *RoleSet {
	return NewRoleSet(
		RoleSysOp,
		RoleSiteManager,
		RoleTitleManager,
		RoleIssueCurator,
		RoleIssueReviewer,
		RoleIssueManager,
		RoleUserManager,
		RoleMOCManager,
		RoleWorkflowManager,
		RoleBatchBuilder,
		RoleBatchReviewer,
		RoleBatchLoader,
	)
}

// Contains returns true if the role is in our set
func (rs *RoleSet) Contains(r *Role) bool {
	var _, exists = rs.items[r]
	return exists
}

// Insert adds the given role to our set
func (rs *RoleSet) Insert(r *Role) {
	rs.items[r] = sentinel
}

// Clone returns a copy of rs
func (rs *RoleSet) Clone() *RoleSet {
	var newRS = NewRoleSet()
	for r := range rs.items {
		newRS.Insert(r)
	}

	return newRS
}

// Union returns a new set that combines rs and target
func (rs *RoleSet) Union(target *RoleSet) *RoleSet {
	var newRS = rs.Clone()
	for r := range target.items {
		newRS.Insert(r)
	}

	return newRS
}

// Remove takes the given role out of our set
func (rs *RoleSet) Remove(r *Role) {
	delete(rs.items, r)
}

// Empty removes all elements from the set
func (rs *RoleSet) Empty() {
	rs.items = make(map[*Role]nothing)
}

// Names returns a sorted slice of roles' names
func (rs *RoleSet) Names() []string {
	var roleNames []string
	for r := range rs.items {
		roleNames = append(roleNames, r.Name)
	}
	sort.Strings(roleNames)
	return roleNames
}

// List returns a logically sorted version of the underlying roles list. The
// sorted data prioritizes SysOp and site manager first, then sorts by name.
func (rs *RoleSet) List() []*Role {
	var roles []*Role
	for r := range rs.items {
		roles = append(roles, r)
	}

	sort.Slice(roles, func(i, j int) bool {
		if roles[i] == RoleSysOp {
			return true
		}
		if roles[i] == RoleSiteManager && roles[j] != RoleSysOp {
			return true
		}

		return roles[i].Name < roles[j].Name
	})

	return roles
}

// Len returns the number of roles in the set
func (rs *RoleSet) Len() int {
	return len(rs.items)
}
