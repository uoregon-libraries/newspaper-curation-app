package batchhandler

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// CanValidation holds specific data (user and batch) related to a single HTTP
// request, allowing simpler validations and action generation.
type CanValidation struct {
	user *models.User
}

// Can sets up the DSL-like validation for the given user and batch
func Can(u *models.User) *CanValidation {
	return &CanValidation{u}
}

// View returns true if the user's privileges allow seeing details for b, based
// primarily on its status
func (c *CanValidation) View(b *Batch) bool {
	// Allow admins to view any batch. We have some statuses we don't normally
	// show, but there's no harm in allowing them to be displayed to admins if
	// they for some odd reason choose to hack up the URL.
	if c.user.HasRole(privilege.RoleAdmin) {
		return true
	}

	var has = c.user.PermittedTo
	switch b.Status {
	case models.BatchStatusStagingReady:
		return has(privilege.LoadBatches)
	case models.BatchStatusQCReady:
		return has(privilege.ViewQCReadyBatches)
	case models.BatchStatusQCFlagIssues:
		return has(privilege.RejectQCReadyBatches)
	case models.BatchStatusPassedQC:
		return has(privilege.LoadBatches)
	case models.BatchStatusLive:
		return has(privilege.ArchiveBatches)
	case models.BatchStatusDeleted, models.BatchStatusPending, models.BatchStatusLiveDone, models.BatchStatusLiveArchived:
		return false
	}

	logger.Errorf("Can view batch: Unhandled status %q", b.Status)
	return false
}

// Load is true if the user can load batches *and* b is in a loadable state
func (c *CanValidation) Load(b *Batch) bool {
	if !c.user.PermittedTo(privilege.LoadBatches) {
		return false
	}
	return b.Status == models.BatchStatusStagingReady || b.Status == models.BatchStatusPassedQC
}

// Archive is true if the user can archive batches and b is ready for archiving
func (c *CanValidation) Archive(b *Batch) bool {
	if !c.user.PermittedTo(privilege.ArchiveBatches) {
		return false
	}
	return b.ReadyForArchive()
}

// Approve is true if the user can approve batches and b is in need of approval
func (c *CanValidation) Approve(b *Batch) bool {
	if !c.user.PermittedTo(privilege.ApproveQCReadyBatches) {
		return false
	}

	return b.Status == models.BatchStatusQCReady
}

// Reject is true if the user can reject in-QC batches and b is ready for QC
func (c *CanValidation) Reject(b *Batch) bool {
	if !c.user.PermittedTo(privilege.RejectQCReadyBatches) {
		return false
	}

	return b.Status == models.BatchStatusQCReady
}

// Purge is true if the user is allowed to purge batches, and the batch is
// ready for issue flagging
func (c *CanValidation) Purge(b *Batch) bool {
	if !c.user.PermittedTo(privilege.PurgeBatches) {
		return false
	}

	return b.Status == models.BatchStatusQCFlagIssues
}

// FlagIssues is true if the user can reject in-QC batches and b is ready for
// issue flagging
func (c *CanValidation) FlagIssues(b *Batch) bool {
	if !c.user.PermittedTo(privilege.RejectQCReadyBatches) {
		return false
	}

	return b.Status == models.BatchStatusQCFlagIssues
}
