### Fixed

- The `manage` command now hacks the SFTPGo docker volume's owner after a
  restore to fix a permissions problem when the container starts up
