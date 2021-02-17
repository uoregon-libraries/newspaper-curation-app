package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Batch extends a models.Batch with functionality for fixing, re-queueing, etc.
type Batch struct {
	db     *models.Batch
	Issues IssueList
}

// FindBatch looks up a batch in the database, then pulls all its issues
func FindBatch(id int) (*Batch, error) {
	var batch, err = models.FindBatch(id)
	if err != nil {
		return nil, fmt.Errorf("database error: %s", err.Error())
	}

	if batch == nil {
		return nil, fmt.Errorf("id not in database")
	}

	var b = &Batch{db: batch}
	err = b.loadIssues()
	if err != nil {
		return nil, fmt.Errorf("error loading batch issues: %s", err)
	}

	return b, nil
}

// Fail deletes all batch files from disk - these are all bagit files or
// hard-links, so we can easily replace everything removed.  The batch location
// is cleared, and its status is then set to "failed_qc" so it's clear it needs
// to be reprocessed in some way.
func (b *Batch) Fail() error {
	if !fileutil.IsDir(b.db.Location) {
		return fmt.Errorf("removing batch files: %q does not exist", b.db.Location)
	}

	var err = os.RemoveAll(b.db.Location)
	if err != nil {
		return fmt.Errorf("removing batch files: %s", err)
	}

	b.db.Status = models.BatchStatusFailedQC
	b.db.Location = ""
	err = b.db.Save()
	if err != nil {
		return fmt.Errorf("updating database status: %s", err)
	}

	return nil
}

func (b *Batch) loadIssues() error {
	var dbIssues, err = b.db.Issues()

	b.Issues = make(IssueList, len(dbIssues))
	for i, dbi := range dbIssues {
		b.Issues[i] = &Issue{db: dbi}
	}
	b.Issues.SortByKey()
	return err
}

// Issue extends a models.Issue with functionality for pulling the issue off a
// batch, rejecting it (which, post-batch, means more than just rejection when
// it's in the metadata review phase), etc.
type Issue struct {
	db *models.Issue
}

// FindIssue looks up the issue in the database by the given id and wraps the
// model struct inside our local struct
func FindIssue(id int) (*Issue, error) {
	var issue, err = models.FindIssue(id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if issue == nil {
		return nil, fmt.Errorf("issue %d wasn't found in the database", id)
	}

	return &Issue{db: issue}, nil
}

// removeMETS attempts to remove the METS XML file, returning an error if any
// problems occur *except* the file already being gone, since that may be a
// sign this was called previously, or somebody had to handle it manually.  We
// verify sanity first by checking that the issue's directory does indeed
// exist (and is a directory as opposed to a file).
func (i *Issue) removeMETS() error {
	var si, err = i.db.SchemaIssue()
	if err != nil {
		return fmt.Errorf("unable to get a schema.Issue from the models.Issue: %s", err)
	}

	// Make sure the dir exists, since lack of a mets file isn't a failure
	if !fileutil.IsDir(i.db.Location) {
		return fmt.Errorf("issue directory %q does not exist; aborting", i.db.Location)
	}

	err = os.Remove(si.METSFile())
	if !os.IsNotExist(err) && err != nil {
		return fmt.Errorf("unable to remove METS file: %s", err)
	}

	return nil
}

type invaliationType byte

const (
	iTypeNil invaliationType = iota
	iTypeError
	iTypeReject
)

// invalidateFromBatch handles the common logic necessary when an issue is
// rejected from a batch to get a fix, or needs to be pulled from NCA entirely.
// The iType determines which lower-level function to use for this.
func (i *Issue) invalidateFromBatch(typ invaliationType, msg string) error {
	var err error

	i.db.BatchID = 0

	switch typ {
	case iTypeError:
		err = i.db.ReportError(models.SystemUser.ID, msg)
	case iTypeReject:
		err = i.db.RejectMetadata(models.SystemUser.ID, msg)
	default:
		err = fmt.Errorf("unknown invalidation type")
	}

	if err != nil {
		return fmt.Errorf("unable to report/reject issue: %s", err)
	}

	err = i.removeMETS()
	if err != nil {
		return fmt.Errorf("unable to remove METS file: %s", err)
	}

	return nil
}

// IssueList is a simple wrapper around a slice of issues to add functionality
// for easier sorting
type IssueList []*Issue

// SortByKey modifies the IssueList in place so they're sorted alphabetically
// by issue key
func (list IssueList) SortByKey() {
	sort.Slice(list, func(i, j int) bool {
		var kA, kB = list[i].db.Key(), list[j].db.Key()
		if kA != kB {
			return kA < kB
		}

		return list[i].db.ID < list[j].db.ID
	})
}
