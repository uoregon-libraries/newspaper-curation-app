### Changed

- Before a batch is sent to ONI, its awardee is checked for existence, and the
  ONI Agent attempts to create it if needed. This fixes some confusion in
  documentation (no docs telling people how to add awardees to ONI, for
  instance), and fixes the automated batch loads from getting stuck when
  awardees don't exist on the ONI side.
