### Changed

- The docker stack has been rewritten pretty much from scratch so that it's
  more like a production setup and includes *all* dependencies now, including
  ONI and ONI Agent for NCA to send its various commands.

### Migration

- Devs: you may need significant changes to your docker setup! There is *no*
  direct migration path.
  - There are now two example compose overrides, and they are are a lot more
    "out of the box" than previously. Copy the one that makes the most sense
    and you should only need a few tweaks.
  - Review your settings file if you're doing hybrid dev. With all the ONI
    services now in containers, various setting will need to change and some
    might need to change depending on how you're set up. It's best to do a full
    review, but below are some of the key settings:
    - `STAGING_AGENT` and `PRODUCTION_AGENT` will definitely need to change to
      point to the correct agent. One of those settings may be fine, but the
      two can't be the same anymore, so one has to change.
    - `STAGING_NEWS_WEBROOT` should change so you can easily verify NCA is
      loading batches into the right environment when they're QC-ready.
    - `BIND_ADDRESS` may need to change if it conflicts with any of the new
      services (or else you'll need to change those services in the compose
      override file)
    - `SFTP_API_URL` will probably have to change, or you'll need to change the
      compose override file, as its port is 8081 in the example overrides.
  - You should probably remove all existing docker images, volumes, containers,
    etc. You should delete your local issue cache if you mounted one.
  - A lot of environment variables are gone now, such as `APP_URL`. If you used
    `.env` you probably need to translate it based on the new setup, using the
    override of your choice as a starting point. There's a good chance you
    won't need to do anything, though, as most docker setup is a lot easier now
    that there are two examples to choose from.
