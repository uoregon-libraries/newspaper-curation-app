# Deploying SFTPGo

These notes are intended for UO and may not be relevant to anybody else, but
they should at least serve as a rough guide if necessary.

## Prep

Communication:

- Let ODNP team know when outage is planned *at least* a week in advance (they
  need to let publishers know)

SFTP Server:

- Replace manual install of SFTPGo with latest 2.1.x
  - More painful to manage than using their custom RPMs, but guaranteed not to
    conflict with automated server updates
- Configuration:
  - `track_quota` = 1
  - Make sure it's still on port 2022
  - Figure out what's necessary to use the existing sftp dirs so we aren't
    having to move files during the downtime. If it's too much trouble, though,
    moving files is still acceptable.
- Create a test user and validate various operations. Purge said user.
- Firewall: allow http traffic from the NCA server at a minimum; maybe library
  IP ranges for easier SFTPGo administration if necessary?

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
