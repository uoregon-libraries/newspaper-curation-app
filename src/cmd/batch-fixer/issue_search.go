package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Valid search fields
const (
	fLCCN  = "lccn"
	fDate  = "date"
	fKey   = "key"
	fTitle = "title"
)

var validFields = []string{fLCCN, fDate, fKey, fTitle}

type query struct {
	field     string
	condition *regexp.Regexp
}

func newQuery(field, condition string) (*query, error) {
	if condition[0] != '^' {
		condition = "^" + condition
	}
	if condition[len(condition)-1] != '$' {
		condition += "$"
	}
	var re, err = regexp.Compile(condition)
	if err != nil {
		return nil, fmt.Errorf("malformed condition %q: %s", condition, err)
	}

	for _, f := range validFields {
		if f == field {
			return &query{field, re}, nil
		}
	}

	return nil, fmt.Errorf("unknown field %q (valid fields are %s)", field, strings.Join(validFields, ", "))
}

func (q *query) match(i *Issue) bool {
	var val string
	switch q.field {
	case fDate:
		val = i.db.Date
	case fLCCN:
		val = i.db.LCCN
	case fKey:
		val = i.db.Key()
	case fTitle:
		val = i.db.Title.Name
	}
	return q.condition.MatchString(val)
}

type queries struct {
	list []*query
}

func (q *queries) add(qtxt string) error {
	var parts = strings.SplitN(qtxt, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("%q is invalid: you must have a field followed by an"+
			`equals sign and a value in regex form: e.g., "date=200[01]0101"`, qtxt)
	}

	var qry, err = newQuery(parts[0], parts[1])
	if err != nil {
		return fmt.Errorf("invalid query %q: %s", qtxt, err)
	}

	q.list = append(q.list, qry)
	return nil
}

func (q *queries) match(i *Issue) bool {
	for _, qry := range q.list {
		if !qry.match(i) {
			return false
		}
	}

	return true
}
