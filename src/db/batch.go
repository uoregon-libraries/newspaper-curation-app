package db

import (
	"time"

	"github.com/Nerdmaster/magicsql"
)

// Batch contains metadata for generating a batch XML.  Issues can be
// associated with a single batch, and a batch will typically have many issues
// assigned to it.
type Batch struct {
	ID          int `sql:",primary"`
	MARCOrgCode string
	Name        string
	CreatedAt   time.Time

	issues []*Issue
}

// CreateBatch creates a batch in the database, using its ID to generate a
// unique batch name, and associating the given list of issues.  This is
// inefficient, but it gets the job done.
func CreateBatch(moc string, issues []*Issue) (*Batch, error) {
	var op = DB.Operation()
	op.BeginTransaction()
	defer op.EndTransaction()

	var b = &Batch{MARCOrgCode: moc, CreatedAt: time.Now(), issues: issues}
	var err = b.SaveOp(op)
	if err != nil {
		return nil, err
	}

	for _, i := range issues {
		i.BatchID = b.ID
		i.SaveOp(op)
	}

	b.Name = RandomBatchName(b.ID)
	err = b.SaveOp(op)
	return b, err
}

// Issues pulls all issues from the database which have this batch's ID
func (b *Batch) Issues() ([]*Issue, error) {
	if len(b.issues) > 0 {
		return b.issues, nil
	}

	if b.ID == 0 {
		return b.issues, nil
	}

	var issues, err = FindIssuesByBatchID(b.ID)
	b.issues = issues
	return b.issues, err
}

// Save creates or updates the Batch in the batches table
func (b *Batch) Save() error {
	var op = DB.Operation()
	op.Dbg = Debug
	return b.SaveOp(op)
}

// SaveOp saves the batch to the batches table with a custom operation for
// easier transactions
func (b *Batch) SaveOp(op *magicsql.Operation) error {
	op.Save("batches", b)
	return op.Err()
}
