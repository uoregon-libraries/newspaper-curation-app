### Fixed

- The `manage` script uses docker's config more effectively in order to ensure
  that backup and restore functions work, and process only the relevant docker
  volumes, no matter what project name you configure in your compose override.
