# Deploying SFTPGo

These notes are intended for UO and may not be relevant to anybody else, but
they should at least serve as a rough guide if necessary.

## Prep

Communication:

- Let ODNP team know when outage is planned *at least* a week in advance (they
  need to let publishers know)

SFTP Server:

- Make sure the latest Go compiler is installed!
- Replace manual install of SFTPGo with latest
  - More painful to manage than using their custom RPMs, but guaranteed not to
    conflict with automated server updates
  - Building from source (easier to pull new tags for updates) - see below
  - Setup service (see below)
- Configuration:
  - `track_quota` = 1
  - Make sure it's still on port 2022
  - Figure out what's necessary to use the existing sftp dirs so we aren't
    having to move files during the downtime. If it's too much trouble, though,
    moving files is still acceptable.
- Create a test user and validate various operations. Purge said user.
- Firewall: allow http traffic from the NCA server at a minimum; maybe library
  IP ranges for easier SFTPGo administration if necessary?

### Build from source

https://github.com/drakkan/sftpgo/blob/main/docs/build-from-source.md

```
go build \
  -tags nomysql,nopgsql,nosqlite,nogcs,nos3,noazblob \
  -ldflags "-s -w -X github.com/drakkan/sftpgo/v2/version.commit=$(git describe --always --dirty) -X github.com/drakkan/sftpgo/v2/version.date=$(date -u +%FT%TZ)" \
  -o sftpgo
```

### Set up service

https://github.com/drakkan/sftpgo/blob/main/docs/service.md

First, to remove everything from a prior install:

```
sudo systemctl stop sftpgo

entries="/etc/sftpgo /var/lib/sftpgo /usr/share/sftpgo /usr/bin/sftpgo /etc/systemd/system/sftpgo.service"

# Make a backup of all removed files/dirs just in case...
sudo tar -czf /root/sftpgo-backup-$(date +"%Y-%m-%d").tgz $entries

# Remove things
sudo rm -rf $entries
```

Full command list from docs, in case they get removed or something:

```
# create the sftpgo user and group
sudo groupadd --system sftpgo
sudo useradd --system \
  --gid sftpgo \
  --no-create-home \
  --home-dir /var/lib/sftpgo \
  --shell /usr/sbin/nologin \
  --comment "SFTPGo user" \
  sftpgo
# create the required directories
sudo mkdir -p /etc/sftpgo \
  /var/lib/sftpgo \
  /usr/share/sftpgo

# install the sftpgo executable
sudo install -Dm755 sftpgo /usr/bin/sftpgo
# install the default configuration file, edit it if required
sudo install -Dm644 sftpgo.json /etc/sftpgo/
# override some configuration keys using environment variables
sudo sh -c 'echo "SFTPGO_HTTPD__BACKUPS_PATH=/var/lib/sftpgo/backups" >> /etc/sftpgo/sftpgo.env'
sudo sh -c 'echo "SFTPGO_DATA_PROVIDER__CREDENTIALS_PATH=/var/lib/sftpgo/credentials" >> /etc/sftpgo/sftpgo.env'
# if you use a file based data provider such as sqlite or bolt consider to set the database path too, for example:
#sudo sh -c 'echo "SFTPGO_DATA_PROVIDER__NAME=/var/lib/sftpgo/sftpgo.db" >> /etc/sftpgo/sftpgo.env'
# also set the provider's PATH as env var to get initprovider to work with SQLite provider:
#export SFTPGO_DATA_PROVIDER__NAME=/var/lib/sftpgo/sftpgo.db
# install static files and templates for the web UI
sudo cp -r static templates openapi /usr/share/sftpgo/
# set files and directory permissions
sudo chown -R sftpgo:sftpgo /etc/sftpgo /var/lib/sftpgo
sudo chmod 750 /etc/sftpgo /var/lib/sftpgo
sudo chmod 640 /etc/sftpgo/sftpgo.json /etc/sftpgo/sftpgo.env
# initialize the configured data provider
# if you want to use MySQL or PostgreSQL you need to create the configured database before running the initprovider command
sudo -E su - sftpgo -m -s /bin/bash -c 'sftpgo initprovider -c /etc/sftpgo'
# install the systemd service
sudo install -Dm644 init/sftpgo.service /etc/systemd/system
# start the service
sudo systemctl start sftpgo
# verify that the service is started
sudo systemctl status sftpgo
# automatically start sftpgo on boot
sudo systemctl enable sftpgo
# optional, create shell completion script, for example for bash
sudo sh -c '/usr/bin/sftpgo gen completion bash > /usr/share/bash-completion/completions/sftpgo'
# optional, create man pages
sudo /usr/bin/sftpgo gen man -d /usr/share/man/man1
```

Make sure bolt is the configured data store, e.g. in `/etc/sftpgo/sftpgo.json`:

```
...

  "data_provider": {
    "driver": "bolt",
    "name": "sftpgo.db",
    "host": "",
...

```

May need to run `systemctl daemon-reload` prior to restarting the service.

## Downtime

Communication:

- Let ODNP team know update is starting and system is unavailable

NCA:

- Shut down httpd and workers

SFTP Server:

- Add a temporary firewall rule on SFTP server to block port 22 from external users
- Swap sshd to port 2022
- Configure SFTPGo to use port 22
- Lock down 2022 to UO ranges (maybe just library staff?)

NCA:

- Update NCA codebase
- Change settings to add new SFTPGo configuration
- Start httpd and workers
- Verify titles are migrated to SFTPGo
- Make sure updating a title in NCA changes it in SFTPGo
- Verify sftp login as an actual publisher works
- If necessary, move files to the new sftpgo-managed locations

Communication:

- ODNP team should test SFTP a second time to be sure devs didn't make any
  incorrect assumptions in above test phase

## Post-verification

SFTP Server:

- Remove temporary firewall rule to block port 22 from outside UO

Communication:

- ODNP team: system is live and NCA is usable again
- Give publishers new ssh key fingerprint and let them know that they can
  resume uploads
