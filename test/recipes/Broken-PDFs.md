# Testing Broken PDFs

This was my "script" when testing out how broken PDFs looked when refactoring
some critical code related to removal of dead issues. You wouldn't use this
script precisely as-is, but it should serve as a decent boilerplate for a
similar kind of test.

## Script

```bash
# Set an environment variable "name" to "baseline" or "fix" depending on which run this is
export name=baseline
export name=fix

# Begin
prep_for_testing >prep.log 2>&1

# Wait until DB is up and start workers
workers >work.log 2>&1

# Check logs for errors just to be sure things did what we expect
cat work.log | grep -v " - \(INFO\|DEBUG\) - "
cat prep.log | grep -v " - \(INFO\|DEBUG\) - "

# Stop workers, save state using the "name" variable from above
./manage backup 01-$name

# Break two PDF issues
echo "bad" > ./test/fakemount/page-review/sn88086023-2022080401-**/seq-0026.pdf
echo "bad" > ./test/fakemount/page-review/2021242619-2020090201-**/seq-0003.pdf

# Renumber and make older
cd test
./rename-page-review.sh && ./make-older.sh
cd ..

# Restart workers, replacing log
workers >work.log 2>&1

# Wait for jobs to complete and the two failures to finalize (~10min)

# Stop workers, make another backup using the "name" variable from above
./manage backup 02-$name

# Run the command to purge dead issues (omit "--live" in new fixed branch)
make clean && make bin/purge-dead-issues && ./bin/purge-dead-issues -c ./settings --live

# Purged issues should include:
#
# sn88086023/2022080401
# 2021242619/2020090201

# Run workers one last time
workers >work.log 2>&1

```

Wait for all jobs to complete, then create your report using `$name`.
