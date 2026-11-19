#!/bin/bash
#
# Makes everything in ./fakemount ours again, e.g., after a container wrote
# files there as a non-root user (under rootless podman, those are owned by
# one of our subordinate UIDs). Inside podman's user namespace, UID 0 is us.
if [[ -d ./fakemount ]]; then
  podman unshare chown -R 0:0 ./fakemount
  podman unshare find ./fakemount -type d -exec chmod 755 {} +
fi
