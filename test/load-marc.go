//go:build ignore

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/marc"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

var conf *config.Config

type _opts struct {
	cli.BaseOptions
}

var opts _opts
var l = logger.New(logger.Debug, false)

func getOpts() {
	var c = cli.New(&opts)
	conf = c.GetConf()
	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		l.Fatalf("Unable to connect to DB: %s", err)
	}
}

func main() {
	getOpts()

	// Find all *.xml and *.mrk files in the marc-xml dir
	var wd, err = os.Getwd()
	if err != nil {
		l.Fatalf("Cannot get current directory!")
	}
	var marcDir = filepath.Join(wd, "sources", "marc-xml")

	var files []string
	files, err = getFiles(marcDir, ".xml", ".XML", ".mrk", ".MRK")
	if err != nil {
		l.Fatalf("Unable to read MARC files in %q: %s", marcDir, err)
	}

	var stagAgent, prodAgent *openoni.RPC
	stagAgent, err = openoni.New(conf.StagingAgentConnection)
	if err != nil {
		l.Fatalf("Invalid staging agent connection string %q: %s", conf.StagingAgentConnection, err)
	}
	prodAgent, err = openoni.New(conf.ProductionAgentConnection)
	if err != nil {
		l.Fatalf("Invalid production agent connection string %q: %s", conf.ProductionAgentConnection, err)
	}

	var data []byte
	for _, f := range files {
		l.Infof("Loading title from MARC in %q", f)
		data, err = os.ReadFile(f)
		if err != nil {
			l.Fatalf("Unable to read file %q: %s", f, err)
		}

		var m *marc.MARC
		m, err = marc.ParseXML(bytes.NewReader(data))
		if err != nil {
			l.Fatalf("Unable to parse %q into a MARC record: %s", f, err)
		}

		_, err = stagAgent.LoadTitle(data)
		if err != nil {
			l.Fatalf("Unable to load %q to staging: %s", f, err)
		}
		_, err = prodAgent.LoadTitle(data)
		if err != nil {
			l.Fatalf("Unable to load %q to production: %s", f, err)
		}
		l.Infof("Successfully loaded %q to staging and production", f)

		var t *models.Title
		t, err = models.FindTitleByLCCN(m.LCCN())
		if err != nil {
			l.Fatalf("Unable to check the database for this title (fname %q, lccn %q): %s", f, m.LCCN(), err)
		}

		// Create or update title. TODO: centralize this?
		t.LCCN = m.LCCN()
		t.Name = m.Title() + " (" + m.Location() + ")"
		t.ValidLCCN = true
		t.MARCTitle = m.Title()
		t.MARCLocation = m.Location()
		t.LangCode3 = m.Language()

		// Put in fake SFTP stuff so we can do sftpgo-based uploads if we want
		t.LegacyPass = "pass"
		t.SFTPUser = t.LCCN

		err = t.Save()
		if err != nil {
			l.Fatalf("Unable to save title (fname %q, lccn %q): %s", f, m.LCCN(), err)
		}
	}
}

func getFiles(dir string, exts ...string) ([]string, error) {
	var fileList, err = fileutil.FindIf(dir, func(i os.FileInfo) bool {
		for _, ext := range exts {
			if filepath.Ext(i.Name()) == ext {
				return true
			}
		}
		return false
	})

	sort.Strings(fileList)
	return fileList, err
}