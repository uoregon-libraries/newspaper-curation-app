package db

import (
	"fmt"
	"schema"
	"sync"
	"time"
)

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
var lastTitleLoad time.Time
var atMutex sync.RWMutex

// FindTitleByLCCN returns the title matching the given LCCN or nil
func FindTitleByLCCN(lccn string) (*Title, error) {
	var err = LoadTitles()
	if err != nil {
		return nil, err
	}

	atMutex.RLock()
	defer atMutex.RUnlock()

	for _, t := range allTitles {
		if t.LCCN == lccn {
			return t, nil
		}
	}
	return nil, nil
}

// FindTitleByDirectory looks up a title by the given directory string,
// matching it against the sftp_dir field in the database
func FindTitleByDirectory(dir string) (*Title, error) {
	var err = LoadTitles()
	if err != nil {
		return nil, err
	}

	atMutex.RLock()
	defer atMutex.RUnlock()

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

	atMutex.Lock()
	defer atMutex.Unlock()

	if len(allTitles) != 0 && time.Since(lastTitleLoad) < time.Minute*15 {
		return nil
	}

	var op = DB.Operation()
	op.Dbg = Debug
	op.Select("titles", &Title{}).AllObjects(&allTitles)
	lastTitleLoad = time.Now()
	return op.Err()
}

// AllTitles returns everything in the title cache, reloading it if needed
func AllTitles() ([]*Title, error) {
	var err = LoadTitles()
	if err != nil {
		return nil, err
	}
	return allTitles, nil
}

// LookupTitle looks up the title in the the database by directory name and
// LCCN to give a simpler way to find titles in a general case.  This only
// works after titles have been loaded in order to simplify usage, but it's up
// to the caller to make sure titles have in fact been loaded.
func LookupTitle(identifier string) *Title {
	for _, t := range allTitles {
		if t.SFTPDir == identifier {
			return t
		}
		if t.LCCN == identifier {
			return t
		}
	}

	return nil
}

// SchemaTitle converts a database Title to a schema.Title instance
func (t *Title) SchemaTitle() *schema.Title {
	// Check for self being nil so we can safely chain this function
	if t == nil {
		return nil
	}

	var ttl, loc = t.MarcTitle, t.MarcLocation

	// Not great, but this does the trick well enough when we haven't gotten a
	// valid MARC record
	if !t.ValidLCCN {
		ttl = t.Title
	}

	return &schema.Title{
		LCCN:               t.LCCN,
		Name:               ttl,
		PlaceOfPublication: loc,
	}
}
