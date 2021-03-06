# report.sh attempts to create a textual representation of the state of the
# fake mount for use when verifying that a given process, after a rewrite /
# refactor, is causing the same results as before.  General process:
#
# - Check out the pre-change branch, e.g., `git checkout develop`
# - Prepare some data in a way that's easily repeated
# - Run jobs / enter metadata / etc.
# - Run a report, e.g., `./report.sh blah`
# - Prepare for diffing - this can be done with git, e.g., `git add blah-report`
# - Reset to the state you set up in step 2
# - Run the same jobs, enter the same metadata, whatever
# - Run a report, e.g., `./report.sh blah`
# - Look at what changed, e.g., `git diff blah-report`
set -eu

source ../settings

sql() {
  mysql -u$DB_USER -p$DB_PASSWORD -D$DB_DATABASE -h$DB_HOST -P$DB_PORT -Ne "$@"
}

strip_dbids() {
  sed 's|\.wip-|XXWIPXX|g' | sed 's|\(..........-..........\)-[0-9]\+|\1|g'
}

repname=${1:-}
if [[ "$repname" == "" ]]; then
  echo "You must specify a name for the report"
  exit 1
fi

repdir="./$repname-report"
rm -rf $repdir
mkdir ./$repdir
find ./fakemount | sort | strip_dbids > $repdir/raw-files.txt

find ./fakemount -name "*.tiff" -or -name "*.tif" | sort | xargs -l1 md5sum | strip_dbids > $repdir/tiffsums.txt

# Dump critical info from the database without having useless churn like
# timestamps or fields that are based on database ids.  This won't cover
# everything, but it should cover enough to have confidence that an end-to-end
# test isn't totally hosed.
sql "
  SELECT marc_org_code, name, status, location
  FROM batches
  ORDER BY marc_org_code, name, status
" > $repdir/dump-batches.sql

sql "
  SELECT
    marc_org_code, date, date_as_labeled, volume, issue, edition,
    edition_label, page_labels_csv, is_from_scanner, workflow_step,
    location, ignored
  FROM issues
  ORDER BY lccn, date, edition
" | strip_dbids > $repdir/dump-issues.sql

sql "
  SELECT job_type, status, object_type, extra_data
  FROM jobs
  ORDER BY job_type, status, object_type, extra_data
" | strip_dbids > $repdir/dump-jobs.sql
