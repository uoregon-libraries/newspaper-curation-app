package privilege

// This is our full, hard-coded list of valid privileges
var (
	// Titles
	ListTitles   = newPrivilege(RoleAny)
	ModifyTitles = newPrivilege(RoleTitleManager)

	// Add or delete MARC org codes
	ManageMOCs = newPrivilege(RoleMOCManager)

	// Workflow
	ViewMetadataWorkflow  = newPrivilege(RoleIssueCurator, RoleIssueReviewer, RoleIssueManager)
	EnterIssueMetadata    = newPrivilege(RoleIssueCurator, RoleIssueManager)
	ReviewIssueMetadata   = newPrivilege(RoleIssueReviewer, RoleIssueManager)
	ReviewOwnMetadata     = newPrivilege(RoleIssueManager)
	ReviewUnfixableIssues = newPrivilege(RoleIssueManager)

	// User management
	ListUsers   = newPrivilege(RoleUserManager)
	ModifyUsers = newPrivilege(RoleUserManager)

	// Uploaded issue viewing & queueing
	ViewUploadedIssues   = newPrivilege(RoleWorkflowManager)
	ModifyUploadedIssues = newPrivilege(RoleWorkflowManager)

	// Search for issues across all locations - this could really be more open,
	// but I don't see it being necessary for anybody but workflow managers at
	// the moment
	SearchIssues = newPrivilege(RoleWorkflowManager)

	// View batch status: anybody who can see the batch status page, regardless
	// of what they can/can't do there
	ViewBatchStatus = newPrivilege(RoleBatchReviewer, RoleBatchLoader)

	// View and manage batches awaiting QC (on staging)
	ViewQCReadyBatches    = newPrivilege(RoleBatchReviewer)
	ApproveQCReadyBatches = newPrivilege(RoleBatchReviewer)
	RejectQCReadyBatches  = newPrivilege(RoleBatchReviewer)

	// View and (eventually) manage batches that need to be loaded or fixed.
	// Staging/prod perms could be divided more granularly if it ever makes sense
	// to do so.
	ViewQCPassedBatches = newPrivilege(RoleBatchLoader)
	LoadBatches         = newPrivilege(RoleBatchLoader)
	PurgeBatches        = newPrivilege(RoleBatchLoader)
	ArchiveBatches      = newPrivilege(RoleBatchLoader)

	// Admins only
	ModifyValidatedLCCNs = newPrivilege()
	ListAuditLogs        = newPrivilege()
)

// A Privilege is a single action a user may be able to take
type Privilege struct {
	roles map[*Role]bool
}

// Privileges holds the full list of valid privileges for enumeration
var Privileges []*Privilege

// newPrivilege sets up a Privilege by name, adds the given roles to its list
// of roles allowed to use it, and keys the privilege lookup so it can be
// discovered by name
func newPrivilege(roles ...*Role) *Privilege {
	var priv = &Privilege{roles: make(map[*Role]bool)}
	for _, r := range roles {
		priv.roles[r] = true
	}
	Privileges = append(Privileges, priv)
	return priv
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
