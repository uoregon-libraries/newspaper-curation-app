---
title: Containers
weight: 25
description: Running NCA with podman compose
---

NCA's compose setup runs the whole suite in containers. It's how we develop
NCA (see the [Developer's Guide][dev-guide] for dev-specific setup), and it's
built so production can use the same files with different configuration.

**We haven't moved our own production systems to this setup yet.** It's
designed with production in mind, but treat this page as a starting point, not
a proven recipe.

[dev-guide]: <{{% ref "/contributing/dev-guide" %}}>

## Requirements

- Rootless [Podman](https://podman.io/) and
  [podman-compose](https://github.com/containers/podman-compose). The setup
  uses podman's `k8s-file` log driver, so Docker won't work.
- Two [Open ONI][oni] sites (staging and production), each with an [ONI
  Agent][agent] the NCA host can reach
- An authenticating proxy (e.g., Apache) in front of the stack. See
  [Security](#security).

[oni]: <https://github.com/open-oni/open-oni>
[agent]: <https://github.com/open-oni/oni-agent>

## How it's put together

Services in the core stack (what production runs):

- `proxy`: Caddy, the stack's only published HTTP port. It sends page images
  (`/images/iiif/...`) to RAIS and everything else to NCA.
- `web`: NCA's web server
- `workers`: NCA's job runner
- `db`: MariaDB
- `iiif`: RAIS, serving page images to NCA's metadata entry and review pages

Optional services, enabled by profiles (`COMPOSE_PROFILES` in `.env`):

- `sftpgo`: SFTPGo for publisher uploads, if you don't run it elsewhere
- `local-dev`: staging and production ONI stacks with agents. **Dev only.**
- `tools`: the `cli` service, for one-off commands (see below)

`web` and `workers` both migrate the database when they start, taking turns
via a lock, so upgrades are just a rebuild and restart.

## Configuration

Everything that varies by environment lives in `.env`, which compose
interpolates into `compose.yml`. Required values use `:?`, so the stack
refuses to start if they're missing. Copy `.env.example` and read every
setting: its defaults are for dev, and several of them (debug mode, the SFTPGo
seed) must never be used in production.

NCA's public URL (`PUBLIC_URL`) is built from pieces. In production, the
browser reaches NCA through your authenticating proxy, usually on https's
standard port, so you'd set something like:

```bash
PUBLIC_SCHEME=https
PUBLIC_HOST=nca.example.edu
PUBLIC_PORT=443
PUBLIC_PORT_SUFFIX=
```

`PUBLIC_PORT_SUFFIX` is the URL's port with its colon (e.g., `:8443`), blank
for the scheme's default port. Compose can't derive it, so set it to agree
with `PUBLIC_PORT`. `PUBLIC_PORT` is the port the *browser* sees; only the
dev override publishes the stack on it.

If NCA doesn't live at the root of its host, set `NCA_URL_PATH` (e.g.,
`/nca` for `https://nca.example.edu/nca`). Only NCA moves: page images are
always at `/images/iiif` from the host's root. NCA builds its routes and links
with the path, so the authenticating proxy forwards both paths to the stack
unchanged rather than stripping them. With Apache, for instance:

```apache
ProxyPass        /nca         http://127.0.0.1:8080/nca
ProxyPassReverse /nca         http://127.0.0.1:8080/nca
ProxyPass        /images/iiif http://127.0.0.1:8080/images/iiif
ProxyPassReverse /images/iiif http://127.0.0.1:8080/images/iiif
```

NCA itself reads its usual settings file: the image contains
`settings-example` verbatim, and `compose.yml` overrides values with `NCA_*`
environment variables (NCA's normal environment override mechanism). If a
setting you need to change isn't exposed, add it to `x-nca-env` in
`compose.yml` and to `.env.example`, rather than working around it.

`compose.override.yml` is for structural changes only: published ports and
volumes bound to host directories. A production override might look like
this:

```yaml
services:
  # Only the authenticating proxy should be able to reach this. It's not
  # PUBLIC_PORT: that's the port the browser uses on the proxy.
  proxy:
    ports:
      - "127.0.0.1:8080:80"

volumes:
  news:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /mnt/news
  batches:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /mnt/news/production-batches
  issue-cache:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /path/to/nca/volumes/issue-cache
```

Use absolute paths, not `${PWD}`: `driver_opts` are baked in when a volume is
created, so a bad path sticks until you remove the volume.

Variables exported in the shell override `.env`, so be careful running
compose from an environment that sets things like `DB_PASSWORD`.

## Volumes

- `news`: NCA's workflow tree, mounted at `/mnt/news`: uploads (`sftp`,
  `scans`), `page-review`, `workflow`, `outgoing`, `batch-archive`, `backup`,
  and `errors`. The paths match `settings-example`.
  - The workflow and outgoing directories *must* share this one mount. Batches
    are built by hard-linking workflow files, which fails across mounts even
    when both are the same filesystem underneath.
- `batches`: batches' live files (`BATCH_PRODUCTION_PATH`), mounted at
  `/mnt/batches`. Your ONI servers need to read these.
- `issue-cache`: NCA's cache of issue data, much of it from the live site.
  It's rebuildable, but that takes a long time.
- `db`: MariaDB's data
- `locks`: coordinates migrations between containers
- `sftpgo-home`, `sftpgo-backups`: SFTPGo's data, if you use the `sftpgo`
  profile

Volume names are prefixed with the compose project name, so set
`COMPOSE_PROJECT_NAME` in `.env`. Changing it later orphans the old volumes
rather than renaming them.

With SELinux, containers can only read host directories that are labeled for
it, e.g.:

```bash
sudo semanage fcontext -a -t container_file_t '/path/to/nca/volumes(/.*)?'
sudo restorecon -R /path/to/nca/volumes
```

Network storage (e.g., NFS) has its own rules for rootless containers; make
sure the podman user can read and write it before pointing volumes there.

## Security

NCA doesn't authenticate anybody. It trusts the `X-Remote-User` header to say
who's logged in, and the proxy passes that header through untouched. So:

- Something in front of the stack (e.g., Apache) *must* authenticate users and
  set `X-Remote-User`, overwriting anything the client sent.
- Nothing else may reach the published port. Bind it to localhost if the
  authenticating proxy is on the same host, or firewall it.
- Never set `NCA_DEBUG` in production: debug mode lets anybody act as any
  user.

Caddy trusts `X-Forwarded-For` from `CADDY_TRUSTED_PROXIES` (private ranges
by default) so logs show real client IPs; rootless podman's port forwarding
hides them otherwise.

## Logs

Every service writes stdout/stderr to its own file in `CONTAINER_LOG_DIR`
(`web.log`, `workers.log`, etc.), so logs survive container recreation. The
directory must be an absolute path and must already exist, or containers
won't start. Keep it outside the checkout and rotate the files, e.g., with
logrotate's `copytruncate`. Don't also set a podman log size limit: when a
file hits it, conmon empties the file, which fights logrotate.

## Running commands

These examples run compose through `scripts/compose`, which turns
`COMPOSE_PROFILES` into `--profile` flags. podman-compose before 1.6.0 ignores
`COMPOSE_PROFILES`, and servers are often stuck with an older distro package.
With 1.6.0 or newer, plain `podman compose` works just as well.

One-off commands run in the `cli` service. NCA binaries don't need `-c`,
since the container sets `NCA_CONFIG`:

```bash
scripts/compose --profile tools run --rm --no-deps cli queue-batches
scripts/compose --profile tools run --rm --no-deps cli delete-live-done-issues
```

Keep `--no-deps`: it guarantees a one-off command never starts services that
were stopped on purpose. Add `-T` when running from cron or anything else
without a terminal.

## Running under systemd

`docker/nca@.service` runs the stack as a rootless podman *user* unit. The
podman user needs lingering (`sudo loginctl enable-linger <user>`) so the unit
starts at boot without anybody logging in.

The unit is a template, which keeps host paths out of the repository: the
instance name is the checkout's path, escaped, and the unit uses it as its
working directory. As the podman user, from the checkout:

```bash
mkdir -p ~/.config/systemd/user
cp docker/nca@.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now "nca@$(systemd-escape --path "$PWD").service"
```

Copy the unit rather than using `systemctl link`, and copy it again (then
`daemon-reload`) whenever it changes. `systemctl --user` needs the user's own
session, so `ssh` in as the podman user rather than switching users with
`sudo`.

## Upgrades

Pull the new code, rebuild, and restart:

```bash
scripts/compose build
systemctl --user restart "nca@$(systemd-escape --path "$PWD").service"
```

Database migrations run automatically on startup. Set `NCA_VERSION` in `.env`
(e.g., to the release tag) if you want the version shown in NCA to be
meaningful; the image build can't read it from git.
