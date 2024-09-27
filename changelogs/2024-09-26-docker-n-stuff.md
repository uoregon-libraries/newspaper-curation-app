### Fixed

- Minor documentation errors have been corrected

### Changed

- Everything related to docker compose should now reflect the best practices /
  conventions: `compose.yml` instead of `docker-compose.yml`, no "version"
  definition, etc.

### Migration

- If you use docker compose, you probably need to rename your override to `compose.override.yml`
