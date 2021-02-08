-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE INDEX actions_created_at ON `actions` (`created_at`);
CREATE INDEX actions_object_id ON `actions` (`object_id`);
CREATE INDEX actions_action_type ON `actions` (`action_type`(255));
CREATE INDEX actions_user_id ON `actions` (`user_id`);

CREATE INDEX audit_logs_when ON `audit_logs` (`when`);
CREATE INDEX audit_logs_user ON `audit_logs` (`user`(255));

CREATE INDEX batches_marc_org_code ON `batches` (`marc_org_code`(255));
CREATE INDEX batches_created_at ON `batches` (`created_at`);
CREATE INDEX batches_status ON `batches` (`status`(255));
CREATE INDEX batches_went_live_at ON `batches` (`went_live_at`);
CREATE INDEX batches_archived_at ON `batches` (`archived_at`);

CREATE INDEX issues_marc_org_code ON `issues` (`marc_org_code`(255));
CREATE INDEX issues_lccn ON `issues` (`lccn`(255));
CREATE INDEX issues_metadata_entry_user_id ON `issues` (`metadata_entry_user_id`);
CREATE INDEX issues_reviewed_by_user_id ON `issues` (`reviewed_by_user_id`);
CREATE INDEX issues_workflow_owner_id ON `issues` (`workflow_owner_id`);
CREATE INDEX issues_workflow_step ON `issues` (`workflow_step`(255));
CREATE INDEX issues_rejected_by_user_id ON `issues` (`rejected_by_user_id`);
CREATE INDEX issues_metadata_approved_at ON `issues` (`metadata_approved_at`);
CREATE INDEX issues_batch_id ON `issues` (`batch_id`);

CREATE INDEX job_logs_job_id ON `job_logs` (`job_id`);
CREATE INDEX job_logs_created_at ON `job_logs` (`created_at`);
CREATE INDEX job_logs_log_level ON `job_logs` (`log_level`(255));

CREATE INDEX jobs_created_at ON `jobs` (`created_at`);
CREATE INDEX jobs_job_type ON `jobs` (`job_type`(255));
CREATE INDEX jobs_object_id ON `jobs` (`object_id`);
CREATE INDEX jobs_status ON `jobs` (`status`(255));

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP INDEX actions_created_at ON `actions`;
DROP INDEX actions_object_id ON `actions`;
DROP INDEX actions_action_type ON `actions`;
DROP INDEX actions_user_id ON `actions`;

DROP INDEX audit_logs_when ON `audit_logs`;
DROP INDEX audit_logs_user ON `audit_logs`;

DROP INDEX batches_marc_org_code ON `batches`;
DROP INDEX batches_created_at ON `batches`;
DROP INDEX batches_status ON `batches`;
DROP INDEX batches_went_live_at ON `batches`;
DROP INDEX batches_archived_at ON `batches`;

DROP INDEX issues_marc_org_code ON `issues`;
DROP INDEX issues_lccn ON `issues`;
DROP INDEX issues_metadata_entry_user_id ON `issues`;
DROP INDEX issues_reviewed_by_user_id ON `issues`;
DROP INDEX issues_workflow_owner_id ON `issues`;
DROP INDEX issues_workflow_step ON `issues`;
DROP INDEX issues_rejected_by_user_id ON `issues`;
DROP INDEX issues_metadata_approved_at ON `issues`;
DROP INDEX issues_batch_id ON `issues`;

DROP INDEX job_logs_job_id ON `job_logs`;
DROP INDEX job_logs_created_at ON `job_logs`;
DROP INDEX job_logs_log_level ON `job_logs`;

DROP INDEX jobs_created_at ON `jobs`;
DROP INDEX jobs_job_type ON `jobs`;
DROP INDEX jobs_object_id ON `jobs`;
DROP INDEX jobs_status ON `jobs`;
