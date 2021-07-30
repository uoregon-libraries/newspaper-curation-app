// Package schema houses simple data types for titles, issues, batches, etc.
// Types which live here are generally meant to be very general-case rather
// than trying to hold all possible information for all possible use cases.
package schema

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// WorkflowStep describes the location within the workflow any issue can exist
// - this is basically a more comprehensive list than what's in the database in
// order to capture every possible location: live batches, sftped issues
// awaiting processing, etc.
type WorkflowStep string

// All possible statuses an issue could have
const (
	// WSNil should only be used to indicate a workflow step is irrelevant or else unset
	WSNil                    WorkflowStep = ""
	WSSFTP                                = "SFTPUpload"
	WSScan                                = "ScanUpload"
	WSAwaitingProcessing                  = "AwaitingProcessing"
	WSAwaitingPageReview                  = "AwaitingPageReview"
	WSReadyForMetadataEntry               = "ReadyForMetadataEntry"
	WSAwaitingMetadataReview              = "AwaitingMetadataReview"
	WSUnfixableMetadataError              = "UnfixableMetadataError"
	WSReadyForMETSXML                     = "ReadyForMETSXML"
	WSReadyForBatching                    = "ReadyForBatching"
	WSInProduction                        = "InProduction"
)

// Batch represents high-level batch information
type Batch struct {
	// MARCOrgCode tells us the organization responsible for the images in the batch
	MARCOrgCode string

	// A batch's keyword is normally short, such as "horsetail", but our in-house
	// batches have much longer keywords to ensure uniqueness
	Keyword string

	// Usually 1, but I've seen "_ver02" batches occasionally
	Version int

	// Issues links the issues which are part of this batch
	Issues IssueList

	// Location is where this batch can be found, either a URL or filesystem path
	Location string

	Errors apperr.List
}

// ParseBatchname creates a Batch by splitting up the full name string
func ParseBatchname(fullname string) (*Batch, error) {
	// All batches must have the format "batch_MARCORGCODE_NAME_ver##"
	var parts = strings.Split(fullname, "_")

	// This is really obnoxious, but we can only test for too few parse.  Despite
	// the spec's claim that the batch keyword must not have underscores, some
	// live batches do.  I'm lookin' at you, "courage_3".
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid batch format")
	}

	if parts[0] != "batch" {
		return nil, fmt.Errorf(`batches must begin with "batch_"`)
	}

	var l = len(parts)
	var b = &Batch{}
	var ver string
	parts, ver = parts[:l-1], parts[l-1]
	b.MARCOrgCode, b.Keyword = parts[1], strings.Join(parts[2:], "_")

	if len(ver) != 5 || ver[:3] != "ver" {
		return nil, fmt.Errorf("invalid version format")
	}

	b.Version, _ = strconv.Atoi(ver[3:])
	if b.Version < 1 {
		return nil, fmt.Errorf("invalid version value")
	}

	return b, nil
}

// Fullname is the full batch name
func (b *Batch) Fullname() string {
	var parts = []string{"batch", b.MARCOrgCode, b.Keyword, fmt.Sprintf("ver%02d", b.Version)}
	return strings.Join(parts, "_")
}

// TSV returns a string uniquely identifying this batch by location as well
// as name, and an issue count to offer some verification or reporting
func (b *Batch) TSV() string {
	return fmt.Sprintf("%s\t%s\t%06d", b.Location, b.Fullname(), len(b.Issues))
}

// AddIssue adds the issue to this batch's list, and sets the issue's batch
func (b *Batch) AddIssue(i *Issue) {
	b.Issues = append(b.Issues, i)
	i.Batch = b
}

// AddError attaches err to this batch
func (b *Batch) AddError(err apperr.Error) {
	b.Errors.Append(err)
}

// Title is a publisher's information, unique per LCCN
type Title struct {
	LCCN               string
	Name               string
	PlaceOfPublication string
	Errors             apperr.List
	hasChildErrors     bool

	// Issues contains the list of issues associated with a single title; though
	// this can be derived by iterating over all the issues, it's useful to store
	// them here, too
	Issues IssueList

	// Location is where the title was found on disk or web; not actual Title metadata
	Location string
}

// TSV returns a string representing this title uniquely by including its
// location and a count of issues.  The issue count won't help us deserialize,
// but the purpose is just for data verification and simple reporting.
func (t *Title) TSV() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%06d", t.Location, t.LCCN, t.Name, t.PlaceOfPublication, len(t.Issues))
}

// AddIssue adds the issue to this title's list, and sets the issue's title
func (t *Title) AddIssue(i *Issue) *Issue {
	t.Issues = append(t.Issues, i)
	i.Title = t
	return i
}

// GenericTitle returns a title with the same generic information, but none of
// the data which is tied to a specific title on the filesystem or website:
// location and issue list
func (t *Title) GenericTitle() *Title {
	return &Title{LCCN: t.LCCN, Name: t.Name, PlaceOfPublication: t.PlaceOfPublication}
}

// AddError attaches err to this title
func (t *Title) AddError(err apperr.Error) {
	t.Errors.Append(err)
}

// addChildError notes that this title has at least one issue with errors.
// Issue errors don't make a title invalid, so we don't add an error to the
// title itself.
func (t *Title) addChildError() {
	t.hasChildErrors = true
}

// HasIssueErrors reports whether any of this title's issues have errors
func (t *Title) HasIssueErrors() bool {
	return t.hasChildErrors
}

// TitleList is a simple slice of titles for easier built-in sorting and
// identifying a unique list of all titles
type TitleList []*Title

// TrimCommonPrefixes strips "The", "A", and "An" from the string if they're at
// the beginning, and removes leading spaces
func TrimCommonPrefixes(s string) string {
	s = strings.TrimPrefix(s, "The ")
	s = strings.TrimPrefix(s, "the ")
	s = strings.TrimPrefix(s, "A ")
	s = strings.TrimPrefix(s, "a ")
	s = strings.TrimPrefix(s, "An ")
	s = strings.TrimPrefix(s, "an ")
	return strings.TrimSpace(s)
}

// SortByName sorts the titles by their name, using location and lccn when
// names are the same
func (list TitleList) SortByName() {
	sort.Slice(list, func(i, j int) bool {
		var a = strings.ToLower(TrimCommonPrefixes(list[i].Name))
		var b = strings.ToLower(TrimCommonPrefixes(list[j].Name))

		if a == b {
			a, b = list[i].Location, list[j].Location
		}
		if a == b {
			a, b = list[i].LCCN, list[j].LCCN
		}

		return a < b
	})
}

// Unique returns a new list containing generic versions of each unique LCCN
func (list TitleList) Unique() TitleList {
	var l2 TitleList
	var seen = make(map[string]bool)
	for _, title := range list {
		if seen[title.LCCN] {
			continue
		}

		seen[title.LCCN] = true
		l2 = append(l2, title.GenericTitle())
	}
	return l2
}

// Issue is an extremely basic encapsulation of an issue's high-level data
type Issue struct {
	DatabaseID     int // This will be zero for issues which aren't instantiated from the database conversion
	MARCOrgCode    string
	Title          *Title
	RawDate        string // This is the date as seen on the filesystem when the issue was uploaded
	Edition        int
	Batch          *Batch
	Files          []*File
	Errors         apperr.List
	hasChildErrors bool

	// Location is where this issue can be found, either a URL or filesystem path
	Location string

	WorkflowStep WorkflowStep
}

// DateEdition returns the combination of condensed date (no hyphens) and
// two-digit edition number for use in issue keys and other places we need the
// "local" unique string
func (i *Issue) DateEdition() string {
	return IssueDateEdition(i.RawDate, i.Edition)
}

// Key returns the unique string that represents this issue
func (i *Issue) Key() string {
	return IssueKey(i.Title.LCCN, i.RawDate, i.Edition)
}

// TSV gives us something which can be used to uniquely identify all aspects of
// this issue's data for reporting and/or data verification
func (i *Issue) TSV() string {
	var bString = "nil"
	if i.Batch != nil {
		bString = strings.Replace(i.Batch.TSV(), "\t", "\\t", -1)
	}
	var tString = strings.Replace(i.Title.TSV(), "\t", "\\t", -1)
	var fileNames []string
	for _, file := range i.Files {
		fileNames = append(fileNames, file.Name)
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s", bString, tString, i.Location,
		i.DateEdition(), i.WorkflowStep, strings.Join(fileNames, ","))
}

// FindFiles clears the issue's file list and then reads everything in the
// issue directory, appending it to the now-empty list.  This will silently
// fail when the issue's location is invalid, not readable, or isn't an
// absolute path beginning with "/".  This is only meant for issues already
// discovered on the filesystem.
func (i *Issue) FindFiles() {
	i.Files = nil

	if i.Location[0] != '/' {
		return
	}

	var infos, err = fileutil.ReaddirSortedNumeric(i.Location)
	if err != nil {
		logger.Errorf("Error trying to open %q to read contents: %s", i.Location, err)
		return
	}

	for _, file := range fileutil.InfosToFiles(infos) {
		var loc = filepath.Join(i.Location, file.Name)
		i.Files = append(i.Files, &File{File: file, Issue: i, Location: loc})
	}
}

// IsLive returns true if the issue both has a batch *and* the batch appears to
// be on the live site
func (i *Issue) IsLive() bool {
	return i.Batch != nil && i.Batch.Location[0:4] == "http"
}

// WorkflowIdentification returns a human-readable explanation of where an
// issue lives currently is in the workflow - currently used for adding to
// "likely duplicate of ..."
//
// Several of the values below won't make sense in dupe reporting, but are in
// this list becuause (a) if some logic changes and a dupe report does happen,
// they need something more than "an unknown issue" describing them, and (b)
// this function may eventually be useful beyond explanation of duped issues.
func (i *Issue) WorkflowIdentification() string {
	switch i.WorkflowStep {
	case WSSFTP:
		return "a born-digital issue waiting for processing"

	case WSScan:
		return "a scanned issue waiting for processing"

	case WSUnfixableMetadataError:
		return "an issue with errors, awaiting remediation"

	case WSAwaitingProcessing:
		return "a pending issue"

	case WSAwaitingPageReview:
		return "an issue awaiting page reordering / renumbering"

	case WSReadyForMetadataEntry:
		return "an issue awaiting metadata entry"

	case WSAwaitingMetadataReview:
		return "an issue awaiting metadata review"

	case WSReadyForBatching:
		return "an issue waiting to be batched"

	case WSInProduction:
		return "a live issue in batch " + i.Batch.Fullname()

	default:
		return fmt.Sprintf("an unknown issue (location: %q)", i.Location)
	}
}

// addError attaches err to this issue and reports to the issue's title that it
// has an error
func (i *Issue) addError(err apperr.Error) {
	i.Errors.Append(err)
	if err.Propagate() && i.Title != nil {
		i.Title.addChildError()
	}
}

// addChildError sets a flag to let us know this issue has a child with an
// error.  If this is the first time a child has reported an error, we store an
// error on the issue itself so we can inform users just once instead of once
// per error.
func (i *Issue) addChildError() {
	if i.hasChildErrors == true {
		return
	}
	i.addError(apperr.New("one or more files are invalid"))
	i.hasChildErrors = true
}

// LastModified tells us when *any* change happened in an issue's folder.  This
// will return a meaningless value on live issues.
func (i *Issue) LastModified() time.Time {
	if i.WorkflowStep == WSInProduction {
		return time.Time{}
	}

	var info, err = os.Stat(i.Location)
	if err != nil {
		logger.Warnf("Unable to stat %q: %s", i.Location, err)
		return time.Now()
	}
	var modified = info.ModTime()

	var files []os.FileInfo
	files, err = ioutil.ReadDir(i.Location)
	if err != nil {
		logger.Warnf("Unable to read dir %q: %s", i.Location, err)
		return time.Now()
	}

	for _, file := range files {
		var mod = file.ModTime()
		if modified.Before(mod) {
			modified = mod
		}
	}

	return modified
}

// METSFile returns the canonical path to an issue's METS file
func (i *Issue) METSFile() string {
	return filepath.Join(i.Location, i.DateEdition()+".xml")
}

// CheckDupes centralizes the logic for seeing if an issue has a duplicate in a
// given lookup, adding a duplication error if there is a dupe and that dupe is
// considered to be more "canonical" than this issue.  e.g., if there's an
// issue in the metadata entry stage and another in the sftp upload, the upload
// is considered the dupe, not the one in metadata entry.
func (i *Issue) CheckDupes(lookup *Lookup) {
	// Get a search key for this issue.  If the issue key is invalid, that
	// probably means a bad upload, and so dupe-checking doesn't really matter
	var sKey, err = ParseSearchKey(i.Key())
	if err != nil {
		return
	}

	for _, i2 := range lookup.Issues(sKey) {
		if i.WorkflowStep.before(i2.WorkflowStep) {
			i.ErrDuped(i2)
		}
	}
}

// before tells us if wsa is logically before wsb in terms of issues flowing
// through the system.  This helps determine what to report if there's
// duplicated data: anything that's earlier in the process is the dupe, as the
// later something is, the more metadata scrutiny has gone into it.
//
// In other words, A step is before another if it represents data that's less
// certain; e.g., an uploaded issue is completely unknown and is therefore
// before all other steps, but a live issue is considered done and wouldn't be
// before anything else.
func (wsa WorkflowStep) before(wsb WorkflowStep) bool {
	var stepOrder = map[WorkflowStep]int{
		// Nil is before literally everything except another nil
		WSNil: 0,

		// Issues that are broken in some way have metadata we can't rely on, so we
		// declare them as being "before" everything else
		WSUnfixableMetadataError: 10,

		// The uploads come before anything that isn't another upload, or nil
		WSSFTP: 20,
		WSScan: 20,

		// Awaiting processing is a meaningless step that just says something
		// automated needs to happen.  Because of this we can't say where it should
		// fit in the workflow.  We have to return *something*, so we just say this
		// isn't "before" anything (other than live issues).  Once processing is
		// complete, it'll have a meaningful step again, and at that point we'll be
		// able to catch any problems.
		WSAwaitingProcessing: 100,

		// Awaiting page review is still fairly unknown, like uploads, and only comes
		// after them to make it clear that a new upload shouldn't supercede a
		// previous upload.  But this could cause false dupe flags if an old upload
		// had the wrong folder name, so I could see a case for changing this to the
		// same value as uploads.
		WSAwaitingPageReview: 30,

		WSReadyForMetadataEntry:  40,
		WSAwaitingMetadataReview: 50,

		// When an issue is waiting for METS XML, its metadata is in exactly the
		// same state as when it's ready for batching, and no dupe checking occurs
		// here anyway, so these are considered equal
		WSReadyForMETSXML:  60,
		WSReadyForBatching: 60,

		// Let's just make sure in-production always comes after everything else,
		// even the unknown awaiting-processing issues
		WSInProduction: math.MaxInt32,
	}

	return stepOrder[wsa] < stepOrder[wsb]
}

// IssueList groups a bunch of issues together
type IssueList []*Issue

// SortByKey modifies the IssueList in place so they're sorted alphabetically
// by issue key.  In cases where the keys are the same, the TSV is used to
// ensure sorting is still consistent, if not ideal.
func (list IssueList) SortByKey() {
	sort.Slice(list, func(i, j int) bool {
		var kA, kB = list[i].Key(), list[j].Key()
		if kA != kB {
			return kA < kB
		}

		return list[i].TSV() < list[j].TSV()
	})
}

var validInternalName = regexp.MustCompile(`(?i:^([0-9]{4}.(pdf|jp2|xml|tif))|[0-9]{10}.xml|[a-z]+.tar)`)

// File just gives fileutil.File a location and issue pointer
type File struct {
	*fileutil.File
	Location string
	Issue    *Issue
	Errors   apperr.List
}

// ValidInternalName just returns whether the filename matches what we allow to exist once
// something has entered NCA's internal workflow
func (f *File) ValidInternalName() bool {
	return validInternalName.MatchString(f.Name)
}

// AddError puts err on this file and reports to its issue that one of its
// children has an error
func (f *File) AddError(err apperr.Error) {
	f.Errors.Append(err)
	if err.Propagate() {
		f.Issue.addChildError()
	}
}
