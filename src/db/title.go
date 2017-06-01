package db

import "fmt"

// Title holds records from the titles table
type Title struct {
	ID           int `sql:",primary"`
	Title        string
	LCCN         string
	Embargoed    bool
	Rights       string
	ValidLCCN    bool
	SFTPDir      string
	MarcTitle    string
	MarcLocation string
	IsHistoric   bool
}

// allTitles is a cache of every title read from the database the first time
// any title operations are requested, since the titles table is fairly small
var allTitles []*Title

// FindTitleByLCCN returns the title matching the given LCCN or nil
func FindTitleByLCCN(lccn string) (*Title, error) {
	var err = LoadTitles()
	if err != nil {
		return nil, err
	}
	for _, t := range allTitles {
		if t.LCCN == lccn {
			return t, nil
		}
	}
	return nil, nil
}

func FindTitleByDirectory(dir string) (*Title, error) {
	var err = LoadTitles()
	if err != nil {
		return nil, err
	}

	for _, t := range allTitles {
		if t.SFTPDir == dir {
			return t, nil
		}
	}
	return nil, nil
}

// LoadTitles reads and stores all title data in memory
func LoadTitles() error {
	if DB == nil {
		return fmt.Errorf("DB is not initialized")
	}

	if len(allTitles) != 0 {
		return nil
	}

	var op = DB.Operation()
	op.Dbg = Debug
	op.Select("titles", &Title{}).AllObjects(&allTitles)
	LastError = op.Err()
	return nil
}
