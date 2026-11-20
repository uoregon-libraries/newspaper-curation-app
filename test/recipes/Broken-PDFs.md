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
bin/ncatest prep >prep.log 2>&1

# Run jobs until there are none left
bin/ncatest workonce >work.log 2>&1

# Check logs for errors just to be sure things did what we expect
grep -v " - \(INFO\|DEBUG\) - " work.log
grep -v " - \(INFO\|DEBUG\) - " prep.log

# Save state using the "name" variable from above
bin/ncatest save 01-$name

# Break two PDF issues
echo "bad" > ./test/fakemount/page-review/sn88086023-2022080401-**/seq-0026.pdf
echo "bad" > ./test/fakemount/page-review/2021242619-2020090201-**/seq-0003.pdf

# Renumber and make older
bin/ncatest rename-page-review && bin/ncatest make-older

# Run jobs again, replacing the log
bin/ncatest workonce >work.log 2>&1

# Wait for jobs to complete and the two failures to finalize (~10min)

# Make another save point using the "name" variable from above
bin/ncatest save 02-$name

# Run the command to remove dead issues
bin/ncatest cli remove-dead-issues

# Deleted issues should include:
#
# sn88086023/2022080401
# 2021242619/2020090201

# Run jobs one last time
bin/ncatest workonce >work.log 2>&1
```

Wait for all jobs to complete, then create your report: `bin/ncatest report
$name`.
