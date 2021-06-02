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
- Create a new `JobType` in [`src/models/job.go`](https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/models/job.go).
  - Add the `JobType` to the const list
    - Make sure the string is 100% unique within that list!
  - Add the new `JobType` to the `ValidJobTypes` list
- Create a new struct that implements the `Process` method.
  - Use an existing Go file if it makes sense (e.g., another metadata or filesystem job) or
    create a new one in `src/jobs/`.
  - Make sure you document the type!  What is its purpose?
  - Need an example?  The metadata jobs are very simple and can be found in
    [`src/jobs/metadata_jobs.go`](https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/jobs/metadata_jobs.go).
- Wire up the `JobType` to the concrete `Process` implementor
  - This is done in
    [`src/jobs/jobs.go`](https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/jobs/jobs.go),
    in the `DBJobToProcessor` function
- Queue a job of the new type.
  - See [`src/jobs/queue.go`](https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/jobs/queue.go)
  - You might create a new `Prepare...Job` function, or simply use an existing
    one with the new type
  - You might need to create a new arg value, like `srcArg`, `forcedArg`, etc.
    for the processor to use
  - You will certainly need to create the job and push it into a queue.
    Typically this happens in a `Queue...` function.
- Make something run jobs of the new type.
  - For almost any new job, you'll just add the type to an existing runner
    function in [`src/cmd/run-jobs/main.go`](https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/cmd/run-jobs/main.go)
    (`runAllQueues`).  This ensures a simple job runner invocation (with the
    `watchall` argument) will run your new job type.
