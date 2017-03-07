package main

import (
	"db"
	"log"
)

var sftpTitlesByName = make(map[string]*Title)

// SFTPTitle is used to read non-historic titles from the database
type SFTPTitle struct {
	ID    int `sql:",primary"`
	SFTPDir string
	LCCN  string
}

// cacheSFTPTitlesByName caches all titles by sftp directory for easy lookup by
// sftp dir since SFTP dirs are often not the same as LCCN
func cacheSFTPTitlesByName() {
	var op = db.DB.Operation()
	op.Dbg = true
	var titles []*SFTPTitle
	op.Select("titles", &SFTPTitle{}).AllObjects(&titles)
	if op.Err() != nil {
		log.Fatalf("ERROR: Unable to query sftp titles")
	}

	for _, t := range titles {
		if t.SFTPDir == "" {
			continue
		}
		sftpTitlesByName[t.SFTPDir] = &Title{LCCN: t.LCCN}
	}
}
