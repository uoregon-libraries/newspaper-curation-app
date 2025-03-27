package models

import (
	"testing"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

type mockModel struct {
	ID  int64 `sql:",primary"`
	Foo string
	Bar string
}

// We wrap this in a test runner "wrapper" function to avoid potential
// collisions as we add tests to things which use the coreFinder
func TestCoreFinder(t *testing.T) {
	dbi.DB = &magicsql.DB{}
	t.Run("newCoreFinder", func(t *testing.T) {
		var cf = newCoreFinder("mocks", &mockModel{})

		if cf.conditions == nil {
			t.Errorf("conditions should be initialized, but was nil")
		}
		if cf.op == nil {
			t.Errorf("operation should be initialized, but was nil")
		}
		if cf.ord != "" {
			t.Errorf("order should be empty initially, but was %q", cf.ord)
		}
		if cf.lim != 0 {
			t.Errorf("limit should be zero initially, but was %d", cf.lim)
		}
	})

	// This does a bit of testing of the underlying magicsql stuff, but ensures
	// that we're actually sending things to that API properly
	t.Run("Expected SQL", func(t *testing.T) {
		type testCase struct {
			conditions map[string]any
			limit      int
			order      string
			expectSQL  string
		}

		var tests = map[string]testCase{
			"No filters": {
				conditions: map[string]any{},
				expectSQL:  "SELECT id,foo,bar FROM mocks",
			},
			"Single condition with arg": {
				conditions: map[string]any{"foo = ?": "baz"},
				expectSQL:  "SELECT id,foo,bar FROM mocks WHERE (foo = ?)",
			},
			"Single condition without arg": {
				conditions: map[string]any{"bar IS NULL": nil},
				expectSQL:  "SELECT id,foo,bar FROM mocks WHERE (bar IS NULL)",
			},
			"Multiple conditions with IN clause": {
				conditions: map[string]any{"foo = ?": "baz", "bar IN (??)": []any{1, 2, 5, 3}},
				expectSQL:  "SELECT id,foo,bar FROM mocks WHERE (bar IN (?,?,?,?)) AND (foo = ?)",
			},
			"Multiple conditions": {
				conditions: map[string]any{"foo = ?": "baz", "bar IS NOT NULL": nil},
				expectSQL:  "SELECT id,foo,bar FROM mocks WHERE (bar IS NOT NULL) AND (foo = ?)",
			},
			"Limit": {
				conditions: map[string]any{},
				limit:      10,
				expectSQL:  "SELECT id,foo,bar FROM mocks LIMIT 10",
			},
			"Order": {
				conditions: map[string]any{},
				order:      "foo DESC",
				expectSQL:  "SELECT id,foo,bar FROM mocks ORDER BY foo DESC",
			},
			"Limit and Order": {
				conditions: map[string]any{},
				limit:      5,
				order:      "bar ASC",
				expectSQL:  "SELECT id,foo,bar FROM mocks ORDER BY bar ASC LIMIT 5",
			},
			"All filters": {
				conditions: map[string]any{"foo = ?": "baz", "id > ?": 100},
				limit:      20,
				order:      "id ASC",
				expectSQL:  "SELECT id,foo,bar FROM mocks WHERE (foo = ?) AND (id > ?) ORDER BY id ASC LIMIT 20",
			},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				var cf = newCoreFinder("mocks", &mockModel{})
				cf.conditions = tc.conditions
				cf.lim = tc.limit
				cf.ord = tc.order

				var got = cf.selector().SQL()
				if got != tc.expectSQL {
					t.Errorf("selector SQL mismatch: got %q, expected %q", got, tc.expectSQL)
				}
			})
		}
	})
}
