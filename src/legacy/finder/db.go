package main

import (
	"db"
	"log"
)

var titlesBySFTPDir = make(map[string]*Title)
var titlesByLCCN = make(map[string]*Title)

type dbTitle struct {
	ID      int `sql:",primary"`
	SFTPDir string
	LCCN    string
}

// cacheDBTitles caches all titles by SFTP directory and LCCN for easy lookup
// when we are dealing with unknown path elements that may be from an SFTP
// source or an in-house scan
func cacheDBTitles() {
	var op = db.DB.Operation()
	op.Dbg = true
	var dbTitles []*dbTitle
	op.Select("titles", &dbTitle{}).AllObjects(&dbTitles)
	if op.Err() != nil {
		log.Fatalf("ERROR: Unable to query titles: %s", op.Err())
	}

	for _, t := range dbTitles {
		var title = &Title{LCCN: t.LCCN}
		if t.SFTPDir != "" {
			titlesBySFTPDir[t.SFTPDir] = title
		}
		titlesByLCCN[t.LCCN] = title
	}
}
