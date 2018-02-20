Black Mamba In Docker
===============

Set up your URLs so the services know how to route the web requests, how to set
up the IIIF server, how to find the live issues, etc.

```bash
  export APP_URL="https://jechols.uoregon.edu"
  export NCA_NEWS_WEBROOT="https://oregonnews.uoregon.edu"
```

Consider copying `docker-compose.override.yml-example` to customize your setup
to be more development-friendly.  You could also put in your own values for the
various environment variables so you didn't have to do the "export..." dance
every time you need to start the system up from a new environment.

The application will be available at `$APP_URL/black-mamba`.  As there's
currently no homepage (or probably is by the time anybody is reading this - I
probably just haven't updated docs yet), you have to go to `/sftp` or another
handler path directly; e.g., `https://jechols.uoregon.edu/sftp`.
