# SFTPGo Files

Everything here is internal to UO, but we're holding onto it because it could
be useful for others when first setting up SFTPGo.

- [`deploy.md`](deploy.md) is a checklist of things we needed to
  do in order to migrate. Having dozens of live publishers, this was a
  complicated process, and this file should give just about any NCA user
  everything they'd need for a similar migration.
- [`curl-example.sh`](curl-example.sh) is a simple example of how you might
  use the SFTPGo API to acquire a token and then use that token to request the
  API's "version" endpoint.
- [`sftpgo.env`](sftpgo.env) is the environment file we put into
  `/etc/sftpgo.env` for our production sftp server.
- [`sftpgo.nginx.conf`](sftpgo.nginx.conf) is a bare-bones nginx configuration
  for exposing the SFTPGo web UI.

Note that these files likely will not be kept in sync with our production
server - they should serve as an example for setup more than a long-term way to
keep a secure SFTPGo system running.
