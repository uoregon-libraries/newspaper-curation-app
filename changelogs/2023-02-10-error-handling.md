## vX.Y.Z

Massive change to error handling and code resilience in general

### Fixed

- Two horrible panic calls from deep within NCA's bowels have been replaced
  with error returns, which should dramatically reduce the risk of a runtime
  crash. These were already rare, but now *should* be nonexistent.
- Various minor style fixes

### Added

- Error handling:
  - Many functions were returning errors which were silently being ignored, and
    are now properly being looked at (or at least explicitly ignored where the
    error truely didn't matter)
  - Hard crashes should no longer be possible in the web server (the `nca-http`
    service) even if we have missed some error handling or there's some crazy
    crash we haven't yet found.
  - Hard crashes from workers (the `nca-workers` service) should no longer be
    possible, though there are still some paths which can't be made 100%
    foolproof. Even so, if there *are* areas that can still crash, they should
    be exceptionally unlikely.

### Changed

- Replaced `golint` with `revive`. `golint` is deprecated, and `revive` offers
  a lot more rules to catch potential errors and/or style issues
- Replaced "interface{}" with "any" for readability now that we're on Go 1.18

### Removed

- Various bits of dead code have been removed. Some were technically part of
  the public API, but NCA's code isn't meant to be imported elsewhere, so if
  this breaks anything I will be absolutely FLABBERGASTED. Yeah, I said
  "flabbergasted". Deal with it.
