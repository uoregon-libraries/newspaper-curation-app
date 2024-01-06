---
title: SFTPGo Integration
weight: 45
description: Setting up NCA for SFTPGo integration
---

[SFTPGo](https://github.com/drakkan/sftpgo) is an sftp server that exposes APIs
and a web interface for administration tasks. We've chosen to integrate NCA
with SFTPGo in order to simplify the process of creating titles for a publisher
that's uploading newspaper PDFs.

If you choose not to use this integration, publisher uploads will have to be
managed entirely by you (as was the case prior to this integration), and NCA
will not track SFTP data (which is a change from NCA 3.x and prior).

It is *highly recommended* that you use SFTPGo, as future versions of NCA may
require it, or at least make it very difficult to do without.

## Opt out

To disable SFTPGo integration, assign "-" to the `SFTPGO_API_URL` setting:

    SFTPGO_API_URL="-"

This ensures that NCA will not try to connect to a nonexistent server.

## SFTPGo Setup

You'll need to use the SFTPGo documentation to set it up however it makes sense
for your system. Once that's done, you have to then make NCA aware of it.

Our configuration files can be found in the NCA project's root under the
`sftpgo` directory. It's obviously somewhat specific to our bare-metal RHEL
system, but they should help get you up and running.

We use nginx to proxy the SFTPGo web interface. This isn't necessary -- SFTPGo
can be exposed directly. We prefer the granular control nginx gives us (e.g.,
being able to lock down certain paths by IP).

For the sftp daemon itself, we just expose that directly on port 22, and set up
sshd on port 2022, locked down to a handful of IP addresses.

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

If you're using docker, an admin user (login: `admin`, password: `password`) is
created for you automatically, but is extremely insecure: you must change the
default admin user's password at a minimum, or better yet use a docker override
to set up an admin that has a unique name *and* a good password.

If you aren't using docker, you will need an admin user. The first time you
connect to the SFTPGo web UI, you will be required to create an admin account,
though you can also use the `create_default_admin` SFTPGo setting to
automatically create one when sftpgo first runs. See the "Configuration file"
section of the SFTPGo configuration documentation for details.

[1]: <https://github.com/drakkan/sftpgo/blob/main/docs/service.md>
[2]: <https://github.com/drakkan/sftpgo/blob/main/docs/full-configuration.md>

## NCA Setup

Open up your settings file and jump to the SFTPGo section. If you're upgrading
from an older version of NCA, you will have to copy this section from the
example settings file into your production settings.

You'll need to tell NCA how to connect to SFTPGo: set the API URL and choose a
default quota for new users.

The API endpoint is simply the SFTPGo host combined with the path `/api/v2`.
For our docker setup, the internal service is `http://sftpgo:8080`, so our API
configuration looks like this:

    SFTPGO_API_URL="http://sftpgo:8080/api/v2"

The default quota is five gigabytes, but you can adjust this as needed. You
will likely want *something*, however: this ensures one publisher can't blast
hundreds of gigs (or even terabytes) of data, taking your server down and
preventing all other publishers from uploading anything.

### SFTPGo API Key

NCA connects to SFTPGo via an API key for a user with admin privileges. If you
are using Docker with defaults for development, this API key is created and
assigned in your NCA `settings` file automatically. For any non-docker use, one
will have to use the Bash scripts in the `sftpgo/` directory. You will need to
provide the environment variable `SETTINGS_PATH` with a value corresponding to
the path to your NCA `settings` file to these Bash scripts. An example
invocation of the key-fetcher might look like this:

    SETTINGS_PATH=/path/to/settings sftpgo/get_admin_api_key.sh

That is the only script that *must* be run. It will prompt for admin user
credentials, use them to request an API key for that user, store the key in the
NCA `settings` file automatically, and then call the accompanying script
`test_api_key.sh` to verify the key was fetched and set up properly.

Normally you only run `get_admin_api_key.sh` once, but if you need to reacquire
a key, you must reset your `settings` file (set
`SFTPGO_ADMIN_API_KEY=!sftpgo_admin_api_key!`) and then re-run this script.

Note that you can instead run `get_admin_api_key.sh` with the `--force` flag to
overwrite an existing key in your `settings` file. This should rarely be
necessary in a production environment, but can be useful on a development or
staging system where the stack may be destroyed and rebuilt regularly.

Other accompanying scripts are provided to list all API keys and delete an API
key.

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
