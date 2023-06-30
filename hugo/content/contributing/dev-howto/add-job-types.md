---
title: Add Job Types
weight: 10
description: Adding a new type of job into the code
---

When a new kind of job is required for backend processing, it has to be done
properly in order to ensure it is used, set up, and processed by NCA.

- Make sure there aren't already existing jobs that do what you want!  There
  are a lot of jobs in NCA already, and some are meant to be very generic, such
  as `JobTypeRenameDir`.
  - Read and make sure you understand *all structs* in `src/jobs` that
    implement `Process`
- Create a new `JobType` in [`src/models/job.go`][1].
  - Add the `JobType` to the const list
    - Make sure the string is 100% unique within that list!
  - Add the new `JobType` to the `ValidJobTypes` list
- Create a new struct that implements the `Process` method.
  - Use an existing Go file if it makes sense (e.g., another metadata or
    filesystem job) or create a new one in `src/jobs/`.
  - Make sure you document the type!  What is its purpose?
  - Need an example?  The metadata jobs are very simple and can be found in
    [`src/jobs/metadata_jobs.go`][2].
- Wire up the `JobType` to the concrete `Process` implementor
  - This is done in [`src/jobs/jobs.go`][3], in the `DBJobToProcessor` function
- Queue a job of the new type.
  - See [`src/jobs/queue.go`][4]
  - You might need to create a new arg value in `src/models/pipeline.go`, like
    `JobArgSource`, `JobArgWorkflowStep`, etc.
  - You will certainly need to create the job and push it into a queue. This
    happens in a `Queue...` function (e.g., `QueueBatchForDeletion`).
- Make something run jobs of the new type.
  - For almost any new job, you'll just add the type to an existing runner
    function in [`src/cmd/run-jobs/main.go`][5] (`runAllQueues`).  This ensures
    a simple job runner invocation (with the `watchall` argument) will run your
    new job type.

[1]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/models/job.go>
[2]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/jobs/metadata_jobs.go>
[3]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/jobs/jobs.go>
[4]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/jobs/queue.go>
[5]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/cmd/run-jobs/main.go>
