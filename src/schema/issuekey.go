package schema

import (
	"fmt"
	"strings"
)

// CondensedDate returns the date in a consistent format for use in issue key TSV output
func CondensedDate(rawDate string) string {
	return strings.Replace(rawDate, "-", "", -1)
}

// IssueDateEdition returns the combination of condensed date (no hyphens) and
// two-digit edition number for use in issue keys and other places we need the
// "local" unique string
func IssueDateEdition(rawDate string, edition int) string {
	return fmt.Sprintf("%s%02d", CondensedDate(rawDate), edition)
}

// IssueKey centralizes the generation of our unique "key" for an issue using
// the lccn + date + edition
func IssueKey(lccn, rawDate string, edition int) string {
	return lccn + "/" + IssueDateEdition(rawDate, edition)
}
