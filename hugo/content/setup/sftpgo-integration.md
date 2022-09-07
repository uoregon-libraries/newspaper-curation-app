---
title: SFTPGo Integration
weight: 45
description: Setting up NCA for SFTPGo integration
---

[SFTPGo](https://github.com/drakkan/sftpgo) is an sftp server that exposes APIs
and a web interface for administration tasks.  We've chosen to integrate NCA
with SFTPGo in order to simplify the process of creating titles for a publisher
that's uploading newspaper PDFs.

If you choose not to use this integration, publisher uploads will have to be
managed entirely by you (as was the case prior to this integration), and NCA
will not track SFTP data (which is a change from NCA 3.x and prior).

## Opt out

To disable SFTPGo integration, assign "-" to the `SFTPGO_API_URL` setting:

    SFTPGO_API_URL="-"

This ensures that NCA will not try to connect to a nonexistent server.

## SFTPGo Setup

If you do opt to use SFTPGo, you'll need to use the SFTPGo documentation to set
it up however it makes sense for your system. Once that's done, you have to
then make NCA aware of it.

Our configuration files can be found in the NCA project's root under the
`sftpgo` directory. It's obviously somewhat specific to our bare-metal RHEL
system, but they should help get you up and running.

We use nginx to proxy the SFTPGo web interface. This isn't necessary -- SFTPGo
can be exposed directly. We prefer the granular control nginx gives us (e.g.,
being able to lock down certain paths by IP).

For the sftp daemon itself, we just expose that directly.

Install nginx, configure it (again, see our configuration if necessary), and
then install SFTPGo as a service. See the ["Running SFTPGo as a service"][1]
page in the official SFTPGo documentation.

SFTPGo configuration: unless you're a masochist, don't try to adapt the huge
`sftpgo.json` file to your needs only to wonder "what did we change again?" the
next time SFTPGo is updated. Use an environment file (`/etc/sftpgo.env`) and
set only those configuration options you need to change. On Linux, SFTPGo's
systemd unit file automatically loads those environment variables on startup.
Make sure you understand how the environment overrides work (see "Environment
variables" on the ["Configuring SFTPGo"][2] page) if you need to add your own
settings.

Don't try to skimp on the proxy settings. If you are using nginx and you don't
use **all four** proxy settings we put in our `sftpgo.env` file, you'll drown
in weird form token errors when you try to log in.

[1]: <https://github.com/drakkan/sftpgo/blob/main/docs/service.md>
[2]: <https://github.com/drakkan/sftpgo/blob/main/docs/full-configuration.md>

## NCA Setup

First, set the URL appropriately to the API endpoint.  For our docker setup, we
expose SFTPGo internally docker-compose services at the URL
`http://sftpgo:8080`.  The API is just that host combined with the path
`/api/v2`, leaving us with this:

    SFTPGO_API_URL="http://sftpgo:8080/api/v2"

Next, create an admin user in SFTPGo and then decomission the default admin
("admin"), or at least alter the default user's password to be significantly
more secure than simply "password", and then update the credentials in NCA's
settings file.

Finally, choose a default quota for new users.  This ensures one publisher
can't blast hundreds of gigs (or even terabytes) of data, preventing all other
publishers from uploading anything.

## Usage

Once SFTPGo is integrated, any titles created in NCA will be sent to SFTPGo.
If you had been doing sftp the traditional way (local accounts using ssh with
the login shell disabled), you will find that a big advantage to SFTPGo is that
it doesn't need a local system administrator to manage users, quotas, etc.
Provisioning accounts will be automated from NCA, and management can be done
using the SFTPGo web API or the REST endpoints.

## Bulk loading

If you were already managing SFTP uploads and you don't want to redo all that
work, NCA can help!  On startup, NCA will scan the `titles` database table.
Any title that (a) is not SFTPGo-connected, and (b) has values for both the
`sftp_user` and `sftp_pass` fields will get sent to SFTPGo in the background.
These SFTP accounts will get the default quota applied, which can then be
edited on an individual basis from NCA.

**Note**: for security reasons, titles' passwords will be removed upon
successful integration with SFTPGo.  *(Storing plaintext passwords was supposed
to stop happening a long time ago)*

**Advanced users** can even populate the NCA `titles` table manually in order
to bulk-load SFTP users which weren't being managed by NCA.
