### Fixed

- Dupe checking against issues in NCA's database is now real-time. This fixes
  the very rare (but very horrible to fix as we just learned) situation where
  two issues have the same metadata entered at roughly the same time, and the
  curators receive no warnings about the duplication.

### Changed

- Various low-level code changes to improve error handling and issue management
