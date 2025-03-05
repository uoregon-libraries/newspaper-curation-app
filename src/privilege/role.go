package privilege

import (
	"regexp"
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

// AssignableRoles is a list of roles which can be assigned to a user
var AssignableRoles = []*Role{
	RoleSysOp,
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
}

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
	var c = cases.Title(language.AmericanEnglish)
	return c.String(strings.Replace(r.Name, "marc", "MARC", -1))
}
