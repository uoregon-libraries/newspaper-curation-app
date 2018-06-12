NCA In Docker
===============

Set up your URLs so the services know how to route the web requests, how to set
up the IIIF server, how to find the live issues, etc.

```bash
  export APP_URL="https://jechols.uoregon.edu"
  export NCA_NEWS_WEBROOT="https://oregonnews.uoregon.edu"
```

Consider copying `docker-compose.override.yml-example` to
`docker-compose.override.yml` and `env-example` to `.env` as both example files
contain decent defaults for a development setup.  (*You'll obviously need to
change `.env` after copying it*)

The application will be available at the configured `$APP_URL`.
