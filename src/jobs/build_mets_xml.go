package jobs

import (
	"config"
	"db"
	"derivatives/mets"
	"path/filepath"
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
	job.templatePath = c.METSXMLTemplatePath
	job.outputXMLPath = filepath.Join(job.Issue.Location, job.Issue.DateEdition()+".xml")
	var err error
	job.Title, err = db.FindTitle("lccn = ?", job.DBIssue.LCCN)
	if err != nil {
		job.Logger.Errorf("Unable to look up title for issue id %d (LCCN %q): %s", job.DBIssue.ID, job.DBIssue.LCCN, err)
		return false
	}

	return job.generateMETS()
}

func (job *BuildMETS) generateMETS() (ok bool) {
	var err = mets.New(job.templatePath, job.outputXMLPath, job.DBIssue, job.Title, time.Now()).Transform()
	if err == nil {
		return true
	}
	job.Logger.Errorf("Unable to generate METS XML for issues %d: %s", job.DBIssue.ID, err)
	return false
}
