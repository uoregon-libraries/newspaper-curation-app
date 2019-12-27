# NCA Changelog

All notable changes to NCA will be documented in this file.

Starting from NCA v2.10.0, a changelog shall be kept based loosely on the
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.  Since this is
an internal project for UO, we won't be attempting to do much in the way of
adding helpful migration notes, deprecating features, etc.  This is basically
to help our team keep up with what's going on.  Ye be warned.  Again.

<!-- Template

## vX.Y.Z

Brief description, if necessary

### Fixed

### Added

### Changed

### Removed
-->

<!-- Unreleased: finalize and uncomment this in the release/* branch

## (Unreleased)

Brief description, if necessary

### Fixed

- Uploads list is now a table, making it a lot easier to read, and fixing the
  odd flow problems for titles with long names

### Added

### Changed

### Removed

-->

## v2.10.0

The big change here is that we don't force all titles to claim they're in
English when we generate ALTO.  Yay!

### Fixed

- JP2 output should be readable by other applications (file mode was 0600
  previously, making things like RAIS unable to even read the JP2s without a
  manual `chmod`)
- The check for a PDF's embedded image DPI is now much more reliable, and has
  unit testing

### Added

- Multi-language support added:
  - We read the 3-letter language code from the MARC XML and use that in the
    Alto output, rather than hard-coding ALTO to claim all blocks are English
  - Please keep in mind: this doesn't deal with the situation where a title can
    be in multiple languages - we use the *last* language we find in the MARC
    XML, because our internal process still has no way to try and differentiate
    languages on a per-text-block basis.
