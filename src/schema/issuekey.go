package schema

import (
	"fmt"
	"strings"
)

// CondensedDate returns the date in a consistent format for use in issue key TSV output
func CondensedDate(rawDate string) string {
	return strings.Replace(rawDate, "-", "", -1)
}

// IssueKey centralizes the generation of our unique "key" for an issue using
// the lccn + date + edition
func IssueKey(lccn, rawDate string, edition int) string {
	return fmt.Sprintf("%s/%s%02d", lccn, CondensedDate(rawDate), edition)
}
