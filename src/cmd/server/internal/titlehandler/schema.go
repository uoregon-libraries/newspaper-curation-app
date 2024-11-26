package titlehandler

import (
	"regexp"
	"sort"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/datasize"
	"github.com/uoregon-libraries/newspaper-curation-app/src/duration"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

var re = regexp.MustCompile(`[^A-Za-z0-9]+`)
var noWordRE = regexp.MustCompile(`\W+`)

// Title wraps a models.Title for web display
type Title struct {
	*models.Title
	SortName  string
	SFTPPass  string // SFTPPass is a temp field so we can send password updates to SFTPGo
	SFTPQuota datasize.Datasize
}

// WrapTitle converts a models.Title to a Title, giving it a useful "SortName"
// based on its name (stripped of common prefixes) and LCCN
func WrapTitle(t *models.Title) *Title {
	return &Title{
		Title:     t,
		SortName:  strings.ToLower(re.ReplaceAllString(schema.TrimCommonPrefixes(t.Name)+t.LCCN, "-")),
		SFTPQuota: conf.SFTPGoNewUserQuota,
	}
}

// WrapTitles takes a models.TitleList and wraps each title individually
func WrapTitles(list models.TitleList) []*Title {
	var titles = make([]*Title, len(list))
	for i, t := range list {
		titles[i] = WrapTitle(t)
	}

	return titles
}

// SortTitles does an in-place sort on the given title list, relying solely on
// the SortName string
func SortTitles(list []*Title) {
	sort.Slice(list, func(i, j int) bool { return list[i].SortName < list[j].SortName })
}

// TitlesDiffer returns true if the MARC title isn't the same as the name we've
// given the title.  We strip all non-word characters for the comparison.
func (t *Title) TitlesDiffer() bool {
	var mt = noWordRE.ReplaceAllString(t.MARCTitle+t.MARCLocation, "")
	var n = noWordRE.ReplaceAllString(t.Name, "")
	return mt != n
}

// EmbargoSortValue is a hack that gets a close enough version of the embargo
// in a sortable way.  I say "close enough" because it's possible, depending on
// an issue's publication date, that this will be incorrect.  If something has
// a one month embargo and is published in February, that's actually less time
// than a 30-day embargo, but the 30-day embargo is less than a one-month
// embargo that's based in December.
func (t *Title) EmbargoSortValue() float64 {
	// We ignore errors here, since this is strictly for sorting purposes
	var d, _ = duration.Parse(t.EmbargoPeriod)
	return float64(d.Days) + float64(d.Weeks*7) + (float64(d.Months)/12.0+float64(d.Years))*365.25
}

// EmbargoRFC3339 returns the more machine-friendly version of the embargo string
func (t *Title) EmbargoRFC3339() string {
	// We ignore errors here, since a bad string would be the same as no embargo
	// for the purposes of any machine-readable value
	var d, _ = duration.Parse(t.EmbargoPeriod)
	return d.RFC3339()
}
