# Testing Broken PDFs Part II

This recipe tests what happens when an issue job fails but leaves behind jobs
that aren't directly tied to an issue (generic jobs like syncdir). This is
useful when testing out whether the issue purge is catching everything left
behind (hint: it wasn't).

## Script

```bash
# Set an environment variable "name" to "baseline" or "fix" depending on which run this is
export name=baseline
export name=fix

# Begin
make clean && make && source scripts/localdev.sh && prep_for_testing >prep.log 2>&1

# Check logs for errors just to be sure things did what we expect
cat prep.log | grep -v " - \(INFO\|DEBUG\) - "

# Save state using the "name" variable from above
./manage backup 01-$name

# Break an issue so the "page_split" job fails. Backup the PDF first since it's
# a hard-link of the file in sources!
cp ./test/fakemount/sftp/polkitemizer/2010-04-19/0005.pdf ./backup.pdf
echo "bad" > ./test/fakemount/sftp/polkitemizer/2010-04-19/0005.pdf

# Hack up the job runner's exponential backoff so it maxes at one second
sed -i 's|var maxDelay =.*$|var maxDelay = time.Second|' src/models/job.go

# Run workers; wait for jobs to complete / fail (~10min)
workers

# Stop workers, make another backup using the "name" variable from above
./manage backup 02-$name

# Run the command to purge dead issues
./bin/purge-dead-issues -c ./settings

# Purged issue should be sn96088087/2010041901

# Run workers one last time (~1 min)
workers

# Restore the bad PDF
cp ./backup.pdf test/sources/sftp/sn96088087-2010041901/0005.pdf
```

Wait for all jobs to complete, then create your report using `$name`. You may
have to search for jobs that are *not* `on_hold` if still running a version of
NCA that doesn't properly clean up non-issue-specific jobs, as those will never
run or fail (hence the fix and this test). e.g.,

```sql
# This will show jobs that should have been closed but weren't:
select * from jobs where status not in ('success', 'failed_done');

# So you have to exclude on_hold to see when the job runner is done:
select * from jobs where status not in ('success', 'failed_done', 'on_hold');
```
