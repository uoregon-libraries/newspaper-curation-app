package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/marc"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// testMOCs are the MARC org codes the test data (see test/README.md) uses
var testMOCs = []models.MOC{
	{Code: "oru", Name: "University of Oregon Libraries; Eugene, OR"},
	{Code: "hoodriverlibrary", Name: "Hood River County Library District; Hood River, OR"},
}

// seedCmd loads what a fresh database needs before the test data works: MARC
// org codes, users, and titles. Titles are loaded into both ONIs as well, so
// both ONIs must be up and done setting up Solr.
type seedCmd struct {
	Sources string `long:"sources" default:"test/sources" description:"directory holding the test data's marc-xml dir"`
}

// Execute implements flags.Commander
func (c *seedCmd) Execute([]string) error {
	var err = connectDB()
	if err == nil {
		err = seedMOCs()
	}
	if err == nil {
		err = seedUsers()
	}
	if err == nil {
		err = loadTitles(filepath.Join(c.Sources, "marc-xml"))
	}
	return err
}

func seedMOCs() error {
	for _, moc := range testMOCs {
		var existing, err = models.FindMOCByCode(moc.Code)
		if err != nil {
			return fmt.Errorf("looking up MOC %q: %w", moc.Code, err)
		}
		if existing != nil {
			continue
		}

		logger.Infof("Adding MOC %q", moc.Code)
		var m = moc
		err = m.Save()
		if err != nil {
			return fmt.Errorf("adding MOC %q: %w", moc.Code, err)
		}
	}
	return nil
}

// seedUsers deletes all users, then creates a user for each assignable role,
// named for the role without spaces (e.g., "sitemanager")
func seedUsers() error {
	var op = dbi.DB.Operation()
	op.Exec("DELETE FROM users")
	if op.Err() != nil {
		return fmt.Errorf("deleting users: %w", op.Err())
	}

	for _, role := range privilege.AssignableRoles().List() {
		var u = models.NewUser(strings.ToLower(strings.ReplaceAll(role.Name, " ", "")))
		u.Grant(role)
		logger.Debugf("Creating user %q with role %q", u.Login, role.Name)
		var err = u.Save()
		if err != nil {
			return fmt.Errorf("creating user for role %q: %w", role.Name, err)
		}
	}
	return nil
}

// loadTitles creates or updates a title in NCA for each MARC record in dir,
// and loads the record into both ONIs
func loadTitles(dir string) error {
	if !fileutil.IsDir(dir) {
		logger.Warnf("No MARC records directory (%q): not loading any titles", dir)
		return nil
	}
	var files, err = findFiles(dir, ".xml", ".XML", ".mrk", ".MRK")
	if err != nil {
		return err
	}

	var staging, prod *openoni.RPC
	staging, err = openoni.New(conf.StagingAgentConnection)
	if err != nil {
		return fmt.Errorf("staging agent connection %q: %w", conf.StagingAgentConnection, err)
	}
	prod, err = openoni.New(conf.ProductionAgentConnection)
	if err != nil {
		return fmt.Errorf("production agent connection %q: %w", conf.ProductionAgentConnection, err)
	}

	var stagingJobs, prodJobs []int64
	for _, f := range files {
		logger.Infof("Loading title from MARC in %q", f)
		var data []byte
		data, err = os.ReadFile(f)
		if err != nil {
			return err
		}

		var m *marc.MARC
		m, err = marc.ParseXML(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("parsing %q: %w", f, err)
		}

		var id int64
		id, err = staging.LoadTitle(data)
		if err != nil {
			return fmt.Errorf("loading %q into staging: %w", f, err)
		}
		stagingJobs = append(stagingJobs, id)
		id, err = prod.LoadTitle(data)
		if err != nil {
			return fmt.Errorf("loading %q into production: %w", f, err)
		}
		prodJobs = append(prodJobs, id)

		err = saveTitle(m)
		if err != nil {
			return fmt.Errorf("saving title from %q: %w", f, err)
		}
	}

	for _, id := range stagingJobs {
		err = waitJob("staging", staging, id)
		if err != nil {
			return err
		}
	}
	for _, id := range prodJobs {
		err = waitJob("production", prod, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveTitle(m *marc.MARC) error {
	var t, err = models.FindTitleByLCCN(m.LCCN())
	if err != nil {
		return fmt.Errorf("looking up LCCN %q: %w", m.LCCN(), err)
	}

	t.LCCN = m.LCCN()
	t.Name = m.Title() + " (" + m.Location() + ")"
	t.ValidLCCN = true
	t.MARCTitle = m.Title()
	t.MARCLocation = m.Location()
	t.LangCode3 = m.Language()

	// Fake SFTP credentials, so SFTPGo-based uploads can be tested
	t.LegacyPass = "pass"
	t.SFTPUser = t.LCCN

	return t.Save()
}

// waitJob waits for an ONI Agent job to finish, returning an error unless it
// succeeds
func waitJob(env string, agent *openoni.RPC, id int64) error {
	for {
		var js, err = agent.GetJobStatus(id)
		if err != nil {
			return fmt.Errorf("checking %s job %d: %w", env, id, err)
		}

		switch js {
		case openoni.JobStatusPending, openoni.JobStatusStarted:
			logger.Debugf("%s job %d is %s; waiting", env, id, js)
			time.Sleep(time.Second)
		case openoni.JobStatusSuccessful:
			logger.Infof("%s job %d succeeded", env, id)
			return nil
		default:
			return fmt.Errorf("%s job %d: status %q", env, id, js)
		}
	}
}
