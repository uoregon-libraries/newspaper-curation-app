package models

import "github.com/uoregon-libraries/newspaper-curation-app/src/dbi"

// MOC contains MARC org codes
type MOC struct {
	ID   int64 `sql:",primary"`
	Code string
	Name string
}

// FindMOCByCode searches the database for the given MOC and returns it if it's
// found, or nil if not
func FindMOCByCode(code string) (*MOC, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var moc = &MOC{}
	var ok = op.Select("mocs", &MOC{}).Where("code = ?", code).First(moc)
	if !ok {
		return nil, op.Err()
	}
	return moc, op.Err()
}

// FindMOCByID finds the MOC by its id
func FindMOCByID(id int64) (*MOC, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var moc = &MOC{}
	var ok = op.Select("mocs", &MOC{}).Where("id = ?", id).First(moc)
	if !ok {
		return nil, op.Err()
	}
	return moc, op.Err()
}

// AllMOCs returns the full list of MOCs in the database, sorted by their org code
func AllMOCs() ([]*MOC, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*MOC
	op.Select("mocs", &MOC{}).Order("code").AllObjects(&list)
	return list, op.Err()
}

// ValidMOC returns true if the given code is in the database
func ValidMOC(code string) bool {
	var moc, err = FindMOCByCode(code)
	return moc != nil && err == nil
}

// Save creates or updates the MOC
func (moc *MOC) Save() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Save("mocs", moc)
	return op.Err()
}

// Delete removes this MOC from the database
func (moc *MOC) Delete() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Exec("DELETE FROM mocs WHERE id = ?", moc.ID)
	return op.Err()
}
