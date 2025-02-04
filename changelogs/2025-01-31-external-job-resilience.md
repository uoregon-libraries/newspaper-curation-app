### Fixed

- Jobs involving external services (ONI Agent) are now set up such that the
  whole group is retried when something goes wrong. This fixes the "wait for
  ONI" loop when an ONI Agent job fails:
  - NCA queues a job that will call out to ONI Agent (e.g., load a batch). This
    job almost always succeeds because it's just asking the agent to put
    something in *its* job queue.
  - NCA queues a job to check ONI Agent for success. This fails if the Agent's
    job failed.
  - On failure of any job, NCA retries that job. In this case, it *only*
    retries the "check for success" job. Which just rechecks a failed Agent
    job.
  - The "check for success" job fails, and a retry is queued. But no matter how
    many times you ask "did you succeed?", if you don't start a new external
    job, the answer is still "no".
- ONI Agent's magic "job is redundant and not queued" response is now handled
  properly as a success. (e.g., queueing a batch to be loaded when it's already
  been loaded successfully)

### Added

- New test recipe script for helping guide a dev in testing a fake Solr outage
