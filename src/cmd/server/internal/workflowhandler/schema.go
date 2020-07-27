package workflowhandler

import (
	"encoding/base64"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// encodedErrors creates a base64 alert for validation errors to be displayed
// after attempting to queue an issue or approve an issue
func encodedErrors(action string, errors []apperr.Error) string {
	var errorstr string
	for _, err := range errors {
		errorstr += "<li>" + err.Message() + "</li>"
	}
	var alertMsg = "Cannot " + action + " this issue:<ul>" + errorstr + "</ul>"
	var encodedAlert = "base64" + base64.StdEncoding.EncodeToString([]byte(alertMsg))
	return encodedAlert
}

// Issue wraps the DB issue, and decorates it with display-friendly functions
// and dataentry-specific errors
type Issue struct {
	*models.Issue
	MetadataAuthorLogin string

	si *schema.Issue

	validationErrors []apperr.Error
}

func wrapDBIssue(dbIssue *models.Issue) *Issue {
	// For workflow presentation, we don't really care if the issue isn't valid
	// so long as we can show its raw data to the user
	var si, _ = dbIssue.SchemaIssue()
	return &Issue{Issue: dbIssue, si: si, MetadataAuthorLogin: models.FindUserByID(dbIssue.MetadataEntryUserID).Login}
}

func wrapDBIssues(dbIssues []*models.Issue) []*Issue {
	var list []*Issue
	for _, dbIssue := range dbIssues {
		var i = wrapDBIssue(dbIssue)
		if i == nil {
			return nil
		}
		list = append(list, i)
	}

	return list
}

// Title returns the issue's title's name
func (i *Issue) Title() string {
	return i.si.Title.Name
}

// LCCN returns the issue's title's LCCN
func (i *Issue) LCCN() string {
	return i.si.Title.LCCN
}

// Date returns the issue's raw date string
func (i *Issue) Date() string {
	return i.si.RawDate
}

// JP2Files aggregates all the JP2s that exist in this issue's directory
func (i *Issue) JP2Files() []string {
	var list []string

	if len(i.si.Files) == 0 {
		i.si.FindFiles()
	}

	for _, f := range i.si.Files {
		if strings.ToUpper(filepath.Ext(f.Location)) == ".JP2" {
			list = append(list, f.Location)
		}
	}

	return list
}

// TaskDescription returns a human-friendly explanation of the current place
// this issue is within the workflow
func (i *Issue) TaskDescription() string {
	switch i.WorkflowStep {
	case schema.WSUnfixableMetadataError:
		return "Unfixable metadata error reported"

	case schema.WSAwaitingProcessing:
		return "Not yet entered into the workflow"

	case schema.WSAwaitingPageReview:
		return "Ready for page review (renaming files / validating raw PDFs / TIFFs)"

	case schema.WSReadyForMetadataEntry:
		return "Awaiting metadata entry / page numbering"

	case schema.WSAwaitingMetadataReview:
		return "Awaiting review (metadata and page numbers)"

	case schema.WSReadyForBatching:
		return "Ready to be built in a batch and loaded"

	default:
		logger.Criticalf("Invalid workflow step for issue %d: %q", i.ID, i.WorkflowStepString)
		return "UNKNOWN!"
	}
}

// WorkflowExpiration returns the date and time of "workflow expiration": when
// this item is no longer claimed by the workflow owner
func (i *Issue) WorkflowExpiration() string {
	return i.WorkflowOwnerExpiresAt.Format("2006-01-02 at 15:04")
}

// IsOwned returns true if the owner ID is nonzero *and* the workflow owner
// expiration time has not passed
func (i *Issue) IsOwned() bool {
	return i.WorkflowOwnerID != 0 && time.Now().Before(i.WorkflowOwnerExpiresAt)
}

// Path returns the path for any basic actions on this issue
func (i *Issue) Path(actionPath string) string {
	return path.Join(basePath, strconv.Itoa(i.ID), actionPath)
}

// ValidateMetadata checks all fields for validity and sets up
// i.validationErrors to describe anything wrong
func (i *Issue) ValidateMetadata() {
	i.validationErrors = nil
	var addError = func(err apperr.Error) { i.validationErrors = append(i.validationErrors, err) }
	var validDate = func(dtString, fieldName string) {
		var dtLayout = "2006-01-02"
		var dt, err = time.Parse(dtLayout, dtString)
		if err != nil || dt.Format(dtLayout) != dtString {
			addError(apperr.Errorf("%q is not a valid date", fieldName))
		}
	}
	var notBlank = func(val, fieldName string) {
		if val == "" {
			addError(apperr.Errorf("%q cannot be blank", fieldName))
		}
	}

	validDate(i.Issue.Date, "Issue Date")
	validDate(i.DateAsLabeled, "Date As Labeled")
	notBlank(i.Volume, "Volume Number")
	notBlank(i.Issue.Issue, "Issue Number")
	if i.Edition == 0 {
		addError(apperr.New(`"Edition Number" cannot be zero`))
	}

	var numLabels = len(i.PageLabels)
	var numFiles = len(i.JP2Files())
	if numLabels < numFiles {
		addError(apperr.New("Page labeling isn't completed"))
	}
	if numLabels > numFiles {
		logger.Errorf("There are %d page labels, but only %d JP2 files!", numLabels, numFiles)
		for _, jp2 := range i.JP2Files() {
			logger.Debugf("  - %q", jp2)
		}

		addError(apperr.New("Unknown error in page labeling; contact support or try again"))
	}

	// Generate a new schema issue to test for dupes
	var err error
	i.si, err = i.Issue.SchemaIssue()
	if err != nil {
		logger.Criticalf("Unable to recreate schema.Issue for issue id %d: %s", i.ID, err)
		addError(apperr.New("Unknown error checking issue validity; contact support or try again"))
		return
	}

	// Check dupes on the schema issue, then pull those errors onto our validations
	i.si.CheckDupes(watcher.Scanner.Lookup)
	for _, err := range i.si.Errors {
		addError(err)
	}
}

// Errors returns validation errors
func (i *Issue) Errors() []apperr.Error {
	return i.validationErrors
}
