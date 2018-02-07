package jobs

import (
	"config"
	"db"
	"derivatives/mets"
	"fmt"
	"path/filepath"
	"schema"
	"time"
)

// BuildMETS creates the final file needed for an issue to be able to be
// batched: the issue's METS XML file.  If the file is generated successfully,
// the issue's status will be updated and it can be included in the next
// available batch.
type BuildMETS struct {
	*IssueJob
	templatePath  string
	outputXMLPath string
	Title         *db.Title
}

// Process generates the METS XML file for the job's issue
func (job *BuildMETS) Process(c *config.Config) bool {
	job.Logger.Debugf("Starting build-mets job for issue id %d", job.DBIssue.ID)

	// Set up variables
	job.templatePath = c.XMLTemplatePath
	var dateEdStr = fmt.Sprintf("%s%02d", job.Issue.DateString(), job.Issue.Edition)
	job.outputXMLPath = filepath.Join(job.Issue.Location, dateEdStr+".xml")
	var err error
	job.Title, err = db.FindTitleByLCCN(job.DBIssue.LCCN)
	if err != nil {
		job.Logger.Errorf("Unable to look up title for issue id %d (LCCN %q): %s", job.DBIssue.ID, job.DBIssue.LCCN, err)
		return false
	}

	var ok = job.generateMETS()
	if !ok {
		return false
	}

	// The METS is generated, so failing to update the workflow doesn't actually
	// mean the operation failed; it just means we have to YELL about the problem
	err = job.updateIssueWorkflow()
	if err != nil {
		job.Logger.Criticalf("Unable to update issue (dbid %d) workflow post-METS: %s", job.DBIssue.ID, err)
	}
	return true
}

func (job *BuildMETS) generateMETS() (ok bool) {
	var err = mets.New(job.templatePath, job.outputXMLPath, job.DBIssue, job.Title, time.Now()).Transform()
	if err == nil {
		return true
	}
	job.Logger.Errorf("Unable to generate METS XML for issues %d: %s", job.DBIssue.ID, err)
	return false
}

func (job *BuildMETS) updateIssueWorkflow() error {
	job.DBIssue.WorkflowStep = schema.WSReadyForBatching
	return job.DBIssue.Save()
}
