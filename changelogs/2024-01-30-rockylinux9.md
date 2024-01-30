### Changed

- The docker build is now based on RockyLinux 9 to match UO's production setup
  more closely. We still don't recommended docker for development or
  production, but it's helpful to test things quickly and validate a new
  environment (in this case RockyLinux).
  - The build process is now *significantly* simpler. You can see the image
    definition yourself in `docker/Dockerfile-app`, but the gist is that we no
    longer compile a patched version of poppler, nor install openjpeg2 tools
    from source.
- The docker build forcibly overwrites the settings file's SFTPGo API key on
  *every run*. This eases dev / testing in some situations, but again makes it
  a bad idea to use docker in production.
- The docker override example file is a little smarter: "bin" is not mounted
  inside the container, as that is the cause of many a headache; and
  "/mnt/news" is not assumed to exist on the host
