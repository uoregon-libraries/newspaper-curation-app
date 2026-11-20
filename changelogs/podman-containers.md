### Changed

- Development now happens entirely in containers. The "hybrid" setup (NCA
  binaries on the host, services in Docker) is gone: NCA's web server, job
  runner, and one-off commands run in containers with your checkout mounted,
  and rebuild from it whenever they start. See the Developer's Guide.
- The compose setup was rebuilt to work for production as well as dev,
  modeled on our other podman stacks: all configuration in `.env`,
  production-minded defaults (no published ports, restart policies,
  per-service log files), named volumes only, and profiles for optional
  services (`sftpgo`, `local-dev` for the demo ONI stacks, and `tools` for a
  `cli` service).
- The compose setup requires podman and podman-compose; Docker is no longer
  supported. Use `scripts/compose` instead of `podman compose`, since older
  podman-compose versions ignore `COMPOSE_PROFILES`.
- The web proxy is Caddy instead of nginx.
- MariaDB is pinned to 12.3 (LTS) instead of `latest`.
- `scripts/localdev.sh`, `manage`, and the test recipes drive the container
  stack instead of host binaries. `manage backup|restore` no longer needs
  sudo.
- `test/copy-sources.go` empties the fake mount's directories instead of
  deleting them, so containers with those directories mounted don't need to
  be restarted.

### Added

- `NCA_CONFIG` can be used in place of `-c` / `--config` for every NCA
  binary.
- `NCA_DEBUG=true` turns on the web server's debug mode, same as `--debug`.
- `.env` builds public URLs from pieces (`PUBLIC_SCHEME`, `PUBLIC_HOST`,
  `PUBLIC_PORT`, and per-service ports for the dev ONI stacks and SFTPGo),
  and the dev override publishes services on those same ports, so URLs and
  ports can't drift apart.
- A rootless podman systemd user unit for the stack (`docker/nca@.service`)
  and a "Containers" page in the setup docs.

### Removed

- `compose.override-example.yml` and `compose.override-hybrid-example.yml`,
  replaced by `compose.override.example.yml` and `.env.example`
- The "Not Compiling Locally" developer docs

### Migration

- Developers: create `.env` from `.env.example` and replace your
  `compose.override.yml` with `compose.override.example.yml`, then run
  `resetdb` or `prep_for_testing` (from `scripts/localdev.sh`) to start
  fresh. Your old `settings` file isn't used by the dev stack. To keep your
  issue cache, move its files into `tmp/issue-cache`.
