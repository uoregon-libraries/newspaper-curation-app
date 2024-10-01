package batchhandler

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// CanValidation holds specific data (user and batch) related to a single HTTP
// request, allowing simpler validations and action generation.
type CanValidation struct {
	user  *models.User
	batch *Batch
}

// Can sets up the DSL-like validation for the given user and batch
func Can(u *models.User, b *Batch) *CanValidation {
	return &CanValidation{user: u, batch: b}
}

// View returns true if the user's privileges allow seeing details for our
// batch, based primarily on its status
func (c *CanValidation) View() bool {
	// Allow admins to view any batch. We have some statuses we don't normally
	// show, but there's no harm in allowing them to be displayed to admins if
	// they for some odd reason choose to hack up the URL.
	if c.user.HasRole(privilege.RoleAdmin) {
		return true
	}

	return c.batch.Status != models.BatchStatusDeleted && c.batch.Status != models.BatchStatusPending
}

// Archive is true if the user can archive batches and batch is ready for archiving
func (c *CanValidation) Archive() bool {
	if !c.user.PermittedTo(privilege.ArchiveBatches) {
		return false
	}
	return c.batch.ReadyForArchive()
}

// Approve is true if the user can approve batches and batch is in need of approval
func (c *CanValidation) Approve() bool {
	if !c.user.PermittedTo(privilege.ApproveQCReadyBatches) {
		return false
	}

	return c.batch.Status == models.BatchStatusQCReady
}

// Reject is true if the user can reject in-QC batches and batch is ready for QC
func (c *CanValidation) Reject() bool {
	if !c.user.PermittedTo(privilege.RejectQCReadyBatches) {
		return false
	}

	return c.batch.Status == models.BatchStatusQCReady
}

// FlagIssues is true if the user can reject in-QC batches and batch is ready
// for issue flagging
func (c *CanValidation) FlagIssues() bool {
	if !c.user.PermittedTo(privilege.RejectQCReadyBatches) {
		return false
	}

	return c.batch.Status == models.BatchStatusQCFlagIssues
}
