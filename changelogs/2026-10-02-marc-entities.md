## vX.Y.Z

### Fixed

- MARC XML values are now decoded properly, so characters like apostrophes and
  ampersands no longer show up as HTML entities (e.g., `&#39;`) in a title's
  MARC title, MARC location, or name.
