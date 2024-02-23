### Changed

- The issue queue functionality that automatically moves scanned issues has
  been improved to not log critical errors when an issue has a pending job
  associated with it. Heavy workloads and/or slow filesystems can take hours to
  get huge scanned issues into NCA, and it's not an error to simply have a job
  that hasn't started yet.
