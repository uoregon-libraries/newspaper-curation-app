package schema

import (
	"testing"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
)

func TestIssueErrorWarning(t *testing.T) {
	var a apperr.Error = &IssueError{
		Err:  "test",
		Msg:  "test message",
		Prop: true,
		Warn: true,
	}

	if !a.Warning() {
		t.Errorf("error wasn't a warning")
	}
}
