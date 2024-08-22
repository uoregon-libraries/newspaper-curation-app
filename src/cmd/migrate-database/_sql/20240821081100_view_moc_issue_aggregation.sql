-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE VIEW moc_issue_aggregation AS
  SELECT m.id AS id, m.code, m.name, i.workflow_step, COUNT(i.id) AS issue_count, SUM(i.page_count) AS total_pages
    FROM mocs m
    JOIN issues i ON (i.marc_org_code = m.code)
    GROUP BY marc_org_code, workflow_step
    ORDER BY marc_org_code, workflow_step;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP VIEW moc_issue_aggregation;
