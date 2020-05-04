package models

import (
	"fmt"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/duration"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Title holds records from the titles table
type Title struct {
	ID            int `sql:",primary"`
	Name          string
	LCCN          string
	EmbargoPeriod string
	Rights        string
	ValidLCCN     bool
	SFTPDir       string
	SFTPUser      string
	SFTPPass      string
	MARCTitle     string
	MARCLocation  string
	LangCode3     string
}

// FindTitle searches the database for a single title
func FindTitle(where string, args ...interface{}) (*Title, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var t = &Title{}
	op.Select("titles", &Title{}).Where(where, args...).First(t)
	return t, op.Err()
}

// FindTitleByID wraps FindTitle to simplify basic finding
func FindTitleByID(id int) (*Title, error) {
	return FindTitle("id = ?", id)
}

// TitleList holds a full list of database titles for quick scan operations on
// all titles, such as is needed to do mass lookups of issues' LCCNs
type TitleList []*Title

// Titles returns all titles in the database for bulk operations
func Titles() (TitleList, error) {
	var allTitles = make(TitleList, 0)
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Select("titles", &Title{}).AllObjects(&allTitles)
	return allTitles, op.Err()
}

// FindByLCCN returns the title matching the given LCCN or nil
func (tl TitleList) FindByLCCN(lccn string) *Title {
	for _, t := range tl {
		if t.LCCN == lccn {
			return t
		}
	}
	return nil
}

// FindByDirectory looks up a title by the given directory string, matching it
// against the sftp_dir field in the database
func (tl TitleList) FindByDirectory(dir string) *Title {
	for _, t := range tl {
		if t.SFTPDir == dir {
			return t
		}
	}
	return nil
}

// Find looks for the title by either directory name or LCCN to give a simpler
// way to find titles in a general case
func (tl TitleList) Find(identifier string) *Title {
	for _, t := range tl {
		if t.SFTPDir == identifier {
			return t
		}
		if t.LCCN == identifier {
			return t
		}
	}

	return nil
}

// Save stores the title data in the database
func (t *Title) Save() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Save("titles", t)
	return op.Err()
}

// CalculateEmbargoLiftDate returns the date an embargo will lift relative to
// the given time (usually this would be an issue's publication date)
func (t *Title) CalculateEmbargoLiftDate(dt time.Time) (time.Time, error) {
	var d, err = duration.Parse(t.EmbargoPeriod)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration: %s", err)
	}

	// If there's no embargo period, the issue's embargo lift date is essentially
	// the beginning of time
	if d.Zero() {
		return time.Time{}, nil
	}

	return dt.AddDate(d.Years, d.Months, d.Weeks*7+d.Days), nil
}

// NormalizedEmbargoPeriod returns a less generic message to describe the
// embargo duration
func (t *Title) NormalizedEmbargoPeriod() string {
	var d, err = duration.Parse(t.EmbargoPeriod)
	if err != nil {
		return "Invalid Embargo!"
	}

	if d.Zero() {
		return "None"
	}

	return d.String()
}

// SchemaTitle converts a database Title to a schema.Title instance
func (t *Title) SchemaTitle() *schema.Title {
	// Check for self being nil so we can safely chain this function
	if t == nil {
		return nil
	}

	var name, loc = t.MARCTitle, t.MARCLocation

	// Not great, but this does the trick well enough when we haven't gotten a
	// valid MARC record
	if !t.ValidLCCN {
		name = t.Name
	}

	return &schema.Title{
		LCCN:               t.LCCN,
		Name:               name,
		PlaceOfPublication: loc,
	}
}

// LangCode ensures that the default is returned in case
// nothing has been retrieved from the MARC record
func (t *Title) LangCode() string {
	if t.LangCode3 == "" {
		return "eng"
	}

	return t.LangCode3
}
