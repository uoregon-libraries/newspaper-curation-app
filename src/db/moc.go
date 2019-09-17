package db

// MOC contains MARC org codes
type MOC struct {
	ID   int `sql:",primary"`
	Code string
}

// FindMOCByCode searches the database for the given MOC and returns it if it's
// found, or nil if not
func FindMOCByCode(code string) (*MOC, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var moc = &MOC{}
	var ok = op.Select("mocs", &MOC{}).Where("code = ?", code).First(moc)
	if !ok {
		return nil, op.Err()
	}
	return moc, op.Err()
}

// FindMOCByID finds the MOC by its id
func FindMOCByID(id int) (*MOC, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var moc = &MOC{}
	var ok = op.Select("mocs", &MOC{}).Where("id = ?", id).First(moc)
	if !ok {
		return nil, op.Err()
	}
	return moc, op.Err()
}

// AllMOCs returns the full list of MOCs in the database, sorted by their org code
func AllMOCs() ([]*MOC, error) {
	var op = DB.Operation()
	op.Dbg = Debug
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
	var op = DB.Operation()
	op.Dbg = Debug
	op.Save("mocs", moc)
	return op.Err()
}

// Delete removes this MOC from the database
func (moc *MOC) Delete() error {
	var op = DB.Operation()
	op.Dbg = Debug
	op.Exec("DELETE FROM mocs WHERE id = ?", moc.ID)
	return op.Err()
}
