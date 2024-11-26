package issuequeue

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func atoi(s string) int {
	var i, _ = strconv.Atoi(s)
	return i
}

// strToIssue converts a string to issue data for easier testing. The string is
// a slash-delimited set of values: parseable date, edition number, and page
// count. All issues use the "good" LCCN, so tests using this shouldn't be
// testing anything related to wrapping potentially bad issues.
func strToIssue(s string) *models.Issue {
	var parts = strings.Split(s, "/")
	return &models.Issue{
		LCCN:      lccnSimple,
		Date:      parts[0],
		Edition:   atoi(parts[1]),
		PageCount: atoi(parts[2]),
		Title:     testTitleList.FindByLCCN(lccnSimple),
	}
}

func TestAppend(t *testing.T) {
	var tests = map[string]struct {
		issues        []string
		expectedCount int
		expectedPages int
		hasError      bool
	}{
		"Two unique issues": {
			issues:        []string{"2020-12-31/01/10", "2020-12-31/02/11"},
			expectedCount: 2,
			expectedPages: 21,
			hasError:      false,
		},
		"Three issues, one dupe": {
			issues:        []string{"2020-12-31/01/10", "2020-12-31/02/11", "2020-12-31/01/12"},
			expectedCount: 2,
			expectedPages: 21,
			hasError:      false,
		},
		"One valid issue, two bad issue dates": {
			issues:        []string{"2020-11-31/01/10", "2020-13-31/02/11", "2020-12-31/01/12"},
			expectedCount: 1,
			expectedPages: 12,
			hasError:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var q = New()
			var hadError error
			for _, i := range tc.issues {
				var err = q.Append(strToIssue(i))
				if err != nil {
					t.Logf("%q error: %s", i, err)
					hadError = err
				}
			}

			if tc.hasError && hadError == nil {
				t.Fatalf("Expected an error")
			}
			if !tc.hasError && hadError != nil {
				t.Fatalf("Expected no errors, got: %s", hadError)
			}

			var got = q.Len()
			if got != tc.expectedCount {
				t.Fatalf("Expected IssueCount to be %d, got %d", tc.expectedCount, got)
			}
			got = q.Pages
			if got != tc.expectedPages {
				t.Fatalf("Expected %d page(s), got %d", tc.expectedPages, got)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	var tests = map[string]struct {
		issuePages         []int
		queuePages         int
		expectedQueueCount int
		expectedQueueSizes []int
	}{
		"6 issues, 20 max pages": {
			issuePages:         []int{5, 6, 7, 8, 9, 11},
			queuePages:         20,
			expectedQueueCount: 3,
			expectedQueueSizes: []int{16, 15, 15},
		},
		"20 issues, 40 max pages": {
			issuePages: []int{
				4, 7, 4, 6, 4,
				8, 4, 8, 6, 6,
				8, 8, 1, 8, 6,
				7, 5, 5, 4, 4,
			},
			queuePages:         40,
			expectedQueueCount: 3,
			expectedQueueSizes: []int{40, 37, 36},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var q = New()
			for i, pages := range tc.issuePages {
				q.Append(strToIssue(fmt.Sprintf("2024-01-02/%02d/%d", i, pages)))
			}

			var qlist = q.Split(tc.queuePages)
			var got = len(qlist)
			if got != tc.expectedQueueCount {
				t.Fatalf("Expected %d queue(s), got %d", tc.expectedQueueCount, got)
			}

			var gotSizes []int
			for _, q := range qlist {
				gotSizes = append(gotSizes, q.Pages)
			}

			var diff = cmp.Diff(gotSizes, tc.expectedQueueSizes)
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}
