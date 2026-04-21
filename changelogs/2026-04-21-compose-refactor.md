### Changed

- Refactored the compose stack (dev-only) to make things more podman-compatible
  and adhere a little better to best practices
  - Environment vars are hashes (key/value) instead of lists
  - Volume overrides in the "hybrid" compose example are done by modifying the
    volumes directly instead of mounting local files per-service
  - All image names have their registry prefix (docker.io)
  - Removed obsolete mysql config
  - Fixed ONI setup for easier dev work in non-localhost environment (e.g., a
    VMWare stack you need to access via an IP address)
