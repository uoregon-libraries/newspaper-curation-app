// Package issuesearch defines a way to search any schema.IssueList by LCCN,
// LCCN + year, LCCN + year + month, LCCN + year + month + day, or even LCCN +
// year + month + day + edition.  Lookup is used to perform searches, and
// requires a Key, which is a text string containing one of the above sets of
// data, such as "sn12345678/2006010201".
package issuesearch
