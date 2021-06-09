---
title: SFTPGo Integration
weight: 45
description: Setting up NCA for SFTPGo integration
---

[SFTPGo](https://github.com/drakkan/sftpgo) is an sftp server that exposes APIs
and a web interface for administration tasks.  We've chosen to integrate NCA
with SFTPGo in order to simplify the process of creating a new publisher.  If
you choose not to use it, publisher uploads will have to be managed entirely by
you (as was the case prior to this integration).

To disable SFTPGo integration, assign "-" to the `SFTPGO_API_URL` setting.

If you use SFTPGo, you'll need to use the SFTPGo documentation to set it up
however it makes sense for your system, and then set the URL appropriately to
the API endpoint.  For our docker setup, we expose SFTPGo internally
docker-compose services at the URL `http://sftpgo:8080`.  The API is just that
host combined with the path `/api/v2`, leaving us with this:

    SFTPGO_API_URL="http://sftpgo:8080/api/v2"

Set up an admin user in SFTPGo or at least alter the default user's password to
be significantly more secure than simply "password", and then update the
credentials in NCA's settings file.

Once SFTPGo is integrated, any titles created in NCA will be sent to SFTPGo.
If you had been doing sftp the traditional way (local accounts using ssh with
the login shell disabled), you will find that a big advantage to SFTPGo is that
it doesn't need a local system administrator to manage users, quotas, etc.
Provisioning accounts will be automated from NCA, and management can be done
using the SFTPGo web API or the REST endpoints.
