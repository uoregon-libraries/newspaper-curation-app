package models

import (
	"sort"
	"strings"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// coreFinder holds the common logic for database finders. It's not meant to be
// used directly, but embedded into specific finders like IssueFinder.
// Generally the "outer" type should only need to alter conditions and order.
//
// The type parameter T represents the concrete finder type which embeds
// coreFinder, allowing methods like Limit to return the correct type.
type coreFinder[T any] struct {
	// outer stores a pointer to the embedding struct (the concrete finder type)
	outer T

	// conditions holds the WHERE clauses for the query. The key is the SQL
	// fragment, and the value is the argument for that fragment (if any). Use
	// nil for fragments without arguments (e.g., "deleted_at IS NULL").
	conditions map[string]any
	op         *magicsql.Operation
	sel        magicsql.Select
	ord        string
	lim        int
}

// newCoreFinder initializes a coreFinder. It requires the embedding struct
// (outer), the table name, and a destination prototype object (e.g., &Issue{}).
func newCoreFinder[T any](outer T, tableName string, dest any) *coreFinder[T] {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	return &coreFinder[T]{
		outer:      outer,
		conditions: make(map[string]any),
		op:         op,
		sel:        op.Select(tableName, dest),
	}
}

// Limit sets the max records to return
func (f *coreFinder[T]) Limit(limit int) T {
	f.lim = limit
	return f.outer
}

// OrderBy sets an order for this finder.
//
// TODO: This currently requires a raw SQL order string which ties business
// logic and DB schema too tightly. Not sure the best way to address this.
func (f *coreFinder[T]) OrderBy(order string) T {
	f.ord = order
	return f.outer
}

// selector modifies the finder's internal magicsql.Select object based on the
// finder's state (conditions, limit, order).
func (f *coreFinder[T]) selector() magicsql.Select {
	var where []string
	var args []any

	// Order the conditions so the query is always the same. I *think* this helps
	// server-side optimizing, but even if not, it definitely helps us test.
	var keys []string
	for k := range f.conditions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		var v = f.conditions[k]
		if strings.Contains(k, "(??)") {
			// If we have an IN-style query, we have to turn the args into a slice.
			// This may panic. I think we're okay with that for now.
			var vals = v.([]any)
			var placeholders = make([]string, len(vals))
			for i, val := range vals {
				placeholders[i] = "?"
				args = append(args, val)
			}
			var replaced = "(" + strings.Join(placeholders, ",") + ")"
			where = append(where, "("+strings.Replace(k, "(??)", replaced, 1)+")")
		} else {
			where = append(where, "("+k+")")
			if v != nil {
				args = append(args, v)
			}
		}
	}

	var sel = f.sel
	if len(where) > 0 {
		sel = sel.Where(strings.Join(where, " AND "), args...)
	}

	if f.lim > 0 {
		sel = sel.Limit(uint64(f.lim))
	}
	if f.ord != "" {
		sel = sel.Order(f.ord)
	}

	return sel
}

// Fetch runs the query and populates the given list with results. The list
// argument must be a pointer to a slice of the destination type (e.g., &[]*Issue).
func (f *coreFinder[T]) Fetch(list any) error {
	f.selector().AllObjects(list)
	return f.op.Err()
}

// Count returns the number of records this query would return, explicitly
// ignoring any previously-set limit value
func (f *coreFinder[T]) Count() (uint64, error) {
	var s = f.selector()
	s.Limit(0)
	return s.Count().RowCount(), f.op.Err()
}
