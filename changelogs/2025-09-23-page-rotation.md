### Fixed

- When pre-processing PDFs, pages should no longer occasionally auto-rotate. We
  noticed this issue very rarely, so it wasn't something we even realized NCA
  was doing, and we're still not entirely sure why. GhostScript simply has been
  deciding to rotate some pages. Well... *no more*! We hope.
