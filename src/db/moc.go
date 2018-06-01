package db

// MOC contains MARC org codes
type MOC struct {
	ID   int
	Code string
}

// ValidMOC returns true if the given code is in the database
func ValidMOC(code string) bool {
	var op = DB.Operation()
	op.Dbg = Debug
	var moc = &MOC{}
	var ok = op.Select("mocs", &MOC{}).Where("code = ?", code).First(moc)
	return op.Err() == nil && ok
}

// AllMOCs returns the full list of MOCs in the database, sorted by their org code
func AllMOCs() ([]*MOC, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*MOC
	op.Select("mocs", &MOC{}).AllObjects(&list)
	return list, op.Err()
}
