-- This will update batches that were archived over 4 weeks ago and then report
-- all locations on disk where issues are in a "live_done" batch (and therefore
-- should be safe to delete).  Make sure the batches are backed up first!
--
-- This can be run as follows to get a simple text file of deletable paths:
--
--     source /path/to/nca/settings
--     mysql -BN -h$DB_HOST -u$DB_USER -p$DB_PASSWORD $DB_DATABASE < identify-removable-issues.sql > removable-paths.txt
UPDATE batches SET status = 'live_done' WHERE status = 'live' AND archived_at < DATE_SUB(NOW(), INTERVAL 4 WEEK);
SELECT i.location
  FROM issues i
  JOIN batches b ON (i.batch_id = b.id)
  WHERE
    i.ignored = 1 AND
    i.workflow_step = 'InProduction' AND
    i.location <> '' AND
    b.status = 'live_done';

-- Commented out because you *really* don't want to do this until the folders are definitely deleted
--
-- UPDATE issues
--   SET location = ''
--   WHERE
--     workflow_step = 'InProduction' AND
--     ignored = 1 AND
--     batch_id in (SELECT id FROM batches WHERE status = 'live_done');
