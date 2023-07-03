# General Testing

This script's purpose is to just run through an end-to-end test starting from
nothing, faking curation, and generating batches. It's a good starting point
for building other tests, and an okay test when doing a refactor that may cause
unexpected changes.

## Script

```bash
# Begin
prep_for_testing

# Wait until DB is up and start workers
workers

# Wait for jobs to finish

# Renumber page review PDFs and hack their date
cd test
./rename-page-review.sh && ./make-older.sh
cd ..

# If necessary, restart workers to make the PDF mover re-read the filesystem quicker
workers

# Wait for jobs to complete (~10min)

# Stop workers, save state
./manage backup 01-$name

# Fake-curate, fake-review
cd test
go run run-workflow.go -c ../settings --operation curate
go run run-workflow.go -c ../settings --operation review
cd ..

# Start workers, wait for jobs to complete (~30sec)
workers

# Stop workers, save state again
./manage backup 02-$name

# Generate batches
./bin/queue-batches -c ./settings

# Start workers, wait for jobs to complete (~30sec)
workers

# Done!
```
