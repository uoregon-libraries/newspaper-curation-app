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
- Dev and test tooling is Go instead of bash. `bin/ncadev` (built by `make
  tools`) replaces `scripts/localdev.sh` and `manage`. The `test/` scripts
  are now `bin/ncatest` plus a test harness that runs in a container
  (`test/harness`), and the recipe scripts are recipes built into `ncatest`.
  Save points (formerly `manage backup|restore`) no longer need sudo.
- Manual testing happens on a separate test stack, run by `bin/ncatest`. It's
  configured by the repository, not your override or shell variables, and
  takes only `PUBLIC_HOST` from your `.env`, so tests behave the same for
  everybody, and testing doesn't touch your dev stack's data. Its workflow
  tree is `test/fakemount`. The dev stack's is wherever your override binds
  it (`DEV-VOLS/news` in the example); the dev tools read that from the
  override.
- Loading test issues copies the source files instead of hard-linking them, so
  breaking a workflow file (as some recipes do) can't damage the test sources.
  It empties the workflow directories rather than deleting them, so containers
  with those directories mounted don't need to be restarted.
- Recipes keep the workers service stopped and run jobs only at set points,
  so runs are more repeatable.
- `run-jobs --exit-when-done` exits five seconds after the last job finishes,
  once the folder scans it runs at startup are done, instead of always
  waiting 30 seconds before it starts checking. That makes `workonce` and
  recipes faster, and fixes the job runner quitting before a slow startup
  scan (e.g., with a cold issue cache) could queue anything.

### Added

- `NCA_CONFIG` can be used in place of `-c` / `--config` for every NCA
  binary.
- `NCA_DEBUG=true` turns on the web server's debug mode, same as `--debug`.
- `.env` builds public URLs from pieces (`PUBLIC_SCHEME`, `PUBLIC_HOST`,
  `PUBLIC_PORT`, and per-service ports for the dev ONI stacks and SFTPGo),
  and the dev override publishes services on those same ports, so URLs and
  ports can't drift apart. `NCA_URL_PATH` puts NCA under a path (e.g.,
  `/nca`) while page images stay at `/images/iiif`.
- A rootless podman systemd user unit for the stack (`docker/nca@.service`)
  and a "Containers" page in the setup docs.

### Removed

- `compose.override-example.yml` and `compose.override-hybrid-example.yml`,
  replaced by `compose.override.example.yml` and `.env.example`
- The "Not Compiling Locally" developer docs
- `scripts/localdev.sh`, `manage`, and the scripts in `test/` and
  `test/recipes/`, replaced by `bin/ncadev`, `bin/ncatest`, and the test
  harness. `test/reset.sh`'s quick reset is gone; use `bin/ncatest prep`.
- The "Replacing Live Issues" docs and `scripts/help-remove-issues.go`. The
  helper needed `find-issues`, which was removed in v6.3.0, so the process has
  been broken since then. A replacement built into NCA is planned.
- Dead scripts: `docker/tag-latest.sh` and `docker/force-tag-latest.sh`
  (built a compose service that no longer exists), `test/mets.sh` (used the
  long-gone `rewrite-mets` command), and `scripts/fake-oni-manager.go` (faked
  ONI's `manage.py`, which NCA hasn't called since moving to the ONI Agent)

### Migration

- Developers: you need Go on your host for the dev tools. Create `.env` from
  `.env.example`, replace your `compose.override.yml` with
  `compose.override.example.yml`, run `make tools`, then `bin/ncadev reset`
  or `bin/ncadev prep` to start fresh. Your old `settings` file isn't used by
  the dev stack. To keep your issue cache, move its files into
  `DEV-VOLS/issue-cache`. Dev NCA now lives under a path, at
  <http://localhost:3333/nca>.
